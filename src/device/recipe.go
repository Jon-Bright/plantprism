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
	duration    int32
	ledVals     [4]byte
	tempTarget  int16
	waterTarget int16
	waterDelay  int16
}

type recipeBlock struct {
	periods  []recipePeriod
	repCount byte
}

type recipeLayer struct {
	blocks []recipeBlock
}

type recipe struct {
	id         int32
	cycleStart int32
	layers     []recipeLayer
}

// Returns a recipe with the specified values on both layers. The
// recipe will have an ID of the specified time and a cycleStart 7
// days earlier, clamped to midnight UTC. (I can't discern a reason
// behind the cycle start dates from captured MQTT messages, but I
// only have two examples. Those examples are exactly 94 days apart,
// at Thu Dec 29th 2022 and Sun Apr 02nd 2023, but 94*4 is not a
// year.)
func CreateRecipe(asOf time.Time, ledVals []byte, tempTargetDay float64, tempTargetNight float64, waterTarget int, waterDelay time.Duration, dayLength time.Duration) (*recipe, error) {

	if len(ledVals) != 4 {
		return nil, fmt.Errorf("wrong ledVals length, want 4, got %d", len(ledVals))
	}

	r := recipe{}
	r.id = int32(asOf.Unix())
	r.cycleStart = int32(asOf.AddDate(0, 0, -CycleStartDaysAgo).Truncate(DayDuration).Unix())

	dayLenSec := int32(dayLength / time.Second)
	nightLenSec := int32((DayDuration - dayLength) / time.Second)
	arrLEDVals := [4]byte{ledVals[0], ledVals[1], ledVals[2], ledVals[3]}
	i16TempDay := int16(tempTargetDay * 100)
	i16TempNight := int16(tempTargetNight * 100)
	i16WaterTarget := int16(waterTarget)
	i16WaterDelay := int16(waterDelay / time.Second)

	skipPeriod := recipePeriod{
		duration:    int32(DayDuration / time.Second),
		ledVals:     ledsOff,
		tempTarget:  i16TempDay,
		waterTarget: i16WaterTarget,
		waterDelay:  -1,
	}
	skipBlock := recipeBlock{
		periods:  []recipePeriod{skipPeriod},
		repCount: CycleStartDaysAgo - 1,
	}

	dayPeriod := recipePeriod{
		duration:    dayLenSec,
		ledVals:     arrLEDVals,
		tempTarget:  i16TempDay,
		waterTarget: i16WaterTarget,
		waterDelay:  i16WaterDelay,
	}
	nightPeriod := recipePeriod{
		duration:    nightLenSec,
		ledVals:     ledsOff,
		tempTarget:  i16TempNight,
		waterTarget: 0,
		waterDelay:  i16WaterDelay,
	}
	dayNightBlock := recipeBlock{
		periods: []recipePeriod{
			dayPeriod,
			nightPeriod,
		},
		repCount: 100, // Unclear why there should even be a limit
	}

	filledLayer := recipeLayer{
		blocks: []recipeBlock{
			skipBlock,
			dayNightBlock,
		},
	}
	emptyLayer := recipeLayer{
		blocks: []recipeBlock{},
	}
	r.layers = []recipeLayer{
		filledLayer,
		filledLayer,
		emptyLayer,
	}
	return &r, nil
}

func (p *recipePeriod) Marshal(buf *bytes.Buffer) error {
	err := binary.Write(buf, binary.LittleEndian, p.duration)
	if err != nil {
		return fmt.Errorf("failed writing duration %d: %w", p.duration, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.ledVals)
	if err != nil {
		return fmt.Errorf("failed writing ledVals {%d,%d,%d,%d}: %w", p.ledVals[0], p.ledVals[1], p.ledVals[2], p.ledVals[3], err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.tempTarget)
	if err != nil {
		return fmt.Errorf("failed writing temp target %d: %w", p.tempTarget, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.waterTarget)
	if err != nil {
		return fmt.Errorf("failed writing water target %d: %w", p.waterTarget, err)
	}
	err = binary.Write(buf, binary.LittleEndian, p.waterDelay)
	if err != nil {
		return fmt.Errorf("failed writing water delay %d: %w", p.waterDelay, err)
	}

	return nil
}

func (blk *recipeBlock) Marshal(buf *bytes.Buffer) error {
	for i, p := range blk.periods {
		err := p.Marshal(buf)
		if err != nil {
			return fmt.Errorf("failed marshalling period %d: %w", i, err)
		}
	}
	return nil
}

func (l *recipeLayer) MarshalHeader(buf *bytes.Buffer) error {
	for i, blk := range l.blocks {
		b := byte(len(blk.periods))
		err := binary.Write(buf, binary.LittleEndian, b)
		if err != nil {
			return fmt.Errorf("failed writing block %d period len %d: %w", i, b, err)
		}
		err = binary.Write(buf, binary.LittleEndian, blk.repCount)
		if err != nil {
			return fmt.Errorf("failed writing block %d repCount %d: %w", i, blk.repCount, err)
		}
	}
	return nil
}

func (l *recipeLayer) MarshalContent(buf *bytes.Buffer) error {
	for i, blk := range l.blocks {
		err := blk.Marshal(buf)
		if err != nil {
			return fmt.Errorf("failed writing block %d content: %w", i, err)
		}
	}
	return nil
}

func (r *recipe) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, r.id)
	if err != nil {
		return nil, fmt.Errorf("failed to write ID %d: %w", r.id, err)
	}
	err = binary.Write(buf, binary.LittleEndian, r.cycleStart)
	if err != nil {
		return nil, fmt.Errorf("failed to write cycle start %d: %w", r.cycleStart, err)
	}
	b := byte(len(r.layers) - 1)
	err = binary.Write(buf, binary.LittleEndian, b)
	if err != nil {
		return nil, fmt.Errorf("failed to write layers len %d: %w", b, err)
	}
	b = byte(RecipeVersion)
	err = binary.Write(buf, binary.LittleEndian, b)
	if err != nil {
		return nil, fmt.Errorf("failed to write recipe version %d: %w", b, err)
	}
	for i, l := range r.layers {
		b = byte(len(l.blocks))
		err = binary.Write(buf, binary.LittleEndian, b)
		if err != nil {
			return nil, fmt.Errorf("failed to write layer %d block len %d: %w", i, b, err)
		}
	}
	for i, l := range r.layers {
		err = l.MarshalHeader(buf)
		if err != nil {
			return nil, fmt.Errorf("failed writing layer %d header: %w", i, err)
		}
	}
	for i, l := range r.layers {
		err = l.MarshalContent(buf)
		if err != nil {
			return nil, fmt.Errorf("failed writing layer %d content: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}
