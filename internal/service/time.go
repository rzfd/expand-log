package service

import "time"

var currentUTC = func() time.Time {
	return time.Now().UTC()
}
