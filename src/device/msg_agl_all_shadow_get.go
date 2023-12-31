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
	m, err := d.getAglShadowGetReply(msg.t)
	if err != nil {
		return nil, err
	}
	return []msgReply{m}, nil
}

func (d *Device) getAglShadowGetReply(t time.Time) (msgReply, error) {
	if d.Reported.RecipeID.Value <= 1 {
		return nil, fmt.Errorf("wanted to send Agl shadow get reply, but recipe ID is %d, time %v", d.Reported.RecipeID.Value, d.Reported.RecipeID.Time)
	}
	if d.Timezone == "" {
		return nil, fmt.Errorf("wanted to send Agl shadow get reply, but timezone is empty")
	}
	msg := msgAglShadowGetAccepted{}
	r := &msg.Reported
	r.Timezone = d.Timezone
	r.UserOffset = d.UserOffset // user_offset doesn't actually get used by the Plantcube
	var err error
	sunrise := time.Duration(d.UserOffset) * time.Second
	r.TotalOffset, err = calcTotalOffset(d.Timezone, t, sunrise)
	if err != nil {
		return nil, fmt.Errorf("total offset calculation failed: %w", err)
	}
	r.Mode = d.Reported.Mode.Value
	r.Stage = FIXED_STAGE
	r.VerboseReporting = FIXED_VERBOSE_REPORTING
	r.RecipeID = d.Reported.RecipeID.Value
	r.FirmwareNCU = FIXED_FIRMWARE_NCU
	r.FirmwareMCU = FIXED_FIRMWARE_MCU
	log.Info.Printf("Reporting timezone '%s', userOffset %ds, totalOffset %ds, mode %d, recipe ID %d", r.Timezone, r.UserOffset, r.TotalOffset, r.Mode, r.RecipeID)
	return &msg, nil
}
