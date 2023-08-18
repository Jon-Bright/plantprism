package device

import (
	"github.com/lupguo/go-render/render"
	"reflect"
	"testing"
)

func TestPickyUnmarshal(t *testing.T) {
	type isb struct {
		I *int
		S *string
		B *bool
	}
	// All of these cases should return an error. We could test
	// working cases, but that's essentially all of the other
	// tests in this file.
	tests := []string{
		`{"i": "foo"}`,  // String instead of int
		`{"s": 23}`,     // int instead of string
		`{"b": tralse}`, // Not a bool
		`{"i": 1, "s": "foo", "b":true, "c":false}`, // Extra field
		`{"i": 1},{"i": 2}`,                         // More than one object
		`{"i": 1}{"i": 2}`,                          // More than one object
		`[{"i": 1},{"i": 2}]`,                       // More than one object
	}
	for _, tc := range tests {
		var x isb
		err := pickyUnmarshal([]byte(tc), &x)
		t.Logf("error %v", err)
		if err == nil {
			t.Errorf("no error for '%s'", tc)
		}
	}
}

func TestParseAglEventInfo(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglEventInfo
	}
	mcuModeState := "MCU_MODE_STATE"
	timestamp := 1687686053
	modeEco := "ECO_MODE"
	state := "0"
	layer := "APPLIANCE"
	tests := []test{
		{
			input:     `{"label":"MCU_MODE_STATE","timestamp":1687686053,"payload":{"mode":"ECO_MODE","state":"0","layer":"APPLIANCE"}}`,
			wantError: false,
			want:      &msgAglEventInfo{Label: &mcuModeState, Timestamp: &timestamp, Payload: msgAglEventInfoPayload{Mode: &modeEco, State: &state, Layer: &layer}},
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglEventInfo(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %+v, got: %+v", tc.want, got)
		}
	}
}

func TestParseAglEventWarning(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglEventWarning
	}
	ncuSysLog := "NCU_SYS_LOG"
	timestamp := 1687329836
	errorLog := `MGOS_SHADOW_UPDATE_REJECTED 400 Missing required node: state_timer: 0; retries: 0; buff: {'clientToken':'5975bc44','state':{'reported':`
	function := "aws_shadow_grp_handler"
	tests := []test{
		{
			input: `{"label":"NCU_SYS_LOG","timestamp":1687329836,"payload":{"error_log":"MGOS_SHADOW_UPDATE_REJECTED 400 Missing required node: state
timer: 0; retries: 0; buff: {'clientToken':'5975bc44','state':{'reported':","function_name":"aws_shadow_grp_handler"}}`,
			wantError: false,
			want:      &msgAglEventWarning{Label: &ncuSysLog, Timestamp: &timestamp, Payload: msgAglEventWarningPayload{ErrorLog: &errorLog, FunctionName: &function}},
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglEventWarning(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %s, got: %s", render.Render(tc.want), render.Render(got))
		}
	}
}

func TestParseAglMode(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglMode
	}
	modeDefault := ModeDefault
	modeCinema := ModeCinema
	modeTriggerDevice := ModeTriggerDevice
	modeTriggerApp := ModeTriggerApp
	tests := []test{
		{
			// Normal mode change
			input:     `{"prev_mode": 0,"mode": 8, "trigger": 1}`,
			wantError: false,
			want:      &msgAglMode{PrevMode: &modeDefault, Mode: &modeCinema, Trigger: &modeTriggerDevice},
		},
		{
			// Triggered by app
			input:     `{"prev_mode": 8,"mode": 0, "trigger": 0}`,
			wantError: false,
			want:      &msgAglMode{PrevMode: &modeCinema, Mode: &modeDefault, Trigger: &modeTriggerApp},
		},
		{
			// Invalid: not a change
			input:     `{"prev_mode": 0,"mode": 0, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid prev_mode
			input:     `{"prev_mode": -1,"mode": 0, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid prev_mode
			input:     `{"prev_mode": 9,"mode": 0, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid mode
			input:     `{"prev_mode": 0,"mode": -1, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid mode
			input:     `{"prev_mode": 0,"mode": 9, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// No prev_mode
			input:     `{"mode": -1, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// No mode
			input:     `{"prev_mode": 0, "trigger": 1}`,
			wantError: true,
			want:      nil,
		},
		{
			// No trigger
			input:     `{"prev_mode": 0, "mode":2}`,
			wantError: true,
			want:      nil,
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglMode(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %+v, got: %+v", tc.want, got)
		}
	}
}

func TestParseAglRecipeGet(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglRecipeGet
	}
	version7 := 7
	formatBinary := "binary"
	tests := []test{
		{
			input:     `{"version":7, "format": "binary" }`,
			wantError: false,
			want:      &msgAglRecipeGet{Version: &version7, Format: &formatBinary},
		}, {
			// Invalid: no version
			input:     `{"format": "binary" }`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: no format
			input:     `{"version":7 }`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: wrong version
			input:     `{"version":8, "format": "binary" }`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: wrong format
			input:     `{"version":8, "format": "yaml" }`,
			wantError: true,
			want:      nil,
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglRecipeGet(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %s, got: %s", render.Render(tc.want), render.Render(got))
		}
	}
}

func TestParseAglShadowUpdate(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglShadowUpdate
	}
	connectedTrue := true
	ec1306 := 1306
	tests := []test{
		{
			input:     `{"state":{"reported":{"connected": true}}}`,
			wantError: false,
			want:      &msgAglShadowUpdate{State: msgAglShadowUpdateState{Reported: msgAglShadowUpdateReported{Connected: &connectedTrue, EC: nil}}},
		},
		{
			input:     `{"state":{"reported":{"ec": 1306}}}`,
			wantError: false,
			want:      &msgAglShadowUpdate{State: msgAglShadowUpdateState{Reported: msgAglShadowUpdateReported{Connected: nil, EC: &ec1306}}},
		},
		{
			// Invalid: no fields set
			input:     `{"state":{"reported":{}}}`,
			wantError: true,
			want:      nil,
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglShadowUpdate(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %+v, got: %+v", tc.want, got)
		}
	}
}

func TestParseAWSShadowGet(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAWSShadowGet
	}
	clientToken := "5975bc44"
	tests := []test{
		{
			input:     `{"clientToken":"5975bc44"}`,
			wantError: false,
			want:      &msgAWSShadowGet{ClientToken: &clientToken},
		},
		{
			// Invalid: empty token
			input:     `{"clientToken":""}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid: token too short
			input:     `{"clientToken":"dead"}`,
			wantError: true,
			want:      nil,
		},
		{
			// Invalid: no token
			input:     `{}`,
			wantError: true,
			want:      nil,
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAWSShadowGet(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %+v, got: %+v", tc.want, got)
		}
	}
}

func TestParseAWSShadowUpdate(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAWSShadowUpdate
	}
	// We don't theoretically need all these values, but the input
	// messages below are from actual captures and I prefer not to
	// mess around with them, just copy/paste.
	clientToken := "5975bc44"
	bTrue := true
	bFalse := false
	firmware := 1667466618
	humid75 := 75
	offset69299 := 69299
	temp2269 := 22.69
	temp2299 := 22.99
	temp2419 := 24.19
	temp2834 := 28.34
	two := 2
	zero := 0
	tests := []test{
		{
			input:     `{"clientToken":"5975bc44","state":{"reported":{"humid_b":75,"temp_a":22.99,"temp_b":24.19}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						HumidB: &humid75,
						TempA:  &temp2299,
						TempB:  &temp2419,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"temp_a":22.69,"firmware_ncu":1667466618,"door":false,"cooling":true,"total_offset":69299,"light_a":false,"light_b":false}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						Cooling:     &bTrue,
						Door:        &bFalse,
						FirmwareNCU: &firmware,
						LightA:      &bFalse,
						LightB:      &bFalse,
						TempA:       &temp2269,
						TotalOffset: &offset69299,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"wifi_level":0}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						WifiLevel: &zero,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"temp_tank":28.34}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						TempTank: &temp2834,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"light_a":true,"light_b":true}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						LightA: &bTrue,
						LightB: &bTrue,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"tank_level_raw":2}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						TankLevelRaw: &two,
					},
				},
			},
		}, {
			input:     `{"clientToken":"5975bc44","state":{"reported":{"door":true}}}`,
			wantError: false,
			want: &msgAWSShadowUpdate{
				ClientToken: &clientToken,
				State: msgAWSShadowUpdateState{
					Reported: msgAWSShadowUpdateData{
						Door: &bTrue,
					},
				},
			},
		}, {
			// Invalid: empty token
			input:     `{"clientToken":"", "state":{"reported":{"door":true}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: token too short
			input:     `{"clientToken":"dead", "state":{"reported":{"door":true}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: no token
			input:     `{"state":{"reported":{"door":true}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: empty update
			input:     `{"clientToken":"5975bc44", "state":{"reported":{}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: old firmware
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"firmware_ncu":1660000000}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low humidity
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"humid_a":10}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high humidity
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"humid_a":101}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low humidity
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"humid_b":10}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high humidity
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"humid_b":101}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: old recipe
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"recipe_id":1680200000}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low tank level
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"tank_level":-1}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high tank level
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"tank_level":3}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low raw tank level
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"tank_level_raw":-1}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high raw tank level
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"tank_level_raw":3}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low temp A
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_a":9}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high temp A
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_a":41}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low temp B
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_b":9}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high temp B
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_b":41}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low temp tank
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_tank":9}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high temp tank
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"temp_tank":41}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: low offset
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"total_offset":-1}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: high offset
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"total_offset":86401}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: valve
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"valve":3}}}`,
			wantError: true,
			want:      nil,
		}, {
			// Invalid: wifi_level
			input:     `{"clientToken":"5975bc44", "state":{"reported":{"wifi_level":3}}}`,
			wantError: true,
			want:      nil,
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAWSShadowUpdate(&msg)
		if tc.wantError != (err != nil) {
			t.Fatalf("parsing '%s', wanted error %v, got %v", tc.input, tc.wantError, err)
		}
		if !tc.wantError && !reflect.DeepEqual(tc.want, got) {
			t.Errorf("want: %s, got: %s", render.Render(tc.want), render.Render(got))
		} else {
			t.Logf("err: %v", err)
		}
	}
}
