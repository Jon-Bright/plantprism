package device

import (
	"fmt"
	"time"
)

// Example: {"reported":{"timezone":"Europe/Berlin","user_offset":25200,"total_offset":68400,"mode":0,"stage":"prod","verbose_reporting":false,"recipe_id":1687013771,"firmware_ncu":1667466618,"firmware_mcu":1667466618}}
type msgAglShadowGetAcceptedReported struct {
	Timezone         string     `json:"timezone"`
	UserOffset       int        `json:"user_offset"`
	TotalOffset      int        `json:"total_offset"`
	Mode             DeviceMode `json:"mode"`
	Stage            string     `json:"stage"`
	VerboseReporting bool       `json:"verbose_reporting"`
	RecipeID         int        `json:"recipe_id"`
	FirmwareNCU      int        `json:"firmware_ncu"`
	FirmwareMCU      int        `json:"firmware_mcu"`
}
type msgAglShadowGetAccepted struct {
	Reported msgAglShadowGetAcceptedReported `json:"reported"`
}

func (m *msgAglShadowGetAccepted) topic() string {
	return MQTT_TOPIC_AGL_GET_ACCEPTED
}

func (d *Device) processAglShadowGet(msg *msgUnparsed) ([]msgReply, error) {
	// No parsing: the only time we see this, it has no content
	m, err := d.getAglShadowGetReply()
	if err != nil {
		return nil, err
	}
	return []msgReply{m}, nil
}

func (d *Device) getAglShadowGetReply() (msgReply, error) {
	if d.reported.recipeID.v <= 1 {
		return nil, fmt.Errorf("wanted to send Agl shadow get reply, but recipe ID is %d, time %v", d.reported.recipeID.v, d.reported.recipeID.t)
	}
	if d.timezone == "" {
		return nil, fmt.Errorf("wanted to send Agl shadow get reply, but timezone is empty")
	}
	msg := msgAglShadowGetAccepted{}
	r := &msg.Reported
	r.Timezone = d.timezone
	r.UserOffset = int(sunriseD.Seconds()) // user_offset doesn't actually get used by the Plantcube
	var err error
	r.TotalOffset, err = calcTotalOffset(d.timezone, time.Now(), sunriseD)
	if err != nil {
		return nil, fmt.Errorf("total offset calculation failed: %w", err)
	}
	log.Info.Printf("totalOffset %d sec", r.TotalOffset)
	r.Mode = d.mode
	r.Stage = FIXED_STAGE
	r.VerboseReporting = FIXED_VERBOSE_REPORTING
	r.RecipeID = d.reported.recipeID.v
	r.FirmwareNCU = FIXED_FIRMWARE_NCU
	r.FirmwareMCU = FIXED_FIRMWARE_MCU
	return &msg, nil
}
