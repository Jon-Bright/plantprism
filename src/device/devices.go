package device

import (
	"flag"
	"fmt"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/thlib/go-timezone-local/tzlocal"
	"golang.org/x/exp/slices"
	"strings"
	"time"
)

const (
	// We sometimes see sprees of 3 or 4 messages. This should be
	// enough buffer to prevent blocking in those situations.
	MSG_QUEUE_BUFFER = 5

	defaultTempDay     = 23.0
	defaultTempNight   = 20.0
	defaultWaterTarget = 70
	defaultWaterDelay  = 8 * time.Hour
	defaultDayLength   = 15*time.Hour + 30*time.Minute
)

type deviceList []string

var (
	deviceMap      map[string]*Device
	allowedDevices deviceList
	log            *logs.Loggers

	timezone       string
	sunriseTimeStr string

	sunriseD time.Duration

	defaultLEDVals = []byte{0x3d, 0x27, 0x21, 0x0a}
)

func SetLoggers(l *logs.Loggers) {
	log = l
}

func (l *deviceList) String() string {
	return strings.Join(*l, ",")
}

func (l *deviceList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func Get(id string, p Publisher) (*Device, error) {
	d, ok := deviceMap[id]
	if !ok {
		return instantiateDevice(id, p)
	}
	return d, nil
}

func InitFlags() {
	flag.Var(&allowedDevices, "device", "Allowed device ID. Can be specified multiple times.")
	defaultTZ, err := tzlocal.RuntimeTZ()
	if err != nil {
		panic(fmt.Sprintf("Failed to get timezone: %v", err))
	}
	flag.StringVar(&timezone, "timezone", defaultTZ, "Timezone to be sent to Plantcube. Default is this machine's timezone.")
	flag.StringVar(&sunriseTimeStr, "sunrise", "07:00", "The time at which the Plantcube's sun rises.")
}

func ProcessFlags() error {
	var err error
	sunriseD, err = parseSunriseToDuration(sunriseTimeStr)
	if err != nil {
		return err
	}
	log.Info.Printf("Sunrise at %02d:%02d", sunriseD/time.Hour, (sunriseD%time.Hour)/time.Minute)

	return nil
}

func parseSunriseToDuration(sunrise string) (time.Duration, error) {
	t, err := time.Parse("15:04", sunrise)
	if err != nil {
		return 0, fmt.Errorf("unable to parse sunrise '%s': %w", sunrise, err)
	}
	zero, _ := time.Parse("15:04", "00:00") // This isn't going to error out
	return t.Sub(zero), nil
}

func instantiateDevice(id string, p Publisher) (*Device, error) {
	if !slices.Contains(allowedDevices, id) {
		return nil, fmt.Errorf("device ID '%s' is not an allowed device", id)
	}
	d := Device{}
	d.ID = id
	d.msgQueue = make(chan *msgUnparsed, MSG_QUEUE_BUFFER)
	d.publisher = p
	d.slotChans = []chan *SlotEvent{}

	// Go is happy to let us reset a Timer later, but refuses to
	// create an unstarted timer. We could create the Timer when
	// we need it, but that needs us to be using sync.Mutex as we
	// might do that from any of several HTTP servers, or from
	// MQTT. The same is _theoretically_ true here, but in
	// practice, devices will be instantiated early in our
	// lifetime, so the risk is minimal. So, we create a Timer for
	// (a long time away), then stop it. It can now be reset later
	// without worry.
	d.saveTimer = time.AfterFunc(24*265*time.Hour, d.queuedSave)
	d.saveTimer.Stop()

	if d.IsSaved() {
		err := d.RestoreFromFile()
		if err != nil {
			return nil, fmt.Errorf("restore from file failed for device ID '%s': %v", id, err)
		}
		if d.ID != id {
			return nil, fmt.Errorf("restored device has incorrect ID, want '%s', got '%s'", id, d.ID)
		}
	} else {
		d.Slots = map[layerID]map[slotID]slot{
			layerA: map[slotID]slot{
				slot1: slot{},
				slot2: slot{},
				slot3: slot{},
				slot4: slot{},
				slot5: slot{},
				slot6: slot{},
				slot7: slot{},
				slot8: slot{},
				slot9: slot{},
			},
			layerB: map[slotID]slot{
				slot1: slot{},
				slot2: slot{},
				slot3: slot{},
				slot4: slot{},
				slot5: slot{},
				slot6: slot{},
				slot7: slot{},
				slot8: slot{},
				slot9: slot{},
			},
		}
		t := time.Now()
		d.Timezone = timezone
		var err error
		d.Recipe, err = CreateRecipe(t, defaultLEDVals, defaultTempDay, defaultTempNight, defaultWaterTarget, defaultWaterDelay, defaultDayLength, false, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create default recipe: %w", err)
		}
		d.Reported.Mode.update(ModeDefault, t)
		d.Reported.RecipeID.update(int(d.Recipe.ID), t)
		err = d.Save()
		if err != nil {
			return nil, fmt.Errorf("device id '%s', failed to save defaults: %v", id, err)
		}
	}

	if deviceMap == nil {
		deviceMap = make(map[string]*Device)
	}

	deviceMap[id] = &d

	go d.processingLoop()
	return &d, nil
}
