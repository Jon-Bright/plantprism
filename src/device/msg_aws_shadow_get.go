package device

import (
	"errors"
	"fmt"
)

// Example: {"clientToken":"5975bc44"}
type msgAWSShadowGet struct {
	ClientToken *string
}

func parseAWSShadowGet(msg *msgUnparsed) (*msgAWSShadowGet, error) {
	var m msgAWSShadowGet
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.ClientToken == nil {
		return nil, errors.New("no ClientToken")
	} else if len(*m.ClientToken) < 8 {
		return nil, fmt.Errorf("ClientToken '%s' too short", *m.ClientToken)
	}
	// Could theoretically check if it's hex, which the
	// Plantcube's all are, but do we care?
	return &m, nil
}

func (d *Device) processAWSShadowGet(msg *msgUnparsed) error {
	m, err := parseAWSShadowGet(msg)
	if err != nil {
		return err
	}
	if d.ClientToken != "" && d.ClientToken != *m.ClientToken {
		return fmt.Errorf("ClientToken changed. want '%s', got '%s'", d.ClientToken, *m.ClientToken)
	}
	d.ClientToken = *m.ClientToken
	log.Info.Printf("Set ClientToken to '%s'", d.ClientToken)
	// NB: there's no reply to this message
	return nil
}
