package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jon-Bright/plantprism/plant"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/lupguo/go-render/render"
	"io"
	"os"
	"reflect"
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
	MQTT_TOPIC_AGL_RECIPE          = "agl/prod/things/" + MQTT_ID_TOKEN + "/recipe"
	MQTT_TOPIC_AWS_UPDATE_ACCEPTED = "$aws/things/" + MQTT_ID_TOKEN + "/shadow/update/accepted"
	MQTT_TOPIC_AWS_UPDATE_DELTA    = "$aws/things/" + MQTT_ID_TOKEN + "/shadow/update/delta"
)

type layerID string

const (
	layerA layerID = "a"
	layerB layerID = "b"
)

type slotID int

const (
	slot1 slotID = 1
	slot2 slotID = 2
	slot3 slotID = 3
	slot4 slotID = 4
	slot5 slotID = 5
	slot6 slotID = 6
	slot7 slotID = 7
	slot8 slotID = 8
	slot9 slotID = 9
)

type deviceReported struct {
	// Reported by Agl mode messages
	Mode valueWithTimestamp[DeviceMode]

	// Reported by Agl update messages
	Connected valueWithTimestamp[bool]
	EC        valueWithTimestamp[int]

	// Reported by AWS update messages
	Cooling      valueWithTimestamp[bool]
	Door         valueWithTimestamp[bool]
	FirmwareNCU  valueWithTimestamp[int]
	HumidA       valueWithTimestamp[int]
	HumidB       valueWithTimestamp[int]
	LightA       valueWithTimestamp[bool]
	LightB       valueWithTimestamp[bool]
	RecipeID     valueWithTimestamp[int]
	TankLevel    valueWithTimestamp[int]
	TankLevelRaw valueWithTimestamp[int]
	TempA        valueWithTimestamp[float64]
	TempB        valueWithTimestamp[float64]
	TempTank     valueWithTimestamp[float64]
	TotalOffset  valueWithTimestamp[int]
	Valve        valueWithTimestamp[ValveState]
	WifiLevel    valueWithTimestamp[int]
}

type slot struct {
	Plant        plant.PlantID
	PlantingTime time.Time
	HarvestFrom  time.Time
	HarvestBy    time.Time
}

type Device struct {
	ID string `json:",omitempty"`

	msgQueue   chan *msgUnparsed
	mqttClient paho.Client
	slotChans  []chan *SlotEvent

	Slots map[layerID]map[slotID]slot

	ClientToken string  `json:",omitempty"`
	Recipe      *recipe `json:",omitempty"`

	// Configuration
	Timezone string `json:",omitempty"`

	// Monotonically increasing ID sent out with update messages
	AWSVersion int `json:",omitempty"`

	// Values reported by the device
	Reported deviceReported `json:",omitempty"`
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

type msgReplyBinary interface {
	msgReply
	Marshal() ([]byte, error)
}

func (d *Device) saveName() string {
	return fmt.Sprintf("plantcube-%s.json", d.ID)
}

// IsSaved returns whether a file exists with saved metadata for the
// device.
func (d *Device) IsSaved() bool {
	sn := d.saveName()
	_, err := os.Stat(sn)
	return !errors.Is(err, os.ErrNotExist)
}

// RestoreFromFile loads all the device's metadata from a file,
// returning any error that happens while doing that.
func (d *Device) RestoreFromFile() error {
	sn := d.saveName()
	m, err := os.ReadFile(sn)
	if err != nil {
		return fmt.Errorf("failed to read '%s': %w", sn, err)
	}
	err = pickyUnmarshal(m, d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal '%s': %w", m, err)
	}
	return nil
}

// Save saves the device's metadata to a file.
func (d *Device) Save() error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal device '%s': %w", d.ID, err)
	}
	sn := d.saveName()
	err = os.WriteFile(sn, b, 0644)
	if err != nil {
		return fmt.Errorf("failed to write '%s': %w", sn, err)
	}
	return nil
}

type SlotEvent struct {
	SlotID string
}

func (d *Device) GetSlotChan() chan *SlotEvent {
	c := make(chan *SlotEvent, 5)
	d.slotChans = append(d.slotChans, c)
	return c
}

func (d *Device) DropSlotChan(drop chan *SlotEvent) {
	for i, c := range d.slotChans {
		if c == drop {
			d.slotChans = append(d.slotChans[:i], d.slotChans[i+1:]...)
			return
		}
	}
}

func (d *Device) AddPlant(slot string, plantID int) error {
	return nil
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
		replies, err = d.processAglMode(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "recipe/get" {
		replies, err = d.processAglRecipeGet(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "shadow/update" {
		replies, err = d.processAglShadowUpdate(msg)
	} else if msg.prefix == "$aws" && msg.event == "shadow/get" {
		err = d.processAWSShadowGet(msg)
	} else if msg.prefix == "$aws" && msg.event == "shadow/update" {
		replies, err = d.processAWSShadowUpdate(msg, time.Now())
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
		var (
			b   []byte
			err error
		)
		rbType := reflect.TypeOf((*msgReplyBinary)(nil)).Elem()
		if reflect.TypeOf(r).Implements(rbType) {
			b, err = r.(msgReplyBinary).Marshal()
		} else {
			b, err = json.Marshal(r)
		}
		if err != nil {
			return fmt.Errorf("failed marshalling '%s': %w", render.Render(r), err)
		}
		topic := strings.ReplaceAll(r.topic(), MQTT_ID_TOKEN, d.ID)
		token := d.mqttClient.Publish(topic, 1, false, b)
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
