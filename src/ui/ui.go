package ui

import (
	"io"
	"net/http"
	"strconv"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/plant"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

var (
	log     *logs.Loggers
	plantDB map[plant.PlantID]plant.Plant
	mqtt    paho.Client
)

func SetPahoClient(c paho.Client) {
	mqtt = c
}

func getDevice(c *gin.Context, isGet bool, reqName string) *device.Device {
	var (
		id  string
		set bool
	)
	if isGet {
		id, set = c.GetQuery("id")
	} else {
		id, set = c.GetPostForm("id")
	}
	if !set {
		log.Warn.Printf("%s request with no Device ID received", reqName)
		c.String(http.StatusBadRequest, "No Device ID specified")
		return nil
	}
	d, err := device.Get(id, mqtt)
	if err != nil {
		log.Warn.Printf("%s request with invalid Device ID '%s': %v", reqName, id, err)
		c.String(http.StatusBadRequest, "Device ID '%s' invalid", id)
		return nil
	}
	return d
}

func indexHandler(c *gin.Context) {
	d := getDevice(c, true, "Index")
	if d == nil {
		// Error, already handled
		return
	}
	type SlotData struct {
		Slot         string
		Planted      bool
		PlantName    string
		PlantingTime int64
		HarvestFrom  int64
		HarvestBy    int64
	}
	type ViewData struct {
		DeviceID string
		Slots    map[string]SlotData
	}
	vd := ViewData{
		DeviceID: d.ID,
		Slots:    map[string]SlotData{},
	}
	for lid, layer := range d.Slots {
		for sid, slot := range layer {
			slotID := string(lid) + strconv.Itoa(int(sid))
			if slot.Plant != 0 {
				vd.Slots[slotID] = SlotData{
					Slot:         slotID,
					Planted:      true,
					PlantName:    plantDB[slot.Plant].Names["de"], //TODO: language
					PlantingTime: slot.PlantingTime.Unix(),
					HarvestFrom:  slot.HarvestFrom.Unix(),
					HarvestBy:    slot.HarvestBy.Unix(),
				}
			} else {
				vd.Slots[slotID] = SlotData{
					Slot:    slotID,
					Planted: false,
				}
			}
		}
	}

	c.HTML(http.StatusOK, "index.templ.html", vd)
}

func plantDBHandler(c *gin.Context) {
	c.JSON(http.StatusOK, plantDB)
}

func streamHandler(c *gin.Context) {
	d := getDevice(c, true, "Stream")
	if d == nil {
		// Error, already handled
		return
	}
	slotChan := d.GetSlotChan()
	defer func() {
		d.DropSlotChan(slotChan)
	}()
	c.Stream(func(w io.Writer) bool {
		select {
		case se := <-slotChan:
			c.SSEvent("se", gin.H{
				"SlotID": se.SlotID,
				// TODO: actual event details
			})
			return true
		}
		return false
	})
}

func Init(l *logs.Loggers, pdb map[plant.PlantID]plant.Plant) {
	log = l
	plantDB = pdb
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.LoadHTMLGlob("resources/*.templ.html")
	r.Static("/static", "resources/static")
	r.GET("/", indexHandler)
	r.GET("/plantdb.json", plantDBHandler)
	r.GET("/stream", streamHandler)
	go func() {
		err := r.Run(":3000")
		log.Critical.Fatalf("gin Run() returned, error %v", err)
	}()
}
