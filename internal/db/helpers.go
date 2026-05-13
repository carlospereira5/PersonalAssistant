package db

import "time"

const timeFormat = time.RFC3339

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func parseTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(timeFormat, *s)
	if err != nil {
		return nil
	}
	return &t
}

func timeNow() time.Time {
	return time.Now()
}
