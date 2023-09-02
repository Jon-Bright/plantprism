package device

import (
	"bytes"
	"time"
)

// Example: {"label":"NCU_SYS_LOG","timestamp":1687329836,"payload":{"error_log":"MGOS_SHADOW_UPDATE_REJECTED 400 Missing required node: state
//
//	timer: 0; retries: 0; buff: {'clientToken':'5975bc44','state':{'reported':","function_name":"aws_shadow_grp_handler"}}
//
// Example: {"label":"MCU_MODE_STATE","timestamp":1687685966,"payload":{"mode":"ECO_MODE","state":"1","layer":"APPLIANCE"}}
// ^- this one is its way of complaining the door's open too long
type msgAglEventWarningPayload struct {
	// These are present for Label==NCU_SYS_LOG
	ErrorLog     *string `json:"error_log"`
	FunctionName *string `json:"function_name"`

	// These are present for Label==MCU_MODE_STATE
	Mode  *string
	State *string
	Layer *string
}

type msgAglEventWarning struct {
	Label     *string
	Timestamp *int
	Payload   msgAglEventWarningPayload
}

func parseAglEventWarning(msg *msgUnparsed) (*msgAglEventWarning, error) {
	var m msgAglEventWarning
	// The warnings frequently contain newlines. Replace them.
	stripped := bytes.ReplaceAll(msg.content, []byte{0x0a}, []byte{'_'})
	err := pickyUnmarshal(stripped, &m)
	if err != nil {
		return nil, err
	}
	// There's some consistency checking we could do here
	// (emptiness, at least), but we don't have a large sample of
	// these messages to be sure what's allowed and what's not.
	return &m, nil
}

func (d *Device) processAglEventWarning(msg *msgUnparsed) error {
	m, err := parseAglEventWarning(msg)
	if err != nil {
		return err
	}
	if *m.Label == "NCU_SYS_LOG" {
		log.Warn.Printf("Plantcube syslog warning, time %s, label '%s', function '%s', log '%s'", time.Unix(int64(*m.Timestamp), 0).Local().String(), *m.Label, *m.Payload.FunctionName, *m.Payload.ErrorLog)
	} else if *m.Label == "MCU_MODE_STATE" {
		// TODO: When we get an MCU_MODE_STATE, we should warn
		// the frontend about the door being open
		log.Warn.Printf("Plantcube mode warning, time %s, label '%s', mode '%s', state '%s', layer '%s'", time.Unix(int64(*m.Timestamp), 0).Local().String(), *m.Label, *m.Payload.Mode, *m.Payload.State, *m.Payload.Layer)
	} else {
		log.Error.Printf("Unknown Plantcube warning! time %s, label '%s', raw '%s'", time.Unix(int64(*m.Timestamp), 0).Local().String(), *m.Label, string(msg.content))
	}

	return nil
}
