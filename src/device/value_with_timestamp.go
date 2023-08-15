package device

import (
	"encoding/json"
	"time"
)

type valueWithTimestamp[T any] struct {
	Value T
	Time  time.Time
}

func (vwt *valueWithTimestamp[T]) update(v T, t time.Time) {
	vwt.Value = v
	vwt.Time = t
}

func (vwt valueWithTimestamp[T]) wasUpdatedAt(t time.Time) bool {
	return vwt.Time == t
}

func (vwt valueWithTimestamp[T]) MarshalJSON() ([]byte, error) {
	if vwt.Time.IsZero() {
		return []byte("{}"), nil
	}
	s := struct {
		Value T
		Time  time.Time
	}{
		vwt.Value,
		vwt.Time,
	}
	return json.Marshal(&s)
}
