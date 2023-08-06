package device

import (
	"errors"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
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
	id         string
	msgQueue   chan *msgUnparsed
	mqttClient paho.Client

	clientToken string

	// Reported values from Agl update messages
	connected bool
	ec        int

	// Monotonically increasing ID sent out with update messages
	awsVersion int

	// Reported values from AWS update messages. These all need
	// timestamps, for providing in published messages.
	cooling       bool
	coolingT      time.Time
	door          bool
	doorT         time.Time
	firmwareNCU   int
	firmwareNCUT  time.Time
	humidA        int
	humidAT       time.Time
	humidB        int
	humidBT       time.Time
	lightA        bool
	lightAT       time.Time
	lightB        bool
	lightBT       time.Time
	recipeID      int
	recipeIDT     time.Time
	tankLevel     int
	tankLevelT    time.Time
	tankLevelRaw  int
	tankLevelRawT time.Time
	tempA         float64
	tempAT        time.Time
	tempB         float64
	tempBT        time.Time
	tempTank      float64
	tempTankT     time.Time
	totalOffset   int
	totalOffsetT  time.Time
	valve         ValveState
	valveT        time.Time
	wifiLevel     int
	wifiLevelT    time.Time
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
	if msg.prefix == "agl/all" && msg.event == "shadow/get" {
		err = d.processAglShadowGet(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "events/software/info/put" {
		err = d.processAglEventInfo(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "events/software/warning/put" {
		err = d.processAglEventWarning(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "mode" {
		err = d.processAglMode(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "recipe/get" {
		err = d.processAglRecipeGet(msg)
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

// Example: {"version":7, "format": "binary" }
type msgAglRecipeGet struct {
	Version *int
	Format  *string
}

func (d *Device) processAglRecipeGet(msg *msgUnparsed) error {
	m, err := parseAglRecipeGet(msg)
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
	r := m.State.Reported
	if r.Connected != nil {
		d.connected = *r.Connected
	}
	if r.EC != nil {
		d.ec = *r.EC
	}
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
	Cooling      *bool       `json:"cooling,omitempty"`
	Door         *bool       `json:"door,omitempty"`
	FirmwareNCU  *int        `json:"firmware_ncu,omitempty"`
	HumidA       *int        `json:"humid_a,omitempty"`
	HumidB       *int        `json:"humid_b,omitempty"`
	LightA       *bool       `json:"light_a,omitempty"`
	LightB       *bool       `json:"light_b,omitempty"`
	RecipeID     *int        `json:"recipe_id,omitempty"`
	TankLevel    *int        `json:"tank_level,omitempty"`
	TankLevelRaw *int        `json:"tank_level_raw,omitempty"`
	TempA        *float64    `json:"temp_a,omitempty"`
	TempB        *float64    `json:"temp_b,omitempty"`
	TempTank     *float64    `json:"temp_tank,omitempty"`
	TotalOffset  *int        `json:"total_offset,omitempty"`
	Valve        *ValveState `json:"valve,omitempty"`
	WifiLevel    *int        `json:"wifi_level,omitempty"`
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
	if *m.ClientToken != d.clientToken {
		return fmt.Errorf("clientToken '%s' received, but device clienToken is '%s'", *m.ClientToken, d.clientToken)
	}
	t := time.Now()
	r := m.State.Reported
	if r.Cooling != nil {
		d.cooling = *r.Cooling
		d.coolingT = t
	}
	if r.Door != nil {
		d.door = *r.Door
		d.doorT = t
	}
	if r.FirmwareNCU != nil {
		d.firmwareNCU = *r.FirmwareNCU
		d.firmwareNCUT = t
	}
	if r.HumidA != nil {
		d.humidA = *r.HumidA
		d.humidAT = t
	}
	if r.HumidB != nil {
		d.humidB = *r.HumidB
		d.humidBT = t
	}
	if r.LightA != nil {
		d.lightA = *r.LightA
		d.lightAT = t
	}
	if r.LightB != nil {
		d.lightB = *r.LightB
		d.lightBT = t
	}
	if r.RecipeID != nil {
		d.recipeID = *r.RecipeID
		d.recipeIDT = t
	}
	if r.TankLevel != nil {
		d.tankLevel = *r.TankLevel
		d.tankLevelT = t
	}
	if r.TankLevelRaw != nil {
		d.tankLevelRaw = *r.TankLevelRaw
		d.tankLevelRawT = t
	}
	if r.TempA != nil {
		d.tempA = *r.TempA
		d.tempAT = t
	}
	if r.TempB != nil {
		d.tempB = *r.TempB
		d.tempBT = t
	}
	if r.TempTank != nil {
		d.tempTank = *r.TempTank
		d.tempTankT = t
	}
	if r.TotalOffset != nil {
		d.totalOffset = *r.TotalOffset
		d.totalOffsetT = t
	}
	if r.Valve != nil {
		d.valve = *r.Valve
		d.valveT = t
	}
	if r.WifiLevel != nil {
		d.wifiLevel = *r.WifiLevel
		d.wifiLevelT = t
	}
	err = d.sendAWSUpdateAccepted(t)
	if err != nil {
		return err
	}
	return nil
}

// TODO: this definition should be somewhere else
type msgUpdTS struct {
	Timestamp int
}

type msgAWSShadowUpdateAcceptedMetadataReported struct {
	Cooling      *msgUpdTS `json:"cooling,omitempty"`
	Door         *msgUpdTS `json:"door,omitempty"`
	FirmwareNCU  *msgUpdTS `json:"firmware_ncu,omitempty"`
	HumidA       *msgUpdTS `json:"humid_a,omitempty"`
	HumidB       *msgUpdTS `json:"humid_b,omitempty"`
	LightA       *msgUpdTS `json:"light_a,omitempty"`
	LightB       *msgUpdTS `json:"light_b,omitempty"`
	RecipeID     *msgUpdTS `json:"recipe_id,omitempty"`
	TankLevel    *msgUpdTS `json:"tank_level,omitempty"`
	TankLevelRaw *msgUpdTS `json:"tank_level_raw,omitempty"`
	TempA        *msgUpdTS `json:"temp_a,omitempty"`
	TempB        *msgUpdTS `json:"temp_b,omitempty"`
	TempTank     *msgUpdTS `json:"temp_tank,omitempty"`
	TotalOffset  *msgUpdTS `json:"total_offset,omitempty"`
	Valve        *msgUpdTS `json:"valve,omitempty"`
	WifiLevel    *msgUpdTS `json:"wifi_level,omitempty"`
}
type msgAWSShadowUpdateAcceptedMetadata struct {
	Reported msgAWSShadowUpdateAcceptedMetadataReported
}
type msgAWSShadowUpdateAccepted struct {
	State       msgAWSShadowUpdateState
	Metadata    msgAWSShadowUpdateAcceptedMetadata
	Version     int
	Timestamp   int
	ClientToken *string
}

func (d *Device) sendAWSUpdateAccepted(t time.Time) error {
	return nil
}
