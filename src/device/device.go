package device

import (
	"encoding/json"
	"errors"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/lupguo/go-render/render"
	"strings"
	"time"
)

const (
	// These values are theoretically changeable over time, but
	// they're the values reported by the actual Agrilution
	// replies and we have no reason to change them, so they're
	// hard-coded.
	FIXED_STAGE             = "prod"
	FIXED_VERBOSE_REPORTING = false
	FIXED_FIRMWARE_NCU      = 1667466618
	FIXED_FIRMWARE_MCU      = 1667466618

	MQTT_PUBLISH_TIMEOUT           = 30 * time.Second
	MQTT_ID_TOKEN                  = "<ID>"
	MQTT_TOPIC_AGL_GET_ACCEPTED    = "agl/all/things/" + MQTT_ID_TOKEN + "/shadow/get/accepted"
	MQTT_TOPIC_AWS_UPDATE_ACCEPTED = "$aws/things/" + MQTT_ID_TOKEN + "/shadow/update/accepted"
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

	// Configuration
	timezone   string
	userOffset int // Seconds by which the day/night cycle is shifted
	mode       DeviceMode

	// Monotonically increasing ID sent out with update messages
	awsVersion int

	// Everything below is a reported value. They all need
	// timestamps for reporting back in update/accepted messages.

	// Reported values from Agl update messages
	connected  bool
	connectedT time.Time
	ec         int
	ecT        time.Time

	// Reported values from AWS update messages.
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

type msgUpdTS struct {
	Timestamp int `json:"timestamp"`
}

type msgReply interface {
	topic() string
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
	var replies []msgReply
	if msg.prefix == "agl/all" && msg.event == "shadow/get" {
		replies, err = d.processAglShadowGet(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "events/software/info/put" {
		err = d.processAglEventInfo(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "events/software/warning/put" {
		err = d.processAglEventWarning(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "mode" {
		err = d.processAglMode(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "recipe/get" {
		err = d.processAglRecipeGet(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "shadow/update" {
		replies, err = d.processAglShadowUpdate(msg)
	} else if msg.prefix == "$aws" && msg.event == "shadow/get" {
		err = d.processAWSShadowGet(msg)
	} else if msg.prefix == "$aws" && msg.event == "shadow/update" {
		replies, err = d.processAWSShadowUpdate(msg)
	} else {
		err = errors.New("no handler found")
	}
	if err != nil {
		return fmt.Errorf("failed parsing prefix '%s', event '%s': %w", msg.prefix, msg.event, err)
	}

	if replies != nil {
		err = d.sendReplies(replies)
		if err != nil {
			return fmt.Errorf("failed reply for prefix '%s', event '%s': %w", msg.prefix, msg.event, err)
		}
	}

	return nil
}

func (d *Device) sendReplies(replies []msgReply) error {
	for _, r := range replies {
		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("failed marshalling '%s': %w", render.Render(r), err)
		}
		topic := strings.ReplaceAll(r.topic(), MQTT_ID_TOKEN, d.id)
		token := d.mqttClient.Publish(topic, 0, false, b)
		if !token.WaitTimeout(MQTT_PUBLISH_TIMEOUT) {
			return errors.New("timeout publishing MQTT msg")
		}
		if token.Error() != nil {
			return fmt.Errorf("failed publishing MQTT message: %w", err)
		}
	}
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

func (d *Device) processAglShadowUpdate(msg *msgUnparsed) ([]msgReply, error) {
	m, err := parseAglShadowUpdate(msg)
	if err != nil {
		return nil, err
	}
	t := time.Now()
	r := m.State.Reported
	if r.Connected != nil {
		d.connected = *r.Connected
		d.connectedT = t
	}
	if r.EC != nil {
		d.ec = *r.EC
		d.ecT = t
	}
	reply := d.getAWSUpdateAcceptedReply(t, true)
	return []msgReply{reply}, nil
}

func calcTotalOffset(tz string, t time.Time, sunrise time.Duration) (int, error) {
	// The total_offset is one day minus sunrise _plus_ the timezone offset
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0, fmt.Errorf("unable to load zone '%s': %w", tz, err)
	}
	_, current_offset := t.In(loc).Zone()
	totalOffset := int((24*time.Hour - sunrise).Seconds()) + current_offset

	// Total offset isn't allowed to be >=86400 (the Plantcube
	// checks this). With a sunrise of 07:00, any timezone further
	// east than UTC+7 will produce a value>86400. I could only
	// check stuff in one timezone (Europe/Berlin), but did do a
	// bunch of tests to try different behaviours, including
	// setting the start of day to 23:30 and to 00:30. When
	// setting sunrise to anything later than 18:00 in the app,
	// it's clamped to 18:00, but times as early as 00:30 are
	// fine. This appears to be an app thing, though - the correct
	// settings go to the device.
	//
	// In any event, for the purpose of not exceeding 86400, a
	// plain mod appears to be adequate.
	totalOffset %= 86400

	return totalOffset, nil
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
	// TODO: Actually process this.
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
	// `Connected` and `EC` shouldn't be reported by $aws/.../update,
	// but they're here because they appear in .../update/accepted
	Connected    *bool       `json:"connected,omitempty"`
	Cooling      *bool       `json:"cooling,omitempty"`
	Door         *bool       `json:"door,omitempty"`
	EC           *int        `json:"ec,omitempty"`
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
	Reported msgAWSShadowUpdateReported `json:"reported"`
}
type msgAWSShadowUpdate struct {
	ClientToken *string
	State       msgAWSShadowUpdateState
}

func (d *Device) processAWSShadowUpdate(msg *msgUnparsed) ([]msgReply, error) {
	m, err := parseAWSShadowUpdate(msg)
	if err != nil {
		return nil, err
	}
	if *m.ClientToken != d.clientToken {
		return nil, fmt.Errorf("clientToken '%s' received, but device clientToken is '%s'", *m.ClientToken, d.clientToken)
	}
	t := time.Now()
	r := m.State.Reported
	if r.Connected != nil {
		return nil, errors.New("unexpected Connected reported in AWS update")
	}
	if r.EC != nil {
		return nil, errors.New("unexpected EC reported in AWS update")
	}
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
	reply := d.getAWSUpdateAcceptedReply(t, false)
	return []msgReply{reply}, nil
}

type msgAWSShadowUpdateAcceptedMetadataReported struct {
	Connected    *msgUpdTS `json:"connected,omitempty"`
	Cooling      *msgUpdTS `json:"cooling,omitempty"`
	Door         *msgUpdTS `json:"door,omitempty"`
	EC           *msgUpdTS `json:"ec,omitempty"`
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
	Reported msgAWSShadowUpdateAcceptedMetadataReported `json:"reported"`
}
type msgAWSShadowUpdateAccepted struct {
	State       msgAWSShadowUpdateState            `json:"state"`
	Metadata    msgAWSShadowUpdateAcceptedMetadata `json:"metadata"`
	Version     int                                `json:"version"`
	Timestamp   int                                `json:"timestamp"`
	ClientToken string                             `json:"clientToken,omitempty"`
}

func (m *msgAWSShadowUpdateAccepted) topic() string {
	return MQTT_TOPIC_AWS_UPDATE_ACCEPTED
}

// Construct a reply featuring all values reported at the given timestamp,
// along with metadata for each of those values with the timestamp.
// /shadow/update to agl/prod _also_ triggers AWS updates, but these come
// without a client token (possibly because from AWS's POV, they're coming
// from a different client?).
func (d *Device) getAWSUpdateAcceptedReply(t time.Time, omitClientToken bool) msgReply {
	msg := msgAWSShadowUpdateAccepted{}
	r := &msg.State.Reported
	m := &msg.Metadata.Reported
	unix := int(t.Unix())

	d.awsVersion++
	msg.Version = d.awsVersion
	msg.Timestamp = unix
	if !omitClientToken {
		msg.ClientToken = d.clientToken
	}
	ts := msgUpdTS{unix}

	if d.connectedT == t {
		r.Connected = &d.connected
		m.Connected = &ts
	}
	if d.coolingT == t {
		r.Cooling = &d.cooling
		m.Cooling = &ts
	}
	if d.doorT == t {
		r.Door = &d.door
		m.Door = &ts
	}
	if d.ecT == t {
		r.EC = &d.ec
		m.EC = &ts
	}
	if d.firmwareNCUT == t {
		r.FirmwareNCU = &d.firmwareNCU
		m.FirmwareNCU = &ts
	}
	if d.humidAT == t {
		r.HumidA = &d.humidA
		m.HumidA = &ts
	}
	if d.humidBT == t {
		r.HumidB = &d.humidB
		m.HumidB = &ts
	}
	if d.lightAT == t {
		r.LightA = &d.lightA
		m.LightA = &ts
	}
	if d.lightBT == t {
		r.LightB = &d.lightB
		m.LightB = &ts
	}
	if d.recipeIDT == t {
		r.RecipeID = &d.recipeID
		m.RecipeID = &ts
	}
	if d.tankLevelT == t {
		r.TankLevel = &d.tankLevel
		m.TankLevel = &ts
	}
	if d.tankLevelRawT == t {
		r.TankLevelRaw = &d.tankLevelRaw
		m.TankLevelRaw = &ts
	}
	if d.tempAT == t {
		r.TempA = &d.tempA
		m.TempA = &ts
	}
	if d.tempBT == t {
		r.TempB = &d.tempB
		m.TempB = &ts
	}
	if d.tempTankT == t {
		r.TempTank = &d.tempTank
		m.TempTank = &ts
	}
	if d.totalOffsetT == t {
		r.TotalOffset = &d.totalOffset
		m.TotalOffset = &ts
	}
	if d.valveT == t {
		r.Valve = &d.valve
		m.Valve = &ts
	}
	if d.wifiLevelT == t {
		r.WifiLevel = &d.wifiLevel
		m.WifiLevel = &ts
	}
	return &msg
}
