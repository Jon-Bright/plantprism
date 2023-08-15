package device

import (
	"fmt"
)

type ModeTrigger int

const (
	ModeTriggerApp        ModeTrigger = 0
	ModeTriggerDevice     ModeTrigger = 1
	ModeTriggerOutOfRange ModeTrigger = 2
)

func (t ModeTrigger) String() string {
	switch t {
	case ModeTriggerApp:
		return "App"
	case ModeTriggerDevice:
		return "Device"
	default:
		return fmt.Sprintf("UnknownTrigger%d", int(t))
	}
}
