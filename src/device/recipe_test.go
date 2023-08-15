package device

import (
	"encoding/hex"
	"github.com/lupguo/go-render/render"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreateRecipe(t *testing.T) {
	ts := time.Unix(1691777926, 0)
	tests := []struct {
		asOf            time.Time
		ledVals         []byte
		tempTargetDay   float64
		tempTargetNight float64
		waterTarget     int
		waterDelay      time.Duration
		dayLength       time.Duration
		wantError       bool
	}{
		{
			// Should all be valid, no error
			asOf:            ts,
			ledVals:         []byte{1, 2, 3, 4},
			tempTargetDay:   23.0,
			tempTargetNight: 20.0,
			waterTarget:     70,
			waterDelay:      time.Hour * 8,
			dayLength:       time.Hour * 15,
			wantError:       false,
		}, {
			// ledVals too long
			asOf:            ts,
			ledVals:         []byte{0, 1, 2, 3, 4},
			tempTargetDay:   23.0,
			tempTargetNight: 20.0,
			waterTarget:     70,
			waterDelay:      time.Hour * 8,
			dayLength:       time.Hour * 15,
			wantError:       true,
		}, {
			// ledVals too short
			asOf:            ts,
			ledVals:         []byte{1, 2, 3},
			tempTargetDay:   23.0,
			tempTargetNight: 20.0,
			waterTarget:     70,
			waterDelay:      time.Hour * 8,
			dayLength:       time.Hour * 15,
			wantError:       true,
		},
	}
	for _, tc := range tests {
		_, err := CreateRecipe(tc.asOf, tc.ledVals, tc.tempTargetDay, tc.tempTargetNight, tc.waterTarget, tc.waterDelay, tc.dayLength)
		gotErr := err != nil
		if gotErr != tc.wantError {
			t.Errorf("CreateRecipe, incorrect error status, got error %v, wantError %v", err, tc.wantError)
		}
	}
}

func TestMarshalRecipe(t *testing.T) {
	ts := time.Unix(1691777926, 0)
	tests := []struct {
		asOf            time.Time
		ledVals         []byte
		tempTargetDay   float64
		tempTargetNight float64
		waterTarget     int
		waterDelay      time.Duration
		dayLength       time.Duration
		want            string
	}{
		{
			// Should all be valid, no error
			asOf:            ts,
			ledVals:         []byte{1, 2, 3, 4},
			tempTargetDay:   23.0,
			tempTargetNight: 20.0,
			waterTarget:     70,
			waterDelay:      time.Hour * 8,
			dayLength:       time.Hour * 15,
			want: `86 7b d6 64 80 3f cc 64  02 07 02 02 00 01 06 02` +
				`64 01 06 02 64 80 51 01  00 00 00 00 00 fc 08 46` +
				`00 ff ff f0 d2 00 00 01  02 03 04 fc 08 46 00 80` +
				`70 90 7e 00 00 00 00 00  00 d0 07 00 00 80 70 80` +
				`51 01 00 00 00 00 00 fc  08 46 00 ff ff f0 d2 00` +
				`00 01 02 03 04 fc 08 46  00 80 70 90 7e 00 00 00` +
				`00 00 00 d0 07 00 00 80  70`,
		},
	}
	for _, tc := range tests {
		r, err := CreateRecipe(tc.asOf, tc.ledVals, tc.tempTargetDay, tc.tempTargetNight, tc.waterTarget, tc.waterDelay, tc.dayLength)
		if err != nil {
			t.Fatalf("CreateRecipe error: %v", err)
		}
		got, err := r.Marshal()
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		wantBytes, err := hex.DecodeString(strings.ReplaceAll(tc.want, " ", ""))
		if err != nil {
			t.Fatalf("Couldn't decode tc.want: %v", err)
		}
		if !reflect.DeepEqual(got, wantBytes) {
			t.Errorf("Marshalling '%s',\ngot:\n%s\nwant:\n%s", render.Render(r), hex.Dump(got), hex.Dump(wantBytes))
		}
	}
}
