package device

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	// Yes, this is only the _typical_ number of hours per day,
	// but DST is handled elsewhere (via total_offset), we don't
	// worry about it here.
	DayDuration       = time.Hour * 24
	CycleStartDaysAgo = 7
	RecipeVersion     = 7
)

var (
	ledsOff = [4]byte{0, 0, 0, 0}
)

type recipePeriod struct {
	Duration    int32
	LEDVals     [4]byte
	TempTarget  int16
	WaterTarget int16
	WaterDelay  int16
}

type recipeBlock struct {
	Periods  []recipePeriod
	RepCount byte
}

type recipeLayer struct {
	Blocks []recipeBlock
}

type recipe struct {
	ID         int32
	CycleStart int32
	Layers     []recipeLayer
}

// Returns a recipe with the specified values on both layers. The
// recipe will have an ID of the specified time and a cycleStart 7
// days earlier, clamped to midnight UTC. (I can't discern a reason
// behind the cycle start dates from captured MQTT messages, but I
// only have two examples. Those examples are exactly 94 days apart,
// at Thu Dec 29th 2022 and Sun Apr 02nd 2023, but 94*4 is not a
// year.)
func CreateRecipe(asOf time.Time, ledVals []byte, tempTargetDay float64, tempTargetNight float64, waterTarget int, waterDelay time.Duration, dayLength time.Duration, layerAActive bool, layerBActive bool) (*recipe, error) {

	if len(ledVals) != 4 {
		return nil, fmt.Errorf("wrong ledVals length, want 4, got %d", len(ledVals))
	}

	r := recipe{}
	r.ID = int32(asOf.Unix())
	r.CycleStart = int32(asOf.AddDate(0, 0, -CycleStartDaysAgo).Truncate(DayDuration).Unix())

	dayLenSec := int32(dayLength / time.Second)
	nightLenSec := int32((DayDuration - dayLength) / time.Second)
	arrLEDVals := [4]byte{ledVals[0], ledVals[1], ledVals[2], ledVals[3]}
	i16TempDay := int16(tempTargetDay * 100)
	i16TempNight := int16(tempTargetNight * 100)
	i16WaterTarget := int16(waterTarget)
	i16WaterDelay := int16(waterDelay / time.Second)

	skipPeriod := recipePeriod{
		Duration:    int32(DayDuration / time.Second),
		LEDVals:     ledsOff,
		TempTarget:  i16TempDay,
		WaterTarget: i16WaterTarget,
		WaterDelay:  -1,
	}
	skipBlock := recipeBlock{
		Periods:  []recipePeriod{skipPeriod},
		RepCount: CycleStartDaysAgo - 1,
	}
	inactiveBlock := recipeBlock{
		Periods:  []recipePeriod{skipPeriod},
		RepCount: 100, // Unclear why there should even be a limit
	}

	dayPeriod := recipePeriod{
		Duration:    dayLenSec,
		LEDVals:     arrLEDVals,
		TempTarget:  i16TempDay,
		WaterTarget: i16WaterTarget,
		WaterDelay:  i16WaterDelay,
	}
	nightPeriod := recipePeriod{
		Duration:    nightLenSec,
		LEDVals:     ledsOff,
		TempTarget:  i16TempNight,
		WaterTarget: 0,
		WaterDelay:  i16WaterDelay,
	}
	dayNightBlock := recipeBlock{
		Periods: []recipePeriod{
			dayPeriod,
			nightPeriod,
		},
		RepCount: 100, // Unclear why there should even be a limit
	}

	activeLayer := recipeLayer{
		Blocks: []recipeBlock{
			skipBlock,
			dayNightBlock,
		},
	}
	inactiveLayer := recipeLayer{
		Blocks: []recipeBlock{
			inactiveBlock,
		},
	}
	var layerA, layerB recipeLayer
	if layerAActive {
		layerA = activeLayer
	} else {
		layerA = inactiveLayer
	}
	if layerBActive {
		layerB = activeLayer
	} else {
		layerB = inactiveLayer
	}
	emptyLayer := recipeLayer{
		Blocks: []recipeBlock{},
	}
	r.Layers = []recipeLayer{
		layerA,
		layerB,
		emptyLayer,
	}
	return &r, nil
}

func (p *recipePeriod) Marshal(buf *bytes.Buffer) error {
	err := binary.Write(buf, binary.LittleEndian, p.Duration)
	if err != nil {
		return fmt.Errorf("failed writing duration %d: %w", p.Duration, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.LEDVals)
	if err != nil {
		return fmt.Errorf("failed writing ledVals {%d,%d,%d,%d}: %w", p.LEDVals[0], p.LEDVals[1], p.LEDVals[2], p.LEDVals[3], err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.TempTarget)
	if err != nil {
		return fmt.Errorf("failed writing temp target %d: %w", p.TempTarget, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.WaterTarget)
	if err != nil {
		return fmt.Errorf("failed writing water target %d: %w", p.WaterTarget, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.WaterDelay)
	if err != nil {
		return fmt.Errorf("failed writing water delay %d: %w", p.WaterDelay, err)
	}

	return nil
}

func (blk *recipeBlock) Marshal(buf *bytes.Buffer) error {
	for i, p := range blk.Periods {
		err := p.Marshal(buf)
		if err != nil {
			return fmt.Errorf("failed marshalling period %d: %w", i, err)
		}
	}
	return nil
}

func (l *recipeLayer) MarshalHeader(buf *bytes.Buffer) error {
	for i, blk := range l.Blocks {
		b := byte(len(blk.Periods))
		err := binary.Write(buf, binary.LittleEndian, b)
		if err != nil {
			return fmt.Errorf("failed writing block %d period len %d: %w", i, b, err)
		}
		err = binary.Write(buf, binary.LittleEndian, blk.RepCount)
		if err != nil {
			return fmt.Errorf("failed writing block %d repCount %d: %w", i, blk.RepCount, err)
		}
	}
	return nil
}

func (l *recipeLayer) MarshalContent(buf *bytes.Buffer) error {
	for i, blk := range l.Blocks {
		err := blk.Marshal(buf)
		if err != nil {
			return fmt.Errorf("failed writing block %d content: %w", i, err)
		}
	}
	return nil
}

func (r *recipe) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, r.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to write ID %d: %w", r.ID, err)
	}
	err = binary.Write(buf, binary.LittleEndian, r.CycleStart)
	if err != nil {
		return nil, fmt.Errorf("failed to write cycle start %d: %w", r.CycleStart, err)
	}
	b := byte(len(r.Layers) - 1)
	err = binary.Write(buf, binary.LittleEndian, b)
	if err != nil {
		return nil, fmt.Errorf("failed to write layers len %d: %w", b, err)
	}
	b = byte(RecipeVersion)
	err = binary.Write(buf, binary.LittleEndian, b)
	if err != nil {
		return nil, fmt.Errorf("failed to write recipe version %d: %w", b, err)
	}
	for i, l := range r.Layers {
		b = byte(len(l.Blocks))
		err = binary.Write(buf, binary.LittleEndian, b)
		if err != nil {
			return nil, fmt.Errorf("failed to write layer %d block len %d: %w", i, b, err)
		}
	}
	for i, l := range r.Layers {
		err = l.MarshalHeader(buf)
		if err != nil {
			return nil, fmt.Errorf("failed writing layer %d header: %w", i, err)
		}
	}
	for i, l := range r.Layers {
		err = l.MarshalContent(buf)
		if err != nil {
			return nil, fmt.Errorf("failed writing layer %d content: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}
