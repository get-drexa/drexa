package nullable

import (
	"bytes"
	"encoding/json"
	"time"
)

// Time tracks whether a JSON field was set and stores an optional time value.
type Time struct {
	Time *time.Time
	Set  bool
}

func (nt *Time) UnmarshalJSON(b []byte) error {
	nt.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		nt.Time = nil
		return nil
	}

	var t time.Time
	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}
	nt.Time = &t
	return nil
}
