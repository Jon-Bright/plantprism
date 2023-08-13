package device

import (
	"flag"
	"fmt"
	"github.com/Jon-Bright/plantprism/logs"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/thlib/go-timezone-local/tzlocal"
	"golang.org/x/exp/slices"
	"strings"
	"time"
)

const (
	// We sometimes see sprees of 3 or 4 messages. This should be
	// enough buffer to prevent blocking in those situations.
	MSG_QUEUE_BUFFER    = 5
	DEFAULT_USER_OFFSET = 7 * 60 * 60 // Sun rises at 7am
)

type deviceList []string

var (
	deviceMap      map[string]*Device
	allowedDevices deviceList
	log            *logs.Loggers

	timezone       string
	sunriseTimeStr string

	totalOffset int
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

func Get(id string, c paho.Client) (*Device, error) {
	d, ok := deviceMap[id]
	if !ok {
		return instantiateDevice(id, c)
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
	t, err := time.Parse("15:04", sunriseTimeStr)
	if err != nil {
		return fmt.Errorf("unable to parse sunrise '%s': %v", sunriseTimeStr, err)
	}
	zero, _ := time.Parse("15:04", "00:00") // This isn't going to error out
	d := t.Sub(zero)
	log.Info.Printf("Sunrise at %02d:%02d", d/time.Hour, (d%time.Hour)/time.Minute)

	// The total_offset is one day minus sunrise _plus_ the timezone offset
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("unable to load zone '%s': %v", timezone, err)
	}
	_, current_offset := time.Now().In(tz).Zone()
	totalOffset = int((24*time.Hour - d).Seconds()) + current_offset

	log.Info.Printf("totalOffset %d sec", totalOffset)
	return nil
}

func instantiateDevice(id string, c paho.Client) (*Device, error) {
	if !slices.Contains(allowedDevices, id) {
		return nil, fmt.Errorf("device ID '%s' is not an allowed device", id)
	}
	d := Device{}
	d.id = id
	d.msgQueue = make(chan *msgUnparsed, MSG_QUEUE_BUFFER)
	d.mqttClient = c

	// TODO: should load previous state here and only set defaults
	// if there's no previous state.

	d.timezone = timezone
	d.userOffset = DEFAULT_USER_OFFSET
	d.mode = ModeDefault

	deviceMap[id] = &d

	go d.processingLoop()
	return &d, nil
}
