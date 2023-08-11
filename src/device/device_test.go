package device

import (
	"encoding/json"
	"github.com/lupguo/go-render/render"
	"testing"
	"time"
)

func TestGetAWSUpdateAcceptedReply(t *testing.T) {
	ts := time.Unix(1691777926, 0)
	tsOld := time.Unix(1691777920, 0)
	tests := []struct {
		d    Device
		want string
	}{
		{
			d: Device{
				cooling:  true,
				coolingT: ts,
				tempA:    22.31,
				tempAT:   ts,
			},
			want: `{"state":{"reported":` +
				`{"cooling":true,"temp_a":22.31}},` +
				`"metadata":{"reported":` +
				`{"cooling":{"timestamp":1691777926},` +
				`"temp_a":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926,"clientToken":""}`,
		}, {
			d: Device{
				cooling:  true,
				coolingT: ts,
				tempA:    22.31,
				tempAT:   tsOld,
			},
			want: `{"state":{"reported":` +
				`{"cooling":true}},` +
				`"metadata":{"reported":` +
				`{"cooling":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926,"clientToken":""}`,
		}, {
			d: Device{
				cooling:       true,
				coolingT:      ts,
				door:          true,
				doorT:         ts,
				firmwareNCU:   int(tsOld.Unix()),
				firmwareNCUT:  ts,
				humidA:        80,
				humidAT:       ts,
				humidB:        70,
				humidBT:       ts,
				lightA:        true,
				lightAT:       ts,
				lightB:        true,
				lightBT:       ts,
				recipeID:      int(tsOld.Unix()),
				recipeIDT:     ts,
				tankLevel:     2,
				tankLevelT:    ts,
				tankLevelRaw:  2,
				tankLevelRawT: ts,
				tempA:         22.31,
				tempAT:        ts,
				tempB:         23.45,
				tempBT:        ts,
				tempTank:      24.56,
				tempTankT:     ts,
				totalOffset:   68040,
				totalOffsetT:  ts,
				valve:         2,
				valveT:        ts,
				wifiLevel:     2,
				wifiLevelT:    ts,
			},
			want: `{"state":{"reported":` +
				`{"cooling":true,"door":true,"firmware_ncu":1691777920,` +
				`"humid_a":80,"humid_b":70,"light_a":true,"light_b":true,` +
				`"recipe_id":1691777920,"tank_level":2,"tank_level_raw":2,` +
				`"temp_a":22.31,"temp_b":23.45,"temp_tank":24.56,` +
				`"total_offset":68040,"valve":2,"wifi_level":2}},` +
				`"metadata":{"reported":` +
				`{"cooling":{"timestamp":1691777926},"door":{"timestamp":1691777926},` +
				`"firmware_ncu":{"timestamp":1691777926},` +
				`"humid_a":{"timestamp":1691777926},"humid_b":{"timestamp":1691777926},` +
				`"light_a":{"timestamp":1691777926},"light_b":{"timestamp":1691777926},` +
				`"recipe_id":{"timestamp":1691777926},` +
				`"tank_level":{"timestamp":1691777926},` +
				`"tank_level_raw":{"timestamp":1691777926},` +
				`"temp_a":{"timestamp":1691777926},"temp_b":{"timestamp":1691777926},` +
				`"temp_tank":{"timestamp":1691777926},` +
				`"total_offset":{"timestamp":1691777926},` +
				`"valve":{"timestamp":1691777926},"wifi_level":{"timestamp":1691777926}` +
				`}},"version":1,"timestamp":1691777926,"clientToken":""}`,
		},
	}
	for _, tc := range tests {
		reply := tc.d.getAWSUpdateAcceptedReply(ts)
		b, err := json.Marshal(reply)
		if err != nil {
			t.Fatalf("shadow update accepted reply for device '%s', ts %d, error %v", render.Render(tc.d), ts.Unix(), err)
		}
		got := string(b)
		if got != tc.want {
			t.Errorf("shadow update accepted reply for device '%s', ts %d, got '%s', want '%s'", render.Render(tc.d), ts.Unix(), got, tc.want)
		}

	}
}
