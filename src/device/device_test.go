package device

import (
	"encoding/json"
	"github.com/lupguo/go-render/render"
	"testing"
	"time"
)

func TestGetAWSUpdateAcceptedReply(t *testing.T) {
	ts := time.Unix(1691777926, 0)
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
			want: `{"state":{"reported":{"cooling":true,"temp_a":22.31}},"metadata":{"reported":{"cooling":{"timestamp":1691777926},"temp_a":{"timestamp":1691777926}}},"version":1,"timestamp":1691777926,"clientToken":""}`,
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
