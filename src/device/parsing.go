package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func pickyUnmarshal(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	err := d.Decode(v)
	if err != nil {
		return err
	}
	// The data should be one object and nothing more
	if t, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing data after decode: %T / %v, err %w", t, t, err)
	}
	return nil
}

func parseAglMode(msg *msgUnparsed) (*msgAglMode, error) {
	var m msgAglMode
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.PrevMode == nil {
		return nil, errors.New("No prev_mode field")
	} else if m.Mode == nil {
		return nil, errors.New("No mode field")
	} else if m.Trigger == nil {
		return nil, errors.New("No trigger field")
	} else if *m.PrevMode < ModeDefault || *m.PrevMode >= ModeOutOfRange {
		return nil, fmt.Errorf("PrevMode %d is invalid", *m.PrevMode)
	} else if *m.Mode < ModeDefault || *m.Mode >= ModeOutOfRange {
		return nil, fmt.Errorf("Mode %d is invalid", *m.Mode)
	} else if *m.Mode == *m.PrevMode {
		return nil, fmt.Errorf("Mode %d is the same as previously", *m.Mode)
	} else if *m.Trigger < ModeTriggerApp || *m.Trigger >= ModeTriggerOutOfRange {
		return nil, fmt.Errorf("Trigger %d is invalid", *m.Trigger)
	}

	return &m, nil
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

func parseAglShadowUpdate(msg *msgUnparsed) (*msgAglShadowUpdate, error) {
	var m msgAglShadowUpdate
	err := pickyUnmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.State.Reported.Connected == nil && m.State.Reported.EC == nil {
		return nil, errors.New("no fields set")
	}
	// Never seen them both set at the same time, but there's no
	// real reason they shouldn't be, so not checking that.
	return &m, nil
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

func (m *msgAWSShadowUpdateReported) empty() bool {
	return m.Cooling == nil && m.Door == nil && m.FirmwareNCU == nil && m.HumidA == nil && m.HumidB == nil &&
		m.LightA == nil && m.LightB == nil && m.RecipeID == nil && m.TankLevel == nil &&
		m.TankLevelRaw == nil && m.TempA == nil && m.TempB == nil && m.TempTank == nil &&
		m.TotalOffset == nil && m.Valve == nil && m.WifiLevel == nil
}

func (m *msgAWSShadowUpdateReported) validate() error {
	if m.empty() {
		return errors.New("update is empty")
	}
	if m.FirmwareNCU != nil && *m.FirmwareNCU < 1667466618 {
		return fmt.Errorf("NCU firmware too old: %d", *m.FirmwareNCU)
	}
	if m.HumidA != nil && (*m.HumidA < 30 || *m.HumidA > 100) {
		// 30% humidity isn't technically an error, but it
		// sure would be surprising
		return fmt.Errorf("humidity A out of range: %d", *m.HumidA)
	}
	if m.HumidB != nil && (*m.HumidB < 30 || *m.HumidB > 100) {
		// 30% humidity isn't technically an error, but it
		// sure would be surprising
		return fmt.Errorf("humidity B out of range: %d", *m.HumidB)
	}
	if m.RecipeID != nil && *m.RecipeID != 1 && *m.RecipeID < 1680300000 {
		// 1680300000 is 2023-04-01. The recipe shouldn't be that old.
		// 1 is a flag value the Plantcube uses to request a recipe.
		return fmt.Errorf("recipe ID invalid: %d", *m.RecipeID)
	}
	if m.TankLevel != nil && (*m.TankLevel < 0 || *m.TankLevel > 2) {
		return fmt.Errorf("tank level invalid: %d", *m.TankLevel)
	}
	if m.TankLevelRaw != nil && (*m.TankLevelRaw < 0 || *m.TankLevelRaw > 2) {
		return fmt.Errorf("raw tank level invalid: %d", *m.TankLevelRaw)
	}
	if m.TempA != nil && (*m.TempA < 10.0 || *m.TempA > 40.0) {
		return fmt.Errorf("temp A out of range: %.1f", *m.TempA)
	}
	if m.TempB != nil && (*m.TempB < 10.0 || *m.TempB > 40.0) {
		return fmt.Errorf("temp B out of range: %.1f", *m.TempB)
	}
	if m.TempTank != nil && (*m.TempTank < 10.0 || *m.TempTank > 40.0) {
		return fmt.Errorf("temp tank out of range: %.1f", *m.TempTank)
	}
	if m.TotalOffset != nil && (*m.TotalOffset < 0 || *m.TotalOffset > 86400) {
		// 86400 = 1 day in seconds. Offset shouldn't exceed this.
		return fmt.Errorf("total offset out of range: %d", *m.TotalOffset)
	}
	if m.Valve != nil && *m.Valve != ValveOpenLayerB && *m.Valve != ValveOpenLayerA && *m.Valve != ValveClosed {
		return fmt.Errorf("valve value invalid: %d", *m.Valve)
	}
	if m.WifiLevel != nil && (*m.WifiLevel < 0 || *m.WifiLevel > 2) {
		return fmt.Errorf("wifi level invalid: %d", *m.WifiLevel)
	}
	return nil
}

func parseAWSShadowUpdate(msg *msgUnparsed) (*msgAWSShadowUpdate, error) {
	var m msgAWSShadowUpdate
	err := json.Unmarshal(msg.content, &m)
	if err != nil {
		return nil, err
	}
	if m.ClientToken == nil {
		return nil, errors.New("no ClientToken")
	} else if len(*m.ClientToken) < 8 {
		return nil, fmt.Errorf("ClientToken '%s' too short", *m.ClientToken)
	}
	err = m.State.Reported.validate()
	if err != nil {
		return nil, err
	}
	return &m, nil
}
