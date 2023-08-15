package device

import (
	"errors"
	"fmt"
)

// Example: {"prev_mode": 0,"mode": 8, "trigger": 1}
type msgAglMode struct {
	PrevMode *DeviceMode `json:"prev_mode"`
	Mode     *DeviceMode
	Trigger  *ModeTrigger
}

func parseAglMode(msg *msgUnparsed) (*msgAglMode, error) {
	var m msgAglMode
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.PrevMode == nil {
		return nil, errors.New("no prev_mode field")
	} else if m.Mode == nil {
		return nil, errors.New("no mode field")
	} else if m.Trigger == nil {
		return nil, errors.New("no trigger field")
	} else if *m.PrevMode < ModeDefault || *m.PrevMode >= ModeOutOfRange {
		return nil, fmt.Errorf("PrevMode %d is invalid", *m.PrevMode)
	} else if *m.Mode < ModeDefault || *m.Mode >= ModeOutOfRange {
		return nil, fmt.Errorf("Mode %d is invalid", *m.Mode)
	} else if *m.Mode == *m.PrevMode {
		return nil, fmt.Errorf("Mode %d is the same as previously", *m.Mode)
	} else if *m.Trigger < ModeTriggerApp || *m.Trigger >= ModeTriggerOutOfRange {
		return nil, fmt.Errorf("Trigger %d is invalid", *m.Trigger)
	}

	return &m, nil
}

func (d *Device) processAglMode(msg *msgUnparsed) error {
	m, err := parseAglMode(msg)
	if err != nil {
		return err
	}
	log.Info.Printf("Device mode changed from %v to %v, trigger %v", *m.PrevMode, *m.Mode, *m.Trigger)
	d.Mode = *m.Mode
	// TODO : In response to some mode changes, we should display
	// stuff for the end user (e.g. during cleaning, tank pumping,
	// etc.)
	return nil
}
