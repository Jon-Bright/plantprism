package device

import (
	"flag"
	"fmt"
	"github.com/Jon-Bright/plantprism/logs"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/thlib/go-timezone-local/tzlocal"
	"golang.org/x/exp/slices"
	"strings"
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

	timezone string
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
