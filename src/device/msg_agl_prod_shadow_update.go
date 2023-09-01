package device

import (
	"errors"
)

// Example: {"state":{"reported":{"connected": true}}}
// Example: {"state":{"reported":{"ec": 1306}}}
type msgAglShadowUpdateReported struct {
	Connected *bool
	EC        *int
	TankLevel *int `json:"tank_level"`
}
type msgAglShadowUpdateState struct {
	Reported msgAglShadowUpdateReported
}
type msgAglShadowUpdate struct {
	State msgAglShadowUpdateState
}

func parseAglShadowUpdate(msg *msgUnparsed) (*msgAglShadowUpdate, error) {
	var m msgAglShadowUpdate
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.State.Reported.Connected == nil && m.State.Reported.EC == nil && m.State.Reported.TankLevel == nil {
		return nil, errors.New("no fields set")
	}
	// Never seen more than one set at the same time, but there's
	// no real reason they shouldn't be, so not checking that.
	return &m, nil
}

func (d *Device) processAglShadowUpdate(msg *msgUnparsed) ([]msgReply, error) {
	m, err := parseAglShadowUpdate(msg)
	if err != nil {
		return nil, err
	}
	r := m.State.Reported
	if r.Connected != nil {
		d.Reported.Connected.update(*r.Connected, msg.t)
	}
	if r.EC != nil {
		d.Reported.EC.update(*r.EC, msg.t)
	}
	if r.TankLevel != nil {
		d.Reported.TankLevel.update(*r.TankLevel, msg.t)
	}
	reply := d.getAWSShadowUpdateAcceptedReply(msg.t, true)
	return []msgReply{reply}, nil
}
