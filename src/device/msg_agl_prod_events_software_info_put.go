package device

import (
	"time"
)

// Example: {"label":"MCU_MODE_STATE","timestamp":1687686053,"payload":{"mode":"ECO_MODE","state":"0","layer":"APPLIANCE"}}
// ^- this appears to be its way of saying the door has been closed after being open too long
type msgAglEventInfoPayload struct {
	Mode  *string
	State *string
	Layer *string
}

type msgAglEventInfo struct {
	Label     *string
	Timestamp *int
	Payload   msgAglEventInfoPayload
}

func parseAglEventInfo(msg *msgUnparsed) (*msgAglEventInfo, error) {
	var m msgAglEventInfo
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	// There's some consistency checking we could do here
	// (emptiness, at least), but we don't have a large sample of
	// these messages to be sure what's allowed and what's not.
	return &m, nil
}

func (d *Device) processAglEventInfo(msg *msgUnparsed) error {
	m, err := parseAglEventInfo(msg)
	if err != nil {
		return err
	}
	log.Info.Printf("Plantcube info, time %s, label '%s', mode '%s', state '%s', layer '%s'",
		time.Unix(int64(*m.Timestamp), 0).String(),
		*m.Label, *m.Payload.Mode, *m.Payload.State, *m.Payload.Layer)
	// TODO: When we get an MCU_MODE_STATE, we should cancel the
	// prior warning to the frontend about the door being open

	return nil
}
