package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/lupguo/go-render/render"
	"io"
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

type valueWithTimestamp[T any] struct {
	v T
	t time.Time
}

func (vwt valueWithTimestamp[T]) update(v T, t time.Time) {
	vwt.v = v
	vwt.t = t
}

func (vmt valueWithTimestamp[T]) wasUpdatedAt(t time.Time) bool {
	return vmt.t == t
}

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
	connected valueWithTimestamp[bool]
	ec        valueWithTimestamp[int]

	// Reported values from AWS update messages.
	cooling      valueWithTimestamp[bool]
	door         valueWithTimestamp[bool]
	firmwareNCU  valueWithTimestamp[int]
	humidA       valueWithTimestamp[int]
	humidB       valueWithTimestamp[int]
	lightA       valueWithTimestamp[bool]
	lightB       valueWithTimestamp[bool]
	recipeID     valueWithTimestamp[int]
	tankLevel    valueWithTimestamp[int]
	tankLevelRaw valueWithTimestamp[int]
	tempA        valueWithTimestamp[float64]
	tempB        valueWithTimestamp[float64]
	tempTank     valueWithTimestamp[float64]
	totalOffset  valueWithTimestamp[int]
	valve        valueWithTimestamp[ValveState]
	wifiLevel    valueWithTimestamp[int]
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

func pickyUnmarshal(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	err := d.Decode(v)
	if err != nil {
		return err
	}
	// The data should be one object and nothing more
	if t, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing data after decode: %T / %v, err %w", t, t, err)
	}
	return nil
}
