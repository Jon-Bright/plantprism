package device

import (
	"bytes"
	"time"
)

// Example: {"label":"NCU_SYS_LOG","timestamp":1687329836,"payload":{"error_log":"MGOS_SHADOW_UPDATE_REJECTED 400 Missing required node: state
//
//	timer: 0; retries: 0; buff: {'clientToken':'5975bc44','state':{'reported':","function_name":"aws_shadow_grp_handler"}}
type msgAglEventWarningPayload struct {
	ErrorLog     *string `json:"error_log"`
	FunctionName *string `json:"function_name"`
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
	log.Warn.Printf("Plantcube warning, time %s, label '%s', function '%s', log '%s'", time.Unix(int64(*m.Timestamp), 0).String(), *m.Label, *m.Payload.FunctionName, *m.Payload.ErrorLog)

	return nil
}
