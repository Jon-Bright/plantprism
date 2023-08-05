package device

import (
	"flag"
	"fmt"
	"golang.org/x/exp/slices"
	"strings"
)

type deviceList []string

type Device struct {
	id string
}

var (
	deviceMap      map[string]*Device
	allowedDevices deviceList
)

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
	d := Device{id}
	deviceMap[id] = &d
	return &d, nil
}
