package device

import (
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
