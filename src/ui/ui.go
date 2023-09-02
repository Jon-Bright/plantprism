package ui

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/plant"
	"github.com/gin-gonic/gin"
)

var (
	log       *logs.Loggers
	publisher device.Publisher
)

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
	d, err := device.Get(id, publisher)
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
			slot := d.Slots[se.Layer][se.Slot]
			slotID := string(se.Layer) + strconv.Itoa(int(se.Slot))
			planted := (slot.Plant != 0)
			if planted {
				p, err := plant.Get(slot.Plant)
				if err != nil {
					log.Error.Printf("couldn't get plant for ID '%v': %v", slot.Plant, err)
					return false
				}
				c.SSEvent("se", gin.H{
					"Slot":         slotID,
					"Planted":      true,
					"PlantName":    p.Names["de"], // TODO: language
					"PlantingTime": slot.PlantingTime.Unix(),
					"HarvestFrom":  slot.HarvestFrom.Unix(),
					"HarvestBy":    slot.HarvestBy.Unix(),
				})
			} else {
				c.SSEvent("se", gin.H{
					"Slot":    slotID,
					"Planted": false,
				})
			}
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
	err = d.AddPlant(slot, plant.PlantID(plantID))
	if err != nil {
		log.Warn.Printf("addPlant slot '%s', plantType '%s' failed: %v", slot, plantIDStr, err)
		c.String(http.StatusInternalServerError, "AddPlant failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func harvestPlantHandler(c *gin.Context) {
	d := getDevice(c, false, "HarvestPlant")
	if d == nil {
		// Error, already handled
		return
	}
	slot, set := c.GetPostForm("slot")
	if !set {
		log.Warn.Printf("harvestPlant request with no slot received")
		c.String(http.StatusBadRequest, "No slot specified")
		return
	}
	err := d.HarvestPlant(slot)
	if err != nil {
		log.Warn.Printf("harvestPlant slot '%s' failed: %v", slot, err)
		c.String(http.StatusInternalServerError, "HarvestPlant failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func setSunriseHandler(c *gin.Context) {
	d := getDevice(c, false, "SetSunrise")
	if d == nil {
		// Error, already handled
		return
	}
	sunrise, set := c.GetPostForm("seconds")
	if !set {
		log.Warn.Printf("setSunrise request with no seconds received")
		c.String(http.StatusBadRequest, "No seconds specified")
		return
	}
	sunriseI, err := strconv.Atoi(sunrise)
	if err != nil {
		log.Warn.Printf("setSunrise seconds '%s' not numeric: %v", sunrise, err)
		c.String(http.StatusBadRequest, "Invalid seconds specified")
		return
	}
	err = d.SetSunrise(time.Duration(sunriseI) * time.Second)
	if err != nil {
		log.Warn.Printf("setSunrise seconds '%s' failed: %v", sunrise, err)
		c.String(http.StatusInternalServerError, "SetSunrise failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func defaultModeHandler(c *gin.Context) {
	d := getDevice(c, false, "DefaultMode")
	if d == nil {
		// Error, already handled
		return
	}
	err := d.SetMode(device.ModeDefault)
	if err != nil {
		log.Warn.Printf("defaultMode failed: %v", err)
		c.String(http.StatusInternalServerError, "DefaultMode failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func silentModeHandler(c *gin.Context) {
	d := getDevice(c, false, "SilentMode")
	if d == nil {
		// Error, already handled
		return
	}
	err := d.SetMode(device.ModeSilent)
	if err != nil {
		log.Warn.Printf("silentMode failed: %v", err)
		c.String(http.StatusInternalServerError, "SilentMode failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func cinemaModeHandler(c *gin.Context) {
	d := getDevice(c, false, "CinemaMode")
	if d == nil {
		// Error, already handled
		return
	}
	err := d.SetMode(device.ModeCinema)
	if err != nil {
		log.Warn.Printf("cinemaMode failed: %v", err)
		c.String(http.StatusInternalServerError, "CinemaMode failed")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func Init(l *logs.Loggers, p device.Publisher) {
	log = l
	publisher = p
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.LoadHTMLGlob("resources/*.templ.html")
	r.Static("/static", "resources/static")
	r.GET("/", indexHandler)
	r.GET("/plantdb.json", plantDBHandler)
	r.GET("/stream", streamHandler)
	r.POST("/addPlant", addPlantHandler)
	r.POST("/harvestPlant", harvestPlantHandler)
	r.POST("/defaultMode", defaultModeHandler)
	r.POST("/silentMode", silentModeHandler)
	r.POST("/cinemaMode", cinemaModeHandler)
	go func() {
		err := r.Run(":3000")
		log.Critical.Fatalf("gin Run() returned, error %v", err)
	}()
}
