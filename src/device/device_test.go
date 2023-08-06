package device

import (
	"reflect"
	"testing"
)

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
