package device

import (
	"errors"
	"fmt"
	"time"
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

type ValveState int

const (
	ValveOpenLayerB ValveState = 0
	ValveOpenLayerA ValveState = 1
	ValveClosed     ValveState = 4
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
	if msg.prefix == "agl/prod" && msg.event == "events/software/info/put" {
		err = d.processAglEventInfo(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "events/software/warning/put" {
		err = d.processAglEventWarning(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "mode" {
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

// Example: {"label":"MCU_MODE_STATE","timestamp":1687686053,"payload":{"mode":"ECO_MODE","state":"0","layer":"APPLIANCE"}}
type msgAglEventInfoPayload struct {
	Mode  *string
	State *string
	Layer *string
}

type msgAglEventInfo struct {
	Label     *string
	Timestamp *int
	Payload   msgAglEventInfoPayload
}

func (d *Device) processAglEventInfo(msg *msgUnparsed) error {
	m, err := parseAglEventInfo(msg)
	if err != nil {
		return err
	}
	log.Info.Printf("Plantcube info, time %s, label '%s', mode '%s', state '%s', layer '%s'", time.Unix(int64(*m.Timestamp), 0).String(), *m.Label, *m.Payload.Mode, *m.Payload.State, *m.Payload.Layer)

	return nil
}

// Example: {"label":"NCU_SYS_LOG","timestamp":1687329836,"payload":{"error_log":"MGOS_SHADOW_UPDATE_REJECTED 400 Missing required node: state
//
//	timer: 0; retries: 0; buff: {'clientToken':'5975bc44','state':{'reported':","function_name":"aws_shadow_grp_handler"}}
type msgAglEventWarningPayload struct {
	ErrorLog     *string `json:"error_log"`
	FunctionName *string `json:"function_name"`
}

type msgAglEventWarning struct {
	Label     *string
	Timestamp *int
	Payload   msgAglEventWarningPayload
}

func (d *Device) processAglEventWarning(msg *msgUnparsed) error {
	m, err := parseAglEventWarning(msg)
	if err != nil {
		return err
	}
	log.Warn.Printf("Plantcube warning, time %s, label '%s', function '%s', log '%s'", time.Unix(int64(*m.Timestamp), 0).String(), *m.Label, *m.Payload.FunctionName, *m.Payload.ErrorLog)

	return nil
}

// Example: {"prev_mode": 0,"mode": 8, "trigger": 1}
type msgAglMode struct {
	PrevMode *DeviceMode `json:"prev_mode"`
	Mode     *DeviceMode
	Trigger  *ModeTrigger
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

func (d *Device) processAglShadowUpdate(msg *msgUnparsed) error {
	m, err := parseAglShadowUpdate(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}

func (d *Device) processAglShadowGet(msg *msgUnparsed) error {
	// No parsing: the only time we see this, it has no content
	return nil
}

// Example: {"clientToken":"5975bc44"}
type msgAWSShadowGet struct {
	ClientToken *string
}

func (d *Device) processAWSShadowGet(msg *msgUnparsed) error {
	m, err := parseAWSShadowGet(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}

// Example: {"clientToken":"5975bc44","state":{"reported":{"humid_b":75,"temp_a":22.99,"temp_b":24.19}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_a":22.69,"firmware_ncu":1667466618,"door":false,"cooling":true,"total_offset":69299,"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"wifi_level":0}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_tank":28.34}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":true,"light_b":true}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"tank_level_raw":2}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"door":true}}}

type msgAWSShadowUpdateReported struct {
	Cooling      *bool
	Door         *bool
	FirmwareNCU  *int        `json:"firmware_ncu"`
	HumidA       *int        `json:"humid_a"`
	HumidB       *int        `json:"humid_b"`
	LightA       *bool       `json:"light_a"`
	LightB       *bool       `json:"light_b"`
	RecipeID     *int        `json:"recipe_id"`
	TankLevel    *int        `json:"tank_level"`
	TankLevelRaw *int        `json:"tank_level_raw"`
	TempA        *float64    `json:"temp_a"`
	TempB        *float64    `json:"temp_b"`
	TempTank     *float64    `json:"temp_tank"`
	TotalOffset  *int        `json:"total_offset"`
	Valve        *ValveState `json:"valve"`
	WifiLevel    *int        `json:"wifi_level"`
}
type msgAWSShadowUpdateState struct {
	Reported msgAWSShadowUpdateReported
}
type msgAWSShadowUpdate struct {
	ClientToken *string
	State       msgAWSShadowUpdateState
}

func (d *Device) processAWSShadowUpdate(msg *msgUnparsed) error {
	m, err := parseAWSShadowUpdate(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}
