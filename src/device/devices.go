package device

import (
	"flag"
	"fmt"
	"github.com/Jon-Bright/plantprism/logs"
	"golang.org/x/exp/slices"
	"strings"
)

const (
	// We sometimes see sprees of 3 or 4 messages. This should be
	// enough buffer to prevent blocking in those situations.
	MSG_QUEUE_BUFFER = 5
)

type deviceList []string

var (
	deviceMap      map[string]*Device
	allowedDevices deviceList
	log            *logs.Loggers
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

func Get(id string) (*Device, error) {
	d, ok := deviceMap[id]
	if !ok {
		return instantiateDevice(id)
	}
	return d, nil
}

func InitFlags() {
	flag.Var(&allowedDevices, "device", "Allowed device ID. Can be specified multiple times.")
}

func instantiateDevice(id string) (*Device, error) {
	if !slices.Contains(allowedDevices, id) {
		return nil, fmt.Errorf("device ID '%s' is not an allowed device", id)
	}
	d := Device{}
	d.id = id
	d.msgQueue = make(chan *msgUnparsed, MSG_QUEUE_BUFFER)
	deviceMap[id] = &d

	go d.processingLoop()
	return &d, nil
}
