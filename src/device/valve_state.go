package device

import (
	"fmt"
)

type ValveState int

const (
	ValveOpenLayerB ValveState = 0
	ValveOpenLayerA ValveState = 1
	ValveClosed     ValveState = 4
)

func (v ValveState) String() string {
	switch v {
	case ValveOpenLayerB:
		return "Open/LayerB"
	case ValveOpenLayerA:
		return "Open/LayerA"
	case ValveClosed:
		return "Closed"
	default:
		return fmt.Sprintf("UnknownValveState%d", int(v))
	}
}
