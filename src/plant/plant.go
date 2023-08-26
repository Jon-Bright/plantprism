package plant

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

type PlantID int
type language string
type plantDuration time.Duration

type Plant struct {
	Names map[language]string

	// All of these durations are measured from planting
	Germination plantDuration
	HarvestFrom plantDuration
	HarvestBy   plantDuration
}

var (
	weekDayRe = regexp.MustCompile(`^(?:([0-9]+)w)?(?:([0-9]+)d)?(?:([0-9]+)h)?`)
	plants    map[PlantID]Plant
)

func parseWeeksDays(wd string) (string, error) {
	matches := weekDayRe.FindStringSubmatch(wd)
	if matches == nil {
		return wd, nil
	}
	hours := 0
	if matches[1] != "" {
		weeks, err := strconv.Atoi(matches[1])
		if err != nil {
			return "", fmt.Errorf("failed to parse weeks '%s': %w", matches[1], err)
		}
		hours += 168 * weeks // 168 hours in a common week
	}
	if matches[2] != "" {
		days, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", fmt.Errorf("failed to parse days '%s': %w", matches[2], err)
		}
		hours += 24 * days // 24 hours in a common day
	}
	if matches[3] != "" {
		sHours, err := strconv.Atoi(matches[3])
		if err != nil {
			return "", fmt.Errorf("failed to parse hours '%s': %w", matches[3], err)
		}
		hours += sHours
	}
	if hours == 0 {
		return wd, nil
	}
	return fmt.Sprintf("%dh%s", hours, wd[len(matches[0]):]), nil
}

func (pd *plantDuration) UnmarshalJSON(b []byte) error {
	var v any
	err := json.Unmarshal(b, &v)
	if err != nil {
		return fmt.Errorf("failed duration unmarshal: %w", err)
	}
	switch value := v.(type) {
	case string:
		p, err := parseWeeksDays(value)
		if err != nil {
			return fmt.Errorf("failed duration weeks/days conversion for '%s': %w", value, err)
		}
		d, err := time.ParseDuration(p)
		if err != nil {
			return fmt.Errorf("failed duration parse for '%s': %w", p, err)
		}
		*pd = plantDuration(d)
	default:
		return fmt.Errorf("invalid duration type: %v", v)
	}
	return nil
}

func LoadPlants() error {
	m, err := os.ReadFile("plants.json")
	if err != nil {
		return fmt.Errorf("failed to read plants.json: %w", err)
	}
	err = json.Unmarshal(m, &plants)
	if err != nil {
		return fmt.Errorf("failed unmarshalling plant JSON '%s': %w", string(m), err)
	}
	return nil
}

func Get(id PlantID) (*Plant, error) {
	p, ok := plants[id]
	if !ok {
		return nil, fmt.Errorf("PlantID %d not found", id)
	}
	return &p, nil
}

func GetDB() any {
	return &plants
}
