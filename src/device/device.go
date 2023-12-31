package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/lupguo/go-render/render"
	"go.einride.tech/pid"

	"github.com/Jon-Bright/plantprism/plant"
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

	MQTT_ID_TOKEN                  = "<ID>"
	MQTT_TOPIC_AGL_GET_ACCEPTED    = "agl/all/things/" + MQTT_ID_TOKEN + "/shadow/get/accepted"
	MQTT_TOPIC_AGL_RECIPE          = "agl/prod/things/" + MQTT_ID_TOKEN + "/recipe"
	MQTT_TOPIC_AGL_RPC_PUT         = "agl/all/things/" + MQTT_ID_TOKEN + "/rpc/put"
	MQTT_TOPIC_AWS_UPDATE_ACCEPTED = "$aws/things/" + MQTT_ID_TOKEN + "/shadow/update/accepted"
	MQTT_TOPIC_AWS_UPDATE_DELTA    = "$aws/things/" + MQTT_ID_TOKEN + "/shadow/update/delta"

	KeepBackups           = 20
	SaveDelay             = 20 * time.Second
	RecipeDelay           = 2 * time.Minute
	WateringDelayHarvest  = 41 * time.Minute               // No idea why this delay, but it's what's in the dumps
	WateringDelayPlanting = 11*time.Minute - 4*time.Second // Also not exactly a round number, same reason
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
	TempA        valueWithTimestamp[floatDP]
	TempB        valueWithTimestamp[floatDP]
	TempTank     valueWithTimestamp[floatDP]
	TotalOffset  valueWithTimestamp[int]
	Valve        valueWithTimestamp[ValveState]
	WifiLevel    valueWithTimestamp[int]
}

type slot struct {
	Plant        plant.PlantID
	PlantingTime time.Time
	GerminatedBy time.Time
	HarvestFrom  time.Time
	HarvestBy    time.Time
}

type Publisher interface {
	Publish(topic string, payload []byte) error
}

type Device struct {
	ID string `json:",omitempty"`

	clock         clock.Clock
	msgQueue      chan *msgUnparsed
	publisher     Publisher
	slotChans     []chan *SlotEvent
	statusChans   []chan *StatusEvent
	saveTimer     *clock.Timer
	recipeTimer   *clock.Timer
	wateringTimer *clock.Timer

	// Stuff we maintain
	SmoothedEC   float64                     `json:",omitempty"`
	NutrientPID  *pid.Controller             `json:",omitempty"`
	WantNutrient int                         `json:",omitempty"`
	Slots        map[layerID]map[slotID]slot `json:",omitempty"`
	Recipe       *recipe                     `json:",omitempty"`

	// Configuration
	Timezone   string `json:",omitempty"`
	UserOffset int    `json:",omitempty"`

	// Monotonically increasing ID sent out with update messages
	AWSVersion int `json:",omitempty"`

	// Values reported by the device
	ClientToken string         `json:",omitempty"`
	Reported    deviceReported `json:",omitempty"`
}

type msgUnparsed struct {
	prefix  string
	event   string
	content []byte
	t       time.Time
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

var testMode = false

func SetTestMode() {
	testMode = true
}

func (d *Device) saveName() string {
	if testMode {
		return fmt.Sprintf("test-plantcube-%s.json", d.ID)
	}
	return fmt.Sprintf("plantcube-%s.json", d.ID)
}

func (d *Device) backupName(gen int) string {
	return fmt.Sprintf("plantcube-%s-backup-%d.json", d.ID, gen)
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

func (d *Device) MakeBackups() error {
	var src, dst string
	src = d.backupName(KeepBackups)
	expectExist := false
	for i := KeepBackups; i >= 0; i-- {
		dst = src
		if i > 0 {
			src = d.backupName(i - 1)
		} else {
			src = d.saveName()
		}
		err := os.Rename(src, dst)
		if err != nil {
			if os.IsNotExist(err) && !expectExist {
				// We might not have made this many
				// backups yet (or, indeed, have saved
				// at all). As soon as we see one that
				// _does_ exist, we'll set expectExist
				// below and this condition won't
				// match.
				continue
			}
			return fmt.Errorf("failed shuffling backup '%s' to '%s', gen %d: %w", src, dst, i, err)
		}
		expectExist = true
	}
	return nil
}

// Save saves the device's metadata to a file.
func (d *Device) Save() error {
	if testMode {
		return nil
	}
	err := d.MakeBackups()
	if err != nil {
		return fmt.Errorf("failed pre-save backup: %w", err)
	}
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

func (d *Device) queuedSave() {
	err := d.Save()
	if err != nil {
		log.Critical.Fatalf("failed queued save: %v", err)
	}
}

func (d *Device) QueueSave() {
	d.saveTimer.Reset(SaveDelay)
}

type SlotEvent struct {
	Layer layerID
	Slot  slotID
}

func (d *Device) GetSlotChan() chan *SlotEvent {
	c := make(chan *SlotEvent, 20)
	d.slotChans = append(d.slotChans, c)
	for _, l := range []layerID{layerA, layerB} {
		for s := slot1; s <= slot9; s++ {
			se := SlotEvent{l, s}
			c <- &se
		}
	}
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

func (d *Device) streamSlotUpdate(l layerID, s slotID) {
	se := SlotEvent{l, s}
	for _, c := range d.slotChans {
		c <- &se
	}
}

type StatusEvent struct {
	TempA        float64
	TempB        float64
	TempTank     float64
	HumidA       int
	HumidB       int
	LightA       bool
	LightB       bool
	TankLevel    int
	Valve        ValveState
	Mode         DeviceMode
	Door         bool
	EC           int
	SmoothedEC   float64
	WantNutrient int
}

func (d *Device) GetStatusChan() chan *StatusEvent {
	c := make(chan *StatusEvent, 5)
	d.statusChans = append(d.statusChans, c)
	c <- d.getStatusUpdate()
	return c
}

func (d *Device) DropStatusChan(drop chan *StatusEvent) {
	for i, c := range d.statusChans {
		if c == drop {
			d.statusChans = append(d.statusChans[:i], d.statusChans[i+1:]...)
			return
		}
	}
}

func (d *Device) getStatusUpdate() *StatusEvent {
	// We could just stream what changed, but that seems like
	// hoop-jumping when we only have a few values to deliver
	// anyway, so we just deliver them all.
	se := StatusEvent{
		TempA:        float64(d.Reported.TempA.Value),
		TempB:        float64(d.Reported.TempB.Value),
		TempTank:     float64(d.Reported.TempTank.Value),
		HumidA:       d.Reported.HumidA.Value,
		HumidB:       d.Reported.HumidB.Value,
		LightA:       d.Reported.LightA.Value,
		LightB:       d.Reported.LightB.Value,
		TankLevel:    d.Reported.TankLevel.Value,
		Valve:        d.Reported.Valve.Value,
		Mode:         d.Reported.Mode.Value,
		Door:         d.Reported.Door.Value,
		EC:           d.Reported.EC.Value,
		SmoothedEC:   d.SmoothedEC,
		WantNutrient: d.WantNutrient,
	}
	return &se
}

func (d *Device) streamStatusUpdate() {
	se := d.getStatusUpdate()
	for _, c := range d.statusChans {
		c <- se
	}
}

func (d *Device) SetSunrise(s time.Duration) error {
	t := d.clock.Now()
	to, err := calcTotalOffset(d.Timezone, t, s)
	if err != nil {
		return fmt.Errorf("failed calculating total offset for sunrise %v: %w", s, err)
	}
	d.UserOffset = int(s / time.Second)
	d.AWSVersion++
	deltaD := Device{
		AWSVersion: d.AWSVersion,
	}
	deltaD.Reported.TotalOffset.update(to, t)
	delta := deltaD.getAWSShadowUpdateDeltaReply(t, t)
	err = d.sendReplies([]msgReply{delta})
	if err != nil {
		return fmt.Errorf("failed sending delta for new sunrise: %w", err)
	}

	return nil
}

func parseSlot(slot string) (layerID, slotID, error) {
	if len(slot) != 2 {
		return "", 0, fmt.Errorf("slot string '%s' has wrong length", slot)
	}
	var l layerID
	if slot[0:1] == string(layerA) {
		l = layerA
	} else if slot[0:1] == string(layerB) {
		l = layerB
	} else {
		return "", 0, fmt.Errorf("slot string '%s' has invalid layer", slot)
	}
	if slot[1] < '1' || slot[1] > '9' {
		return "", 0, fmt.Errorf("slot string '%s' has invalid slot", slot)
	}
	s := slotID(slot[1] - '0')
	return l, s, nil
}

func (d *Device) QueueRecipe() {
	d.recipeTimer.Reset(RecipeDelay)
}

func (d *Device) QueueWatering(harvest bool) {
	if harvest {
		d.wateringTimer.Reset(WateringDelayHarvest)
	} else {
		d.wateringTimer.Reset(WateringDelayPlanting)
	}
}

func (d *Device) AddPlant(slotStr string, plantID plant.PlantID) error {
	l, s, err := parseSlot(slotStr)
	if err != nil {
		return err
	}
	if d.Slots[l][s].Plant != 0 {
		return fmt.Errorf("can't plant in slot '%s', it already contains plant ID '%d'", slotStr, d.Slots[l][s].Plant)
	}
	p, err := plant.Get(plantID)
	if err != nil {
		return fmt.Errorf("can't plant in slot '%s': %w", slotStr, err)
	}
	t := d.clock.Now()
	d.Slots[l][s] = slot{
		Plant:        plantID,
		PlantingTime: t,
		GerminatedBy: t.Add(time.Duration(p.Germination)),
		HarvestFrom:  t.Add(time.Duration(p.HarvestFrom)),
		HarvestBy:    t.Add(time.Duration(p.HarvestBy)),
	}
	d.streamSlotUpdate(l, s)
	d.QueueRecipe()
	d.QueueWatering(false)
	d.QueueSave()
	return nil
}

func (d *Device) HarvestPlant(slotStr string) error {
	l, s, err := parseSlot(slotStr)
	if err != nil {
		return err
	}
	if d.Slots[l][s].Plant == 0 {
		return fmt.Errorf("can't harvest in slot '%s', it's already empty", slotStr)
	}
	d.Slots[l][s] = slot{}
	d.streamSlotUpdate(l, s)
	d.QueueRecipe()
	d.QueueWatering(true)
	d.QueueSave()
	return nil
}

func (d *Device) TriggerManualWatering() {
	d.wateringTimer.Stop()
	d.sendWateringRPC()
}

func (d *Device) sendWateringRPC() {
	l := layerA
	if d.layerHasPlants(layerB) {
		l = layerB
	}
	msg := getAglRPCPutWatering(l)
	err := d.sendReplies([]msgReply{msg})
	if err != nil {
		log.Error.Printf("failed sending watering RPC: %v", err)
	}

}

func (d *Device) sendRecipe() {
	err := d.makeNewRecipe()
	if err != nil {
		log.Error.Printf("Failed delayed recipe: %v", err)
	}
}

func (d *Device) layerHasPlants(l layerID) bool {
	for _, s := range d.Slots[l] {
		if s.Plant != 0 {
			return true
		}
	}
	return false
}

func (d *Device) makeNewRecipe() error {
	// TODO: there's a lot more we could do here, but for now, we
	// just activate the layers we need to. We then only replace
	// the recipe when one or both of two conditions is true:
	//
	// 1. it's different
	// 2. the current one is old, for some definition of old
	//
	// Future potential improvements:
	//
	// * We could adjust temperatures when there are germinating
	// plants (maybe with a plain average, maybe just with
	// germination taking priority on the theory that
	// non-germinating plants are more robust).
	// * We could reduce day length when there are only mature
	// plants (where further growth should maybe be inhibited?)
	// * We could adjust lighting colours depending on plant
	// phases.
	//
	// It's unclear from our minimal recipe sample whether the
	// Agrilution code did any of this.
	layerAActive := d.layerHasPlants(layerA)
	layerBActive := d.layerHasPlants(layerB)
	r, err := CreateRecipe(d.clock.Now(), defaultLEDVals, defaultTempDay, defaultTempNight, defaultWaterTarget, defaultWaterDelay, defaultDayLength, layerAActive, layerBActive)
	if err != nil {
		return fmt.Errorf("CreateRecipe failed, layerAActive=%v, layerBActive=%v: %w", layerAActive, layerBActive, err)
	}

	t := d.clock.Now()
	ad := r.AgeDifference(d.Recipe)
	eq, err := r.EqualExceptTimestamps(d.Recipe)
	if err != nil {
		return fmt.Errorf("failed comparing old/new recipes: %w", err)
	}

	log.Info.Printf("New recipe generated at %v, equal %v, age difference %v, layerAActive %v, layerBActive %v", t.Local(), eq, ad, layerAActive, layerBActive)

	d.Recipe = r
	d.AWSVersion++
	deltaD := Device{
		AWSVersion: d.AWSVersion,
	}
	deltaD.Reported.RecipeID.update(int(d.Recipe.ID), t)
	delta := deltaD.getAWSShadowUpdateDeltaReply(t, t)
	err = d.sendReplies([]msgReply{delta})
	if err != nil {
		return fmt.Errorf("failed sending delta for new recipe: %w", err)
	}

	return nil
}

func (d *Device) SetMode(mode DeviceMode) error {
	// TODO: we should check if this is a valid mode change. Can't change to
	// e.g. cinema if we're in the middle of cleaning.
	d.AWSVersion++
	deltaD := Device{
		AWSVersion: d.AWSVersion,
	}
	t := d.clock.Now()
	deltaD.Reported.Mode.update(mode, t)
	delta := deltaD.getAWSShadowUpdateDeltaReply(t, t)
	err := d.sendReplies([]msgReply{delta})
	if err != nil {
		return fmt.Errorf("failed sending delta for mode change: %w", err)
	}

	return nil
}

func (d *Device) ProcessMessage(prefix string, event string, content []byte) {
	d.msgQueue <- &msgUnparsed{prefix, event, content, d.clock.Now()}
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
		// Technically, we could always queue a save, but "if
		// the message was worth replying to, it's probably
		// worth saving" is a decent rule of thumb.
		d.QueueSave()
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
		err = d.publisher.Publish(topic, b)
		if err != nil {
			return fmt.Errorf("failed publishing reply: %w", err)
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
