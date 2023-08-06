package device

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Device struct {
	id       string
	msgQueue chan *msgUnparsed
}

type msgUnparsed struct {
	prefix  string
	event   string
	content []byte
}

func (d *Device) ProcessMessage(prefix string, event string, content []byte) {
	d.msgQueue <- &msgUnparsed{prefix, event, content}
}

func (d *Device) processingLoop() {
	for {
		msg := <-d.msgQueue
		err := d.processMessage(msg)
		if err != nil {
			log.Error.Printf(err.Error())
		}
	}
}

func (d *Device) processMessage(msg *msgUnparsed) error {
	var err error
	if msg.prefix == "agl/prod" && msg.event == "shadow/update" {
		err = d.processAglShadowUpdate(msg)
	} else if msg.prefix == "agl/prod" && msg.event == "shadow/get" {
		err = d.processAglShadowGet(msg)
	} else {
		err = errors.New("no handler found")
	}
	if err != nil {
		return fmt.Errorf("failed parsing prefix '%s', event '%s': %v", msg.prefix, msg.event, err)
	}
	return nil
}

// Example: {"state":{"reported":{"connected": true}}}
// Example: {"state":{"reported":{"ec": 1306}}}
type msgAglShadowUpdateReported struct {
	Connected bool
	EC        int
}
type msgAglShadowUpdateState struct {
	Reported msgAglShadowUpdateReported
}
type msgAglShadowUpdate struct {
	State msgAglShadowUpdateState
}

func parseAglShadowUpdate(msg *msgUnparsed) (*msgAglShadowUpdate, error) {
	var m msgAglShadowUpdate
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *Device) processAglShadowUpdate(msg *msgUnparsed) error {
	m, err := parseAglShadowUpdate(msg)
	if err != nil {
		return err
	}
	_ = m
	return nil
}

// No parsing: the only time we see this, it has no content
func (d *Device) processAglShadowGet(msg *msgUnparsed) error {
	return nil
}

// Example: {"clientToken":"5975bc44"}
type msgAWSShadowGet struct {
	ClientToken string
}

func parseAWSShadowGet(msg *msgUnparsed) (*msgAWSShadowGet, error) {
	var m msgAWSShadowGet
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.ClientToken == "" {
		return nil, errors.New("no ClientToken")
	} else if len(m.ClientToken) < 8 {
		return nil, fmt.Errorf("ClientToken '%s' too short", m.ClientToken)
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
	_ = m
	return nil
}
