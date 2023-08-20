package ui

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func handler(c *gin.Context) {
	type SlotData struct {
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
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a2": SlotData{
				Planted:      true,
				PlantName:    "Basilikum",
				PlantingTime: 1692456791,
				HarvestFrom:  1694876035,
				HarvestBy:    1696690455,
			},
			"a3": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a4": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a5": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a6": SlotData{
				Planted:      true,
				PlantName:    "Brunnenkresse",
				PlantingTime: 1690642676,
				HarvestFrom:  1692453502,
				HarvestBy:    1693061925,
			},
			"a7": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a8": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"a9": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b1": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b2": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b3": SlotData{
				Planted:      true,
				PlantName:    "Basilikum",
				PlantingTime: 1692456791,
				HarvestFrom:  1694876035,
				HarvestBy:    1696690455,
			},
			"b4": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b5": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b6": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b7": SlotData{
				Planted:      true,
				PlantName:    "Brunnenkresse",
				PlantingTime: 1690642676,
				HarvestFrom:  1692453502,
				HarvestBy:    1693061925,
			},
			"b8": SlotData{
				Planted:      false,
				PlantName:    "",
				PlantingTime: 0,
				HarvestFrom:  0,
				HarvestBy:    0,
			},
			"b9": SlotData{
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

func Init() error {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.LoadHTMLGlob("resources/*.templ.html")
	r.Static("/static", "resources/static")
	r.GET("/", handler)
	return r.Run(":3000")
}
