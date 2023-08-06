package device

import (
	"reflect"
	"testing"
)

func TestParseAglMode(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAglMode
	}
	tests := []test{
		{
			// Normal mode change
			input:     `{"prev_mode": 0,"mode": 8, "trigger": 1}`,
			wantError: false,
			want:      &msgAglMode{PrevMode: ModeDefault, Mode: ModeCinema, Trigger: ModeTriggerDevice},
		},
		{
			// Triggered by app
			input:     `{"prev_mode": 8,"mode": 0, "trigger": 0}`,
			wantError: false,
			want:      &msgAglMode{PrevMode: ModeCinema, Mode: ModeDefault, Trigger: ModeTriggerApp},
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
		input string
		want  msgAglShadowUpdate
	}
	tests := []test{
		{
			input: `{"state":{"reported":{"connected": true}}}`,
			want:  msgAglShadowUpdate{State: msgAglShadowUpdateState{Reported: msgAglShadowUpdateReported{Connected: true, EC: 0}}},
		},
		{
			input: `{"state":{"reported":{"ec": 1306}}}`,
			want:  msgAglShadowUpdate{State: msgAglShadowUpdateState{Reported: msgAglShadowUpdateReported{Connected: false, EC: 1306}}},
		},
	}
	for _, tc := range tests {
		msg := msgUnparsed{"", "", []byte(tc.input)}
		got, err := parseAglShadowUpdate(&msg)
		if err != nil {
			t.Fatalf("error on parsing '%s': %v", tc.input, err)
		}
		if !reflect.DeepEqual(tc.want, *got) {
			t.Errorf("want: %+v, got: %+v", tc.want, *got)
		}
	}
}

func TestParseAWSShadowGet(t *testing.T) {
	type test struct {
		input     string
		wantError bool
		want      *msgAWSShadowGet
	}
	tests := []test{
		{
			input:     `{"clientToken":"5975bc44"}`,
			wantError: false,
			want:      &msgAWSShadowGet{ClientToken: "5975bc44"},
		},
		{
			input:     `{"clientToken":""}`,
			wantError: true,
			want:      nil,
		},
		{
			input:     `{"clientToken":"dead"}`,
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
