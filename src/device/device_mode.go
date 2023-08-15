package device

import (
	"fmt"
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

func (m DeviceMode) String() string {
	switch m {
	case ModeDefault:
		return "Default"
	case ModeDebug:
		return "Debug"
	case ModeRinseEnd:
		return "RinseEnd"
	case ModeTankDrainCleaning:
		return "TankDrainCleaning"
	case ModeTankDrainExplicit:
		return "TankDrainExplicit"
	case ModeCleaning:
		return "Cleaning"
	case ModeUnknown:
		return "Unknown"
	case ModeSilent:
		return "Silent"
	case ModeCinema:
		return "Cinema"
	default:
		return fmt.Sprintf("UnknownMode%d", int(m))
	}
}
