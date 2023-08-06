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
