package main

import (
	"testing"
)

func TestParseWeeksDays(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "1w",
			want: "168h",
		}, {
			in:   "1w1d",
			want: "192h",
		}, {
			in:   "1w1d1h",
			want: "193h",
		}, {
			in:   "1w1d1h1m1s",
			want: "193h1m1s",
		}, {
			in:   "1d1h1m1s",
			want: "25h1m1s",
		}, {
			in:   "1h1m1s",
			want: "1h1m1s",
		}, {
			in:   "1m1s",
			want: "1m1s",
		},
	}
	for _, tc := range tests {
		got, err := parseWeeksDays(tc.in)
		if err != nil {
			t.Fatalf("error parsing '%s': %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("parsing '%s', got '%s', want '%s'", tc.in, got, tc.want)
		}
	}
}
