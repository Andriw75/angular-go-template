package database

import (
	"time"
)

const timeLayout = "2006-01-02T15:04:05.000000000Z07:00"

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{timeLayout, time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func strPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func floatPtr(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intBool(n int64) bool { return n != 0 }
