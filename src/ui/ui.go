package ui

import (
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

func handler(c *gin.Context) {
	id, set := c.GetQuery("id")
	if !set {
		c.String(http.StatusBadRequest, "No Device ID specified")
		return
	}
	d, err := device.Get(id, mqtt)
	if err != nil {
		c.String(http.StatusBadRequest, "Device ID '%s' invalid: %v", id, err)
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
		Slots map[string]SlotData
	}
	vd := ViewData{
		Slots: map[string]SlotData{},
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
	// TODO
}

func Init(l *logs.Loggers, pdb map[plant.PlantID]plant.Plant) {
	log = l
	plantDB = pdb
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.LoadHTMLGlob("resources/*.templ.html")
	r.Static("/static", "resources/static")
	r.GET("/", handler)
	r.GET("/plantdb.json", plantDBHandler)
	r.GET("/stream", streamHandler)
	go func() {
		err := r.Run(":3000")
		log.Critical.Fatalf("gin Run() returned, error %v", err)
	}()
}
