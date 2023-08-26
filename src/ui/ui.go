package ui

import (
	"net/http"

	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/plant"
	"github.com/gin-gonic/gin"
)

var (
	log     *logs.Loggers
	plantDB []plant.Plant
)

func handler(c *gin.Context) {
	type SlotData struct {
		Slot         string
		Planted      bool
		PlantName    string
		PlantingTime int
		HarvestFrom  int
		HarvestBy    int
	}
	type ViewData struct {
		Slots map[string]SlotData
	}
	// TODO: Get from Device
	vd := ViewData{
		Slots: map[string]SlotData{
			"a1": SlotData{
				Slot:         "a1",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a2": SlotData{
				Slot:         "a2",
				Planted:      true,
				PlantName:    "Basilikum",
				PlantingTime: 1692456791,
				HarvestFrom:  1694876035,
				HarvestBy:    1696690455,
			},
			"a3": SlotData{
				Slot:         "a3",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a4": SlotData{
				Slot:         "a4",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a5": SlotData{
				Slot:         "a5",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a6": SlotData{
				Slot:         "a6",
				Planted:      true,
				PlantName:    "Brunnenkresse",
				PlantingTime: 1690642676,
				HarvestFrom:  1692453502,
				HarvestBy:    1693061925,
			},
			"a7": SlotData{
				Slot:         "a7",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a8": SlotData{
				Slot:         "a8",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a9": SlotData{
				Slot:         "a9",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b1": SlotData{
				Slot:         "b1",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b2": SlotData{
				Slot:         "b2",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b3": SlotData{
				Slot:         "b3",
				Planted:      true,
				PlantName:    "Basilikum",
				PlantingTime: 1692456791,
				HarvestFrom:  1694876035,
				HarvestBy:    1696690455,
			},
			"b4": SlotData{
				Slot:         "b4",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b5": SlotData{
				Slot:         "b5",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b6": SlotData{
				Slot:         "b6",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b7": SlotData{
				Slot:         "b7",
				Planted:      true,
				PlantName:    "Brunnenkresse",
				PlantingTime: 1690642676,
				HarvestFrom:  1692453502,
				HarvestBy:    1693061925,
			},
			"b8": SlotData{
				Slot:         "b8",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b9": SlotData{
				Slot:         "b9",
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
		},
	}

	c.HTML(http.StatusOK, "index.templ.html", vd)
}

func plantDBHandler(c *gin.Context) {
	c.JSON(http.StatusOK, plantDB)
}

func streamHandler(c *gin.Context) {
	// TODO
}

func Init(l *logs.Loggers, pdb []plant.Plant) {
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
