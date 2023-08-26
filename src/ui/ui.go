package ui

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/plant"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

var (
	log  *logs.Loggers
	mqtt paho.Client
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
				plant, err := plant.Get(slot.Plant)
				if err != nil {
					log.Error.Printf("couldn't find plant %d for slot %s", slot.Plant, slotID)
					c.String(http.StatusInternalServerError, "Unable to find plant for slot")
					return
				}
				vd.Slots[slotID] = SlotData{
					Slot:         slotID,
					Planted:      true,
					PlantName:    plant.Names["de"], //TODO: language
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
	c.JSON(http.StatusOK, plant.GetDB())
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

func addPlantHandler(c *gin.Context) {
	d := getDevice(c, false, "AddPlant")
	if d == nil {
		// Error, already handled
		return
	}
	slot, set := c.GetPostForm("slot")
	if !set {
		log.Warn.Printf("addPlant request with no slot received")
		c.String(http.StatusBadRequest, "No slot specified")
		return
	}
	plantIDStr, set := c.GetPostForm("plantType")
	if !set {
		log.Warn.Printf("addPlant request with no plantType received")
		c.String(http.StatusBadRequest, "No plantType specified")
		return
	}
	plantID, err := strconv.Atoi(plantIDStr)
	if err != nil {
		log.Warn.Printf("addPlant, plantType '%s' not convertible to integer: %v", plantIDStr, err)
		c.String(http.StatusBadRequest, "Invalid plantType specified")
		return
	}
	err = d.AddPlant(slot, plant.PlantID(plantID), time.Now())
	if err != nil {
		log.Warn.Printf("addPlant slot '%s', plantType '%s' failed: %v", slot, plantIDStr, err)
		c.String(http.StatusInternalServerError, "AddPlant failed")
	}
	c.JSON(http.StatusNoContent, nil)
}

func Init(l *logs.Loggers) {
	log = l
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.LoadHTMLGlob("resources/*.templ.html")
	r.Static("/static", "resources/static")
	r.GET("/", indexHandler)
	r.GET("/plantdb.json", plantDBHandler)
	r.GET("/stream", streamHandler)
	r.POST("/addPlant", addPlantHandler)
	go func() {
		err := r.Run(":3000")
		log.Critical.Fatalf("gin Run() returned, error %v", err)
	}()
}
