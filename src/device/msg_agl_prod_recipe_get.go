package device

import (
	"errors"
	"fmt"
)

// Example: {"version":7, "format": "binary" }
type msgAglRecipeGet struct {
	Version *int
	Format  *string
}

func parseAglRecipeGet(msg *msgUnparsed) (*msgAglRecipeGet, error) {
	var m msgAglRecipeGet
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.Version == nil {
		return nil, errors.New("missing version")
	}
	if m.Format == nil {
		return nil, errors.New("missing format")
	}
	if *m.Version != 7 {
		return nil, fmt.Errorf("invalid version: %d", *m.Version)
	}
	if *m.Format != "binary" {
		return nil, fmt.Errorf("invalid format: '%s'", *m.Format)
	}
	return &m, nil
}

func (d *Device) processAglRecipeGet(msg *msgUnparsed) error {
	m, err := parseAglRecipeGet(msg)
	if err != nil {
		return err
	}
	// TODO : Process this
	_ = m
	return nil
}
