package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	EXPECTED_NCU_MCU_FW_VERSION = 1667466618
)

type floatDP float64

func (f floatDP) MarshalJSON() ([]byte, error) {
	if float64(f) == float64(int(f)) {
		return []byte(strconv.FormatFloat(float64(f), 'f', 1, 32)), nil
	}
	return []byte(strconv.FormatFloat(float64(f), 'f', -1, 32)), nil
}

// Example: {"clientToken":"5975bc44","state":{"reported":{"humid_b":75,"temp_a":22.99,"temp_b":24.19}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_a":22.69,"firmware_ncu":1667466618,"door":false,"cooling":true,"total_offset":69299,"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"wifi_level":0}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_tank":28.34}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":true,"light_b":true}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"tank_level_raw":2}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"door":true}}}

// This struct does triple duty. We use it to interpret incoming
// .../shadow/update messages, but we _also_ use it to construct
// outgoing .../shadow/update/accepted and .../shadow/update/delta
// messages.
type msgAWSShadowUpdateData struct {
	// `Mode`, `Connected` and `EC` shouldn't be reported by
	// $aws/.../update, but they're here because they appear in
	// .../update/accepted
	Connected    *bool       `json:"connected,omitempty"`
	Cooling      *bool       `json:"cooling,omitempty"`
	Door         *bool       `json:"door,omitempty"`
	EC           *int        `json:"ec,omitempty"`
	FirmwareNCU  *int        `json:"firmware_ncu,omitempty"`
	HumidA       *int        `json:"humid_a,omitempty"`
	HumidB       *int        `json:"humid_b,omitempty"`
	LightA       *bool       `json:"light_a,omitempty"`
	LightB       *bool       `json:"light_b,omitempty"`
	Mode         *DeviceMode `json:"mode,omitempty"`
	RecipeID     *int        `json:"recipe_id,omitempty"`
	TankLevel    *int        `json:"tank_level,omitempty"`
	TankLevelRaw *int        `json:"tank_level_raw,omitempty"`
	TempA        *floatDP    `json:"temp_a,omitempty"`
	TempB        *floatDP    `json:"temp_b,omitempty"`
	TempTank     *floatDP    `json:"temp_tank,omitempty"`
	TotalOffset  *int        `json:"total_offset,omitempty"`
	Valve        *ValveState `json:"valve,omitempty"`
	WifiLevel    *int        `json:"wifi_level,omitempty"`
}
type msgAWSShadowUpdateState struct {
	Reported msgAWSShadowUpdateData `json:"reported"`
}
type msgAWSShadowUpdate struct {
	ClientToken *string
	State       msgAWSShadowUpdateState
}

func (m *msgAWSShadowUpdateData) empty() bool {
	return m.Cooling == nil && m.Door == nil && m.FirmwareNCU == nil && m.HumidA == nil && m.HumidB == nil &&
		m.LightA == nil && m.LightB == nil && m.RecipeID == nil && m.TankLevel == nil &&
		m.TankLevelRaw == nil && m.TempA == nil && m.TempB == nil && m.TempTank == nil &&
		m.TotalOffset == nil && m.Valve == nil && m.WifiLevel == nil
}

func (m *msgAWSShadowUpdateData) validate() error {
	if m.empty() {
		return errors.New("update is empty")
	}
	if m.FirmwareNCU != nil && *m.FirmwareNCU < EXPECTED_NCU_MCU_FW_VERSION {
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

func (d *Device) processAWSShadowUpdate(msg *msgUnparsed) ([]msgReply, error) {
	m, err := parseAWSShadowUpdate(msg)
	if err != nil {
		return nil, err
	}
	if *m.ClientToken != d.ClientToken {
		return nil, fmt.Errorf("clientToken '%s' received, but device clientToken is '%s'", *m.ClientToken, d.ClientToken)
	}
	r := &m.State.Reported
	dr := &d.Reported
	if r.Mode != nil {
		return nil, errors.New("unexpected Mode reported in AWS update")
	}
	if r.Connected != nil {
		return nil, errors.New("unexpected Connected reported in AWS update")
	}
	if r.EC != nil {
		return nil, errors.New("unexpected EC reported in AWS update")
	}
	if r.Cooling != nil {
		dr.Cooling.update(*r.Cooling, msg.t)
	}
	if r.Door != nil {
		dr.Door.update(*r.Door, msg.t)
	}
	if r.FirmwareNCU != nil {
		dr.FirmwareNCU.update(*r.FirmwareNCU, msg.t)
	}
	if r.HumidA != nil {
		dr.HumidA.update(*r.HumidA, msg.t)
	}
	if r.HumidB != nil {
		dr.HumidB.update(*r.HumidB, msg.t)
	}
	if r.LightA != nil {
		dr.LightA.update(*r.LightA, msg.t)
	}
	if r.LightB != nil {
		dr.LightB.update(*r.LightB, msg.t)
	}
	if r.RecipeID != nil {
		dr.RecipeID.update(*r.RecipeID, msg.t)
	}
	if r.TankLevel != nil {
		dr.TankLevel.update(*r.TankLevel, msg.t)
	}
	if r.TankLevelRaw != nil {
		dr.TankLevelRaw.update(*r.TankLevelRaw, msg.t)
	}
	if r.TempA != nil {
		dr.TempA.update(*r.TempA, msg.t)
	}
	if r.TempB != nil {
		dr.TempB.update(*r.TempB, msg.t)
	}
	if r.TempTank != nil {
		dr.TempTank.update(*r.TempTank, msg.t)
	}
	if r.TotalOffset != nil {
		dr.TotalOffset.update(*r.TotalOffset, msg.t)
	}
	if r.Valve != nil {
		dr.Valve.update(*r.Valve, msg.t)
	}
	if r.WifiLevel != nil {
		dr.WifiLevel.update(*r.WifiLevel, msg.t)
	}
	replies := []msgReply{
		d.getAWSShadowUpdateAcceptedReply(msg.t, false),
	}
	if dr.RecipeID.wasUpdatedAt(msg.t) && dr.RecipeID.Value != int(d.Recipe.ID) {
		if log != nil {
			log.Info.Printf("Seen recipe difference (%d!=%d), generating delta", dr.RecipeID.Value, d.Recipe.ID)
		}
		// We need to generate a delta message that just
		// covers the Recipe ID.  We therefore make a new
		// Device with our current AWS version, update just
		// the Recipe ID and generate a delta based on this.
		deltaD := Device{
			AWSVersion: d.AWSVersion,
		}
		deltaT := time.Unix(int64(d.Recipe.ID), 0)
		deltaD.Reported.RecipeID.update(int(d.Recipe.ID), deltaT)
		replies = append(replies, deltaD.getAWSShadowUpdateDeltaReply(deltaT, msg.t))
	}
	if dr.Valve.wasUpdatedAt(msg.t) && dr.Valve.Value != ValveClosed {
		d.wateringTimer.Stop()
	}
	return replies, nil
}

type msgAWSShadowUpdateMetadata struct {
	Connected    *msgUpdTS `json:"connected,omitempty"`
	Cooling      *msgUpdTS `json:"cooling,omitempty"`
	Door         *msgUpdTS `json:"door,omitempty"`
	EC           *msgUpdTS `json:"ec,omitempty"`
	FirmwareNCU  *msgUpdTS `json:"firmware_ncu,omitempty"`
	HumidA       *msgUpdTS `json:"humid_a,omitempty"`
	HumidB       *msgUpdTS `json:"humid_b,omitempty"`
	LightA       *msgUpdTS `json:"light_a,omitempty"`
	LightB       *msgUpdTS `json:"light_b,omitempty"`
	Mode         *msgUpdTS `json:"mode,omitempty"`
	RecipeID     *msgUpdTS `json:"recipe_id,omitempty"`
	TankLevel    *msgUpdTS `json:"tank_level,omitempty"`
	TankLevelRaw *msgUpdTS `json:"tank_level_raw,omitempty"`
	TempA        *msgUpdTS `json:"temp_a,omitempty"`
	TempB        *msgUpdTS `json:"temp_b,omitempty"`
	TempTank     *msgUpdTS `json:"temp_tank,omitempty"`
	TotalOffset  *msgUpdTS `json:"total_offset,omitempty"`
	Valve        *msgUpdTS `json:"valve,omitempty"`
	WifiLevel    *msgUpdTS `json:"wifi_level,omitempty"`
}
type msgAWSShadowUpdateAcceptedMetadata struct {
	Reported msgAWSShadowUpdateMetadata `json:"reported"`
}
type msgAWSShadowUpdateAcceptedReply struct {
	State       msgAWSShadowUpdateState            `json:"state"`
	Metadata    msgAWSShadowUpdateAcceptedMetadata `json:"metadata"`
	Version     int                                `json:"version"`
	Timestamp   int                                `json:"timestamp"`
	ClientToken string                             `json:"clientToken,omitempty"`
}

func (m *msgAWSShadowUpdateAcceptedReply) topic() string {
	return MQTT_TOPIC_AWS_UPDATE_ACCEPTED
}

// Construct a reply featuring all values reported at the given
// timestamp, along with metadata for each of those values with the
// timestamp.  /shadow/update to agl/prod _also_ triggers AWS updates,
// as does agl/prod/.../mode, but these come without a client token
// (possibly because they're making it into AWS's shadow via
// Agrilution code, not via a client?), so allow generating these
// without a client ID.
func (d *Device) getAWSShadowUpdateAcceptedReply(t time.Time, omitClientToken bool) msgReply {
	msg := msgAWSShadowUpdateAcceptedReply{}
	r := &msg.State.Reported
	m := &msg.Metadata.Reported

	d.AWSVersion++
	msg.Version = d.AWSVersion
	msg.Timestamp = int(t.Unix())
	if !omitClientToken {
		msg.ClientToken = d.ClientToken
	}
	statusUpdated := d.fillAWSUpdateDataMetadata(t, r, m)
	if statusUpdated {
		d.streamStatusUpdate()
	}
	return &msg
}

func (dev *Device) fillAWSUpdateDataMetadata(t time.Time, d *msgAWSShadowUpdateData, m *msgAWSShadowUpdateMetadata) bool {
	dr := &dev.Reported

	su := false // whether any of the fields in StatusEvent was updated

	ts := msgUpdTS{int(t.Unix())}
	if dr.Connected.wasUpdatedAt(t) {
		d.Connected = &dr.Connected.Value
		m.Connected = &ts
	}
	if dr.Cooling.wasUpdatedAt(t) {
		d.Cooling = &dr.Cooling.Value
		m.Cooling = &ts
	}
	if dr.Door.wasUpdatedAt(t) {
		d.Door = &dr.Door.Value
		m.Door = &ts
		su = true
	}
	if dr.EC.wasUpdatedAt(t) {
		d.EC = &dr.EC.Value
		m.EC = &ts
		su = true
	}
	if dr.FirmwareNCU.wasUpdatedAt(t) {
		d.FirmwareNCU = &dr.FirmwareNCU.Value
		m.FirmwareNCU = &ts
	}
	if dr.HumidA.wasUpdatedAt(t) {
		d.HumidA = &dr.HumidA.Value
		m.HumidA = &ts
		su = true
	}
	if dr.HumidB.wasUpdatedAt(t) {
		d.HumidB = &dr.HumidB.Value
		m.HumidB = &ts
		su = true
	}
	if dr.LightA.wasUpdatedAt(t) {
		d.LightA = &dr.LightA.Value
		m.LightA = &ts
		su = true
	}
	if dr.LightB.wasUpdatedAt(t) {
		d.LightB = &dr.LightB.Value
		m.LightB = &ts
		su = true
	}
	if dr.Mode.wasUpdatedAt(t) {
		d.Mode = &dr.Mode.Value
		m.Mode = &ts
		su = true
	}
	if dr.RecipeID.wasUpdatedAt(t) {
		d.RecipeID = &dr.RecipeID.Value
		m.RecipeID = &ts
	}
	if dr.TankLevel.wasUpdatedAt(t) {
		d.TankLevel = &dr.TankLevel.Value
		m.TankLevel = &ts
		su = true
	}
	if dr.TankLevelRaw.wasUpdatedAt(t) {
		d.TankLevelRaw = &dr.TankLevelRaw.Value
		m.TankLevelRaw = &ts
	}
	if dr.TempA.wasUpdatedAt(t) {
		d.TempA = &dr.TempA.Value
		m.TempA = &ts
		su = true
	}
	if dr.TempB.wasUpdatedAt(t) {
		d.TempB = &dr.TempB.Value
		m.TempB = &ts
		su = true
	}
	if dr.TempTank.wasUpdatedAt(t) {
		d.TempTank = &dr.TempTank.Value
		m.TempTank = &ts
		su = true
	}
	if dr.TotalOffset.wasUpdatedAt(t) {
		d.TotalOffset = &dr.TotalOffset.Value
		m.TotalOffset = &ts
	}
	if dr.Valve.wasUpdatedAt(t) {
		d.Valve = &dr.Valve.Value
		m.Valve = &ts
		su = true
	}
	if dr.WifiLevel.wasUpdatedAt(t) {
		d.WifiLevel = &dr.WifiLevel.Value
		m.WifiLevel = &ts
	}
	return su
}

// Example: {"version":944757,"timestamp":1687710613,"state":{"recipe_id":1687710613},"metadata":{"recipe_id":{"timestamp":1687710613}}}
type msgAWSShadowUpdateDeltaReply struct {
	Version   int                        `json:"version"`
	Timestamp int                        `json:"timestamp"`
	State     msgAWSShadowUpdateData     `json:"state"`
	Metadata  msgAWSShadowUpdateMetadata `json:"metadata"`
}

func (m *msgAWSShadowUpdateDeltaReply) topic() string {
	return MQTT_TOPIC_AWS_UPDATE_DELTA
}

func (d *Device) getAWSShadowUpdateDeltaReply(deltaT, t time.Time) msgReply {
	msg := msgAWSShadowUpdateDeltaReply{}
	s := &msg.State
	m := &msg.Metadata

	msg.Version = d.AWSVersion
	msg.Timestamp = int(t.Unix())
	// Status won't be updated, ignore return.
	_ = d.fillAWSUpdateDataMetadata(deltaT, s, m)
	return &msg
}
