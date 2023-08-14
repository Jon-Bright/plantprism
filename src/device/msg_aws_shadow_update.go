package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Example: {"clientToken":"5975bc44","state":{"reported":{"humid_b":75,"temp_a":22.99,"temp_b":24.19}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_a":22.69,"firmware_ncu":1667466618,"door":false,"cooling":true,"total_offset":69299,"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"wifi_level":0}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"temp_tank":28.34}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":false,"light_b":false}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"light_a":true,"light_b":true}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"tank_level_raw":2}}}
// Example: {"clientToken":"5975bc44","state":{"reported":{"door":true}}}

type msgAWSShadowUpdateReported struct {
	// `Connected` and `EC` shouldn't be reported by $aws/.../update,
	// but they're here because they appear in .../update/accepted
	Connected    *bool       `json:"connected,omitempty"`
	Cooling      *bool       `json:"cooling,omitempty"`
	Door         *bool       `json:"door,omitempty"`
	EC           *int        `json:"ec,omitempty"`
	FirmwareNCU  *int        `json:"firmware_ncu,omitempty"`
	HumidA       *int        `json:"humid_a,omitempty"`
	HumidB       *int        `json:"humid_b,omitempty"`
	LightA       *bool       `json:"light_a,omitempty"`
	LightB       *bool       `json:"light_b,omitempty"`
	RecipeID     *int        `json:"recipe_id,omitempty"`
	TankLevel    *int        `json:"tank_level,omitempty"`
	TankLevelRaw *int        `json:"tank_level_raw,omitempty"`
	TempA        *float64    `json:"temp_a,omitempty"`
	TempB        *float64    `json:"temp_b,omitempty"`
	TempTank     *float64    `json:"temp_tank,omitempty"`
	TotalOffset  *int        `json:"total_offset,omitempty"`
	Valve        *ValveState `json:"valve,omitempty"`
	WifiLevel    *int        `json:"wifi_level,omitempty"`
}
type msgAWSShadowUpdateState struct {
	Reported msgAWSShadowUpdateReported `json:"reported"`
}
type msgAWSShadowUpdate struct {
	ClientToken *string
	State       msgAWSShadowUpdateState
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

func (d *Device) processAWSShadowUpdate(msg *msgUnparsed) ([]msgReply, error) {
	m, err := parseAWSShadowUpdate(msg)
	if err != nil {
		return nil, err
	}
	if *m.ClientToken != d.clientToken {
		return nil, fmt.Errorf("clientToken '%s' received, but device clientToken is '%s'", *m.ClientToken, d.clientToken)
	}
	t := time.Now()
	r := m.State.Reported
	if r.Connected != nil {
		return nil, errors.New("unexpected Connected reported in AWS update")
	}
	if r.EC != nil {
		return nil, errors.New("unexpected EC reported in AWS update")
	}
	if r.Cooling != nil {
		d.cooling = *r.Cooling
		d.coolingT = t
	}
	if r.Door != nil {
		d.door = *r.Door
		d.doorT = t
	}
	if r.FirmwareNCU != nil {
		d.firmwareNCU = *r.FirmwareNCU
		d.firmwareNCUT = t
	}
	if r.HumidA != nil {
		d.humidA = *r.HumidA
		d.humidAT = t
	}
	if r.HumidB != nil {
		d.humidB = *r.HumidB
		d.humidBT = t
	}
	if r.LightA != nil {
		d.lightA = *r.LightA
		d.lightAT = t
	}
	if r.LightB != nil {
		d.lightB = *r.LightB
		d.lightBT = t
	}
	if r.RecipeID != nil {
		d.recipeID = *r.RecipeID
		d.recipeIDT = t
	}
	if r.TankLevel != nil {
		d.tankLevel = *r.TankLevel
		d.tankLevelT = t
	}
	if r.TankLevelRaw != nil {
		d.tankLevelRaw = *r.TankLevelRaw
		d.tankLevelRawT = t
	}
	if r.TempA != nil {
		d.tempA = *r.TempA
		d.tempAT = t
	}
	if r.TempB != nil {
		d.tempB = *r.TempB
		d.tempBT = t
	}
	if r.TempTank != nil {
		d.tempTank = *r.TempTank
		d.tempTankT = t
	}
	if r.TotalOffset != nil {
		d.totalOffset = *r.TotalOffset
		d.totalOffsetT = t
	}
	if r.Valve != nil {
		d.valve = *r.Valve
		d.valveT = t
	}
	if r.WifiLevel != nil {
		d.wifiLevel = *r.WifiLevel
		d.wifiLevelT = t
	}
	reply := d.getAWSUpdateAcceptedReply(t, false)
	return []msgReply{reply}, nil
}

type msgAWSShadowUpdateAcceptedMetadataReported struct {
	Connected    *msgUpdTS `json:"connected,omitempty"`
	Cooling      *msgUpdTS `json:"cooling,omitempty"`
	Door         *msgUpdTS `json:"door,omitempty"`
	EC           *msgUpdTS `json:"ec,omitempty"`
	FirmwareNCU  *msgUpdTS `json:"firmware_ncu,omitempty"`
	HumidA       *msgUpdTS `json:"humid_a,omitempty"`
	HumidB       *msgUpdTS `json:"humid_b,omitempty"`
	LightA       *msgUpdTS `json:"light_a,omitempty"`
	LightB       *msgUpdTS `json:"light_b,omitempty"`
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
	Reported msgAWSShadowUpdateAcceptedMetadataReported `json:"reported"`
}
type msgAWSShadowUpdateAccepted struct {
	State       msgAWSShadowUpdateState            `json:"state"`
	Metadata    msgAWSShadowUpdateAcceptedMetadata `json:"metadata"`
	Version     int                                `json:"version"`
	Timestamp   int                                `json:"timestamp"`
	ClientToken string                             `json:"clientToken,omitempty"`
}

func (m *msgAWSShadowUpdateAccepted) topic() string {
	return MQTT_TOPIC_AWS_UPDATE_ACCEPTED
}

// Construct a reply featuring all values reported at the given timestamp,
// along with metadata for each of those values with the timestamp.
// /shadow/update to agl/prod _also_ triggers AWS updates, but these come
// without a client token (possibly because from AWS's POV, they're coming
// from a different client?).
func (d *Device) getAWSUpdateAcceptedReply(t time.Time, omitClientToken bool) msgReply {
	msg := msgAWSShadowUpdateAccepted{}
	r := &msg.State.Reported
	m := &msg.Metadata.Reported
	unix := int(t.Unix())

	d.awsVersion++
	msg.Version = d.awsVersion
	msg.Timestamp = unix
	if !omitClientToken {
		msg.ClientToken = d.clientToken
	}
	ts := msgUpdTS{unix}

	if d.connectedT == t {
		r.Connected = &d.connected
		m.Connected = &ts
	}
	if d.coolingT == t {
		r.Cooling = &d.cooling
		m.Cooling = &ts
	}
	if d.doorT == t {
		r.Door = &d.door
		m.Door = &ts
	}
	if d.ecT == t {
		r.EC = &d.ec
		m.EC = &ts
	}
	if d.firmwareNCUT == t {
		r.FirmwareNCU = &d.firmwareNCU
		m.FirmwareNCU = &ts
	}
	if d.humidAT == t {
		r.HumidA = &d.humidA
		m.HumidA = &ts
	}
	if d.humidBT == t {
		r.HumidB = &d.humidB
		m.HumidB = &ts
	}
	if d.lightAT == t {
		r.LightA = &d.lightA
		m.LightA = &ts
	}
	if d.lightBT == t {
		r.LightB = &d.lightB
		m.LightB = &ts
	}
	if d.recipeIDT == t {
		r.RecipeID = &d.recipeID
		m.RecipeID = &ts
	}
	if d.tankLevelT == t {
		r.TankLevel = &d.tankLevel
		m.TankLevel = &ts
	}
	if d.tankLevelRawT == t {
		r.TankLevelRaw = &d.tankLevelRaw
		m.TankLevelRaw = &ts
	}
	if d.tempAT == t {
		r.TempA = &d.tempA
		m.TempA = &ts
	}
	if d.tempBT == t {
		r.TempB = &d.tempB
		m.TempB = &ts
	}
	if d.tempTankT == t {
		r.TempTank = &d.tempTank
		m.TempTank = &ts
	}
	if d.totalOffsetT == t {
		r.TotalOffset = &d.totalOffset
		m.TotalOffset = &ts
	}
	if d.valveT == t {
		r.Valve = &d.valve
		m.Valve = &ts
	}
	if d.wifiLevelT == t {
		r.WifiLevel = &d.wifiLevel
		m.WifiLevel = &ts
	}
	return &msg
}
