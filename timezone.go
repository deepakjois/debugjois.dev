package main

import (
	"time"

	"github.com/matthalp/go-meridian/cet"
)

// AppTimezone represents the application's configured timezone.
// To change the timezone, update this type alias and the corresponding import.
// Available timezones: utc, et, pt, ct, mt, cst, jst, ist, hkt, sgt, gmt, cet, brt, aest
type AppTimezone = cet.Time

// Now returns the current time in the application's configured timezone.
func Now() AppTimezone {
	return cet.Now()
}

// ParseDate parses a date string in "YYYY-MM-DD" format in the application's timezone.
func ParseDate(dateStr string) (AppTimezone, error) {
	return cet.Parse("2006-01-02", dateStr)
}

// DateInAppTimezone creates a date in the application's timezone.
func DateInAppTimezone(year int, month time.Month, day, hour, min, sec, nsec int) AppTimezone {
	return cet.Date(year, month, day, hour, min, sec, nsec)
}

// TodayString returns today's date in "YYYY-MM-DD" format in the application's timezone.
func TodayString() string {
	return Now().Format("2006-01-02")
}
