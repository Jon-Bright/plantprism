package device

import (
	"encoding/json"
	"errors"
	"fmt"
)

type DeviceMode int

const (
	ModeDefault           DeviceMode = 0
	ModeDebug             DeviceMode = 1
	ModeRinseEnd          DeviceMode = 2
	ModeTankDrainCleaning DeviceMode = 3
	ModeTankDrainExplicit DeviceMode = 4
	ModeCleaning          DeviceMode = 5
	ModeUnknown           DeviceMode = 6
	ModeSilent            DeviceMode = 7
	ModeCinema            DeviceMode = 8
	ModeOutOfRange        DeviceMode = 9
)

type ModeTrigger int

const (
	ModeTriggerApp        ModeTrigger = 0
	ModeTriggerDevice     ModeTrigger = 1
	ModeTriggerOutOfRange ModeTrigger = 2
)

type Device struct {
	id       string
	msgQueue chan *msgUnparsed
}

type msgUnparsed struct {
	prefix  string
	event   string
	content []byte
}

func (d *Device) ProcessMessage(prefix string, event string, content []byte) {
	d.msgQueue <- &msgUnparsed{prefix, event, content}
}

func (d *Device) processingLoop() {
	for {
		msg := <-d.msgQueue
		err := d.processMessage(msg)
		if err != nil {
			log.Error.Printf(err.Error())
		}
	}
}

func (d *Device) processMessage(msg *msgUnparsed) error {
	var err error
	if msg.prefix == "agl/prod" && msg.event == "mode" {
		err = d.processAglMode(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "shadow/get" {
		err = d.processAglShadowGet(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "shadow/update" {
		err = d.processAglShadowUpdate(msg)
	} else if msg.prefix == "$aws" && msg.event == "shadow/get" {
		err = d.processAWSShadowGet(msg)
	} else {
		err = errors.New("no handler found")
	}
	if err != nil {
		return fmt.Errorf("failed parsing prefix '%s', event '%s': %v", msg.prefix, msg.event, err)
	}
	return nil
}

// Example: {"prev_mode": 0,"mode": 8, "trigger": 1}
type msgAglMode struct {
	PrevMode *DeviceMode `json:"prev_mode"`
	Mode     *DeviceMode
	Trigger  *ModeTrigger
}

func parseAglMode(msg *msgUnparsed) (*msgAglMode, error) {
	var m msgAglMode
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.PrevMode == nil {
		return nil, errors.New("No prev_mode field")
	} else if m.Mode == nil {
		return nil, errors.New("No mode field")
	} else if m.Trigger == nil {
		return nil, errors.New("No trigger field")
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
	_ = m
	return nil
}

// Example: {"state":{"reported":{"connected": true}}}
// Example: {"state":{"reported":{"ec": 1306}}}
type msgAglShadowUpdateReported struct {
	Connected *bool
	EC        *int
}
type msgAglShadowUpdateState struct {
	Reported msgAglShadowUpdateReported
}
type msgAglShadowUpdate struct {
	State msgAglShadowUpdateState
}

func parseAglShadowUpdate(msg *msgUnparsed) (*msgAglShadowUpdate, error) {
	var m msgAglShadowUpdate
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.State.Reported.Connected == nil && m.State.Reported.EC == nil {
		return nil, errors.New("no fields set")
	}
	// Never seen them both set at the same time, but there's no
	// real reason they shouldn't be, so not checking that.
	return &m, nil
}

func (d *Device) processAglShadowUpdate(msg *msgUnparsed) error {
	m, err := parseAglShadowUpdate(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}

// No parsing: the only time we see this, it has no content
func (d *Device) processAglShadowGet(msg *msgUnparsed) error {
	return nil
}

// Example: {"clientToken":"5975bc44"}
type msgAWSShadowGet struct {
	ClientToken *string
}

func parseAWSShadowGet(msg *msgUnparsed) (*msgAWSShadowGet, error) {
	var m msgAWSShadowGet
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.ClientToken == nil {
		return nil, errors.New("no ClientToken")
	} else if len(*m.ClientToken) < 8 {
		return nil, fmt.Errorf("ClientToken '%s' too short", *m.ClientToken)
	}
	// Could theoretically check if it's hex, which the
	// Plantcube's all are, but do we care?
	return &m, nil
}

func (d *Device) processAWSShadowGet(msg *msgUnparsed) error {
	m, err := parseAWSShadowGet(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}
