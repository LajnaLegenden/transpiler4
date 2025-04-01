package helpers

import "time"

// GetCurrentTimeMillis returns the current time in milliseconds since epoch
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
