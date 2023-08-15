package device

import (
	"encoding/json"
	"github.com/lupguo/go-render/render"
	"reflect"
	"testing"
	"time"
)

func TestMarshalUnmarshal(t *testing.T) {
	ts := time.Unix(1691777926, 0)
	tests := []struct {
		d    Device
		want string
	}{
		{
			d: Device{
				ID:          "a8d39911-7955-47d3-981b-fbd9d52f9221",
				ClientToken: "12345678",
				Reported: deviceReported{
					Cooling: valueWithTimestamp[bool]{true, ts},
					TempA:   valueWithTimestamp[float64]{22.31, ts},
				},
			},
			want: `{
  "ID": "a8d39911-7955-47d3-981b-fbd9d52f9221",
  "ClientToken": "12345678",
  "Reported": {
    "Mode": {},
    "Connected": {},
    "EC": {},
    "Cooling": {
      "Value": true,
      "Time": "2023-08-11T20:18:46+02:00"
    },
    "Door": {},
    "FirmwareNCU": {},
    "HumidA": {},
    "HumidB": {},
    "LightA": {},
    "LightB": {},
    "RecipeID": {},
    "TankLevel": {},
    "TankLevelRaw": {},
    "TempA": {
      "Value": 22.31,
      "Time": "2023-08-11T20:18:46+02:00"
    },
    "TempB": {},
    "TempTank": {},
    "TotalOffset": {},
    "Valve": {},
    "WifiLevel": {}
  }
}`,
		},
	}
	for _, tc := range tests {
		got, err := json.MarshalIndent(tc.d, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal device '%s': %v", render.Render(tc.d), err)
		}
		if !reflect.DeepEqual([]byte(tc.want), got) {
			t.Errorf("Marshalling device '%s', got:\n%s\nwant:\n%s", render.Render(tc.d), string(got), tc.want)
		}
		var gotDev Device
		err = pickyUnmarshal(got, &gotDev)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON '%s': %v", string(got), err)
		}
		if !reflect.DeepEqual(&tc.d, &gotDev) {
			t.Errorf("Unmarshalling, got:\n%s\nwant:\n%s", render.Render(gotDev), render.Render(tc.d))
		}
	}
}

func TestCalcTotalOffset(t *testing.T) {
	// For all of the "Observed from Plantcube comms" values
	// below: these aren't the exact values observed. When setting
	// sunrise via the app, the actual offset sent to the device
	// varies by plus/minus 60s. This might be to smear load, so a
	// whole bunch of devices don't wake up at the same second, or
	// it might just be a bug. Either way, we don't need it and
	// don't replicate it, so these times are rounded.
	tests := []struct {
		date     string
		timezone string
		sunrise  string
		want     int
	}{
		{
			// Observed from Plantcube comms
			date:     "2023-06-30",
			timezone: "Europe/Berlin",
			sunrise:  "07:00",
			want:     68400,
		}, {
			// Observed from Plantcube comms
			date:     "2023-06-30",
			timezone: "Europe/Berlin",
			sunrise:  "06:45",
			want:     69300,
		}, {
			// Observed from Plantcube comms
			date:     "2023-06-30",
			timezone: "Europe/Berlin",
			sunrise:  "06:30",
			want:     70200,
		}, {
			// Same as previous, but winter time
			date:     "2023-02-28",
			timezone: "Europe/Berlin",
			sunrise:  "06:30",
			want:     66600,
		}, {
			// Same as previous, but one timezone left
			date:     "2023-02-28",
			timezone: "Europe/London",
			sunrise:  "06:30",
			want:     63000,
		}, {
			// Same as previous, but rightmost timezone
			date:     "2023-02-28",
			timezone: "Pacific/Kiritimati",
			sunrise:  "06:30",
			want:     27000,
		}, {
			// Same as previous, but leftmost timezone
			date:     "2023-02-28",
			timezone: "Etc/GMT+12",
			sunrise:  "06:30",
			want:     19800,
		}, {
			// Observed from Plantcube comms
			date:     "2023-08-14",
			timezone: "Europe/Berlin",
			sunrise:  "20:30",
			want:     19800,
		}, {
			// Observed from Plantcube comms
			date:     "2023-08-14",
			timezone: "Europe/Berlin",
			sunrise:  "23:30",
			want:     9000,
		}, {
			// Observed from Plantcube comms
			date:     "2023-08-14",
			timezone: "Europe/Berlin",
			sunrise:  "00:30",
			want:     5400,
		}, {
			// Observed from Plantcube comms
			date:     "2023-08-14",
			timezone: "Europe/Berlin",
			sunrise:  "02:30",
			want:     84600,
		},
	}
	for _, tc := range tests {
		sunriseD, err := parseSunriseToDuration(tc.sunrise)
		if err != nil {
			t.Fatal(err)
		}
		// This date constant could be time.DateOnly, but the
		// newest golang on RasPi doesn't have that yet.
		date, err := time.Parse("2006-01-02", tc.date)
		if err != nil {
			t.Fatal(err)
		}
		got, err := calcTotalOffset(tc.timezone, date, sunriseD)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("calcTotalOffset for tz '%s', sunrise '%s', got %d, want %d", tc.timezone, tc.sunrise, got, tc.want)
		}
	}
}

func TestGetAWSUpdateAcceptedReply(t *testing.T) {
	ts := time.Unix(1691777926, 0)
	tsOld := time.Unix(1691777920, 0)
	tests := []struct {
		d               Device
		omitClientToken bool
		want            string
	}{
		{
			// Simple update, two values, both with the current timestamp
			d: Device{
				ClientToken: "12345678",
				Reported: deviceReported{
					Cooling: valueWithTimestamp[bool]{true, ts},
					TempA:   valueWithTimestamp[float64]{22.31, ts},
				},
			},
			omitClientToken: false,
			want: `{"state":{"reported":` +
				`{"cooling":true,"temp_a":22.31}},` +
				`"metadata":{"reported":` +
				`{"cooling":{"timestamp":1691777926},` +
				`"temp_a":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926,"clientToken":"12345678"}`,
		}, {
			// Update with an old timestamp for one value, which should be omitted
			d: Device{
				ClientToken: "12345678",
				Reported: deviceReported{
					Cooling: valueWithTimestamp[bool]{true, ts},
					TempA:   valueWithTimestamp[float64]{22.31, tsOld},
				},
			},
			omitClientToken: false,
			want: `{"state":{"reported":` +
				`{"cooling":true}},` +
				`"metadata":{"reported":` +
				`{"cooling":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926,"clientToken":"12345678"}`,
		}, {
			// Complete update
			d: Device{
				ClientToken: "12345678",
				Reported: deviceReported{
					Connected:    valueWithTimestamp[bool]{true, ts},
					Cooling:      valueWithTimestamp[bool]{true, ts},
					Door:         valueWithTimestamp[bool]{true, ts},
					EC:           valueWithTimestamp[int]{1234, ts},
					FirmwareNCU:  valueWithTimestamp[int]{int(tsOld.Unix()), ts},
					HumidA:       valueWithTimestamp[int]{80, ts},
					HumidB:       valueWithTimestamp[int]{70, ts},
					LightA:       valueWithTimestamp[bool]{true, ts},
					LightB:       valueWithTimestamp[bool]{true, ts},
					RecipeID:     valueWithTimestamp[int]{int(tsOld.Unix()), ts},
					TankLevel:    valueWithTimestamp[int]{2, ts},
					TankLevelRaw: valueWithTimestamp[int]{2, ts},
					TempA:        valueWithTimestamp[float64]{22.31, ts},
					TempB:        valueWithTimestamp[float64]{23.45, ts},
					TempTank:     valueWithTimestamp[float64]{24.56, ts},
					TotalOffset:  valueWithTimestamp[int]{68040, ts},
					Valve:        valueWithTimestamp[ValveState]{ValveClosed, ts},
					WifiLevel:    valueWithTimestamp[int]{2, ts},
				},
			},
			want: `{"state":{"reported":` +
				`{"connected":true,"cooling":true,"door":true,"ec":1234,` +
				`"firmware_ncu":1691777920,"humid_a":80,"humid_b":70,"light_a":true,` +
				`"light_b":true,"recipe_id":1691777920,"tank_level":2,` +
				`"tank_level_raw":2,"temp_a":22.31,"temp_b":23.45,"temp_tank":24.56,` +
				`"total_offset":68040,"valve":4,"wifi_level":2}},` +
				`"metadata":{"reported":` +
				`{"connected":{"timestamp":1691777926},` +
				`"cooling":{"timestamp":1691777926},"door":{"timestamp":1691777926},` +
				`"ec":{"timestamp":1691777926},` +
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
				`}},"version":1,"timestamp":1691777926,"clientToken":"12345678"}`,
		}, {
			// agl/prod/.../shadow/update, no client token. Also two old values.
			d: Device{
				ClientToken: "12345678",
				Reported: deviceReported{
					Cooling: valueWithTimestamp[bool]{true, tsOld},
					EC:      valueWithTimestamp[int]{1234, ts},
					TempA:   valueWithTimestamp[float64]{22.31, tsOld},
				},
			},
			omitClientToken: true,
			want: `{"state":{"reported":` +
				`{"ec":1234}},` +
				`"metadata":{"reported":` +
				`{"ec":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926}`,
		}, {
			// agl/prod/.../mode, no client token. Also two old values.
			d: Device{
				ClientToken: "12345678",
				Reported: deviceReported{
					LightA:   valueWithTimestamp[bool]{true, tsOld},
					Mode:     valueWithTimestamp[DeviceMode]{ModeCinema, ts},
					TempTank: valueWithTimestamp[float64]{22.31, tsOld},
				},
			},
			omitClientToken: true,
			want: `{"state":{"reported":` +
				`{"mode":8}},` +
				`"metadata":{"reported":` +
				`{"mode":{"timestamp":1691777926}}},` +
				`"version":1,"timestamp":1691777926}`,
		},
	}
	for _, tc := range tests {
		reply := tc.d.getAWSUpdateAcceptedReply(ts, tc.omitClientToken)
		b, err := json.Marshal(reply)
		if err != nil {
			t.Fatalf("shadow update accepted reply for device '%s',\nts %d, error %v", render.Render(tc.d), ts.Unix(), err)
		}
		got := string(b)
		if got != tc.want {
			t.Errorf("shadow update accepted reply for device '%s',\nts %d,\ngot '%s',\nwant '%s'", render.Render(tc.d), ts.Unix(), got, tc.want)
		}

	}
}
