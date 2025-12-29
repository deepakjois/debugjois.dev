package main

import (
	"testing"
	"time"
)

func TestLastSaturday(t *testing.T) {
	tests := []struct {
		name     string
		input    AppTimezone
		expected string // YYYY-MM-DD format
	}{
		{
			name:     "from Sunday returns previous Saturday",
			input:    DateInAppTimezone(2024, time.December, 29, 12, 0, 0, 0), // Sunday
			expected: "2024-12-28",
		},
		{
			name:     "from Monday returns previous Saturday",
			input:    DateInAppTimezone(2024, time.December, 30, 12, 0, 0, 0), // Monday
			expected: "2024-12-28",
		},
		{
			name:     "from Saturday returns Saturday one week ago",
			input:    DateInAppTimezone(2024, time.December, 28, 12, 0, 0, 0), // Saturday
			expected: "2024-12-21",
		},
		{
			name:     "from Friday returns previous Saturday",
			input:    DateInAppTimezone(2024, time.December, 27, 12, 0, 0, 0), // Friday
			expected: "2024-12-21",
		},
		{
			name:     "crossing year boundary from January",
			input:    DateInAppTimezone(2025, time.January, 2, 12, 0, 0, 0), // Thursday
			expected: "2024-12-28",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := lastSaturday(tc.input)
			got := result.Format("2006-01-02")
			if got != tc.expected {
				t.Errorf("lastSaturday(%s) = %s, want %s", tc.input.Format("2006-01-02"), got, tc.expected)
			}
			// Verify it's actually a Saturday
			if result.Weekday() != time.Saturday {
				t.Errorf("lastSaturday returned %s which is %s, not Saturday", got, result.Weekday())
			}
		})
	}
}

func TestCalculateNewsletterWeek(t *testing.T) {
	tests := []struct {
		name            string
		input           AppTimezone
		expectedStart   string // Sunday YYYY-MM-DD
		expectedEnd     string // Saturday YYYY-MM-DD
		expectedYear    int
		expectedWeekNum int
	}{
		{
			name:            "regular week in middle of year",
			input:           DateInAppTimezone(2024, time.June, 17, 12, 0, 0, 0), // Monday June 17
			expectedStart:   "2024-06-09",                                        // Sunday
			expectedEnd:     "2024-06-15",                                        // Saturday
			expectedYear:    2024,
			expectedWeekNum: 24, // Week 24 (Monday June 10 is in ISO week 24)
		},
		{
			name:            "called on Sunday gets previous week",
			input:           DateInAppTimezone(2024, time.June, 16, 12, 0, 0, 0), // Sunday June 16
			expectedStart:   "2024-06-09",                                        // Sunday
			expectedEnd:     "2024-06-15",                                        // Saturday
			expectedYear:    2024,
			expectedWeekNum: 24,
		},
		{
			name:            "called on Saturday gets week before last",
			input:           DateInAppTimezone(2024, time.June, 15, 12, 0, 0, 0), // Saturday June 15
			expectedStart:   "2024-06-02",                                        // Sunday
			expectedEnd:     "2024-06-08",                                        // Saturday
			expectedYear:    2024,
			expectedWeekNum: 23, // Week 23 (Monday June 3 is in ISO week 23)
		},
		// Year transition edge cases
		{
			name:            "year transition: called on Jan 1 2025 (Wednesday)",
			input:           DateInAppTimezone(2025, time.January, 1, 12, 0, 0, 0),
			expectedStart:   "2024-12-22", // Sunday
			expectedEnd:     "2024-12-28", // Saturday
			expectedYear:    2024,
			expectedWeekNum: 52, // Monday Dec 23 is in ISO week 52 of 2024
		},
		{
			name:            "year transition: called on Jan 5 2025 (Sunday)",
			input:           DateInAppTimezone(2025, time.January, 5, 12, 0, 0, 0),
			expectedStart:   "2024-12-29", // Sunday
			expectedEnd:     "2025-01-04", // Saturday (crosses year boundary)
			expectedYear:    2025,
			expectedWeekNum: 1, // Monday Dec 30 is in ISO week 1 of 2025
		},
		{
			name:            "year transition: called on Jan 6 2025 (Monday)",
			input:           DateInAppTimezone(2025, time.January, 6, 12, 0, 0, 0),
			expectedStart:   "2024-12-29", // Sunday
			expectedEnd:     "2025-01-04", // Saturday
			expectedYear:    2025,
			expectedWeekNum: 1,
		},
		{
			name:            "ISO week 1 starts in previous year (2020)",
			input:           DateInAppTimezone(2020, time.January, 5, 12, 0, 0, 0), // Sunday
			expectedStart:   "2019-12-29",                                          // Sunday
			expectedEnd:     "2020-01-04",                                          // Saturday
			expectedYear:    2020,
			expectedWeekNum: 1, // Monday Dec 30 2019 is ISO week 1 of 2020
		},
		{
			name:            "week 53 edge case (2020 has 53 weeks)",
			input:           DateInAppTimezone(2021, time.January, 3, 12, 0, 0, 0), // Sunday
			expectedStart:   "2020-12-27",                                          // Sunday
			expectedEnd:     "2021-01-02",                                          // Saturday
			expectedYear:    2020,
			expectedWeekNum: 53, // Monday Dec 28 2020 is ISO week 53 of 2020
		},
		{
			name:            "week 53 to week 1 transition (2021)",
			input:           DateInAppTimezone(2021, time.January, 10, 12, 0, 0, 0), // Sunday
			expectedStart:   "2021-01-03",                                           // Sunday
			expectedEnd:     "2021-01-09",                                           // Saturday
			expectedYear:    2021,
			expectedWeekNum: 1, // Monday Jan 4 2021 is ISO week 1 of 2021
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateNewsletterWeek(tc.input)

			gotStart := result.Start.Format("2006-01-02")
			if gotStart != tc.expectedStart {
				t.Errorf("Start = %s, want %s", gotStart, tc.expectedStart)
			}

			gotEnd := result.End.Format("2006-01-02")
			if gotEnd != tc.expectedEnd {
				t.Errorf("End = %s, want %s", gotEnd, tc.expectedEnd)
			}

			if result.Year != tc.expectedYear {
				t.Errorf("Year = %d, want %d", result.Year, tc.expectedYear)
			}

			if result.WeekNum != tc.expectedWeekNum {
				t.Errorf("WeekNum = %d, want %d", result.WeekNum, tc.expectedWeekNum)
			}

			// Verify Start is a Sunday
			if result.Start.Weekday() != time.Sunday {
				t.Errorf("Start (%s) is %s, want Sunday", gotStart, result.Start.Weekday())
			}

			// Verify End is a Saturday
			if result.End.Weekday() != time.Saturday {
				t.Errorf("End (%s) is %s, want Saturday", gotEnd, result.End.Weekday())
			}

			// Verify the period is exactly 6 days (Sunday to Saturday)
			diff := result.End.Sub(result.Start)
			expectedDiff := 6 * 24 * time.Hour
			if diff != expectedDiff {
				t.Errorf("Period duration = %v, want %v", diff, expectedDiff)
			}
		})
	}
}

func TestCalculateNewsletterWeek_WeekNumberFromMondayNotSunday(t *testing.T) {
	// This test specifically verifies that the week number is based on Monday,
	// not Sunday. This is important because the newsletter runs Sunday-Saturday,
	// and ISO weeks change on Monday.

	// Dec 29, 2024 is a Sunday - it's in ISO week 52 of 2024
	// Dec 30, 2024 is a Monday - it's in ISO week 1 of 2025
	// So a newsletter covering Dec 29 - Jan 4 should use week 1 of 2025 (from Monday)

	input := DateInAppTimezone(2025, time.January, 5, 12, 0, 0, 0) // Sunday Jan 5
	result := calculateNewsletterWeek(input)

	// The newsletter covers Dec 29 (Sun) to Jan 4 (Sat)
	if result.Start.Format("2006-01-02") != "2024-12-29" {
		t.Errorf("Start = %s, want 2024-12-29", result.Start.Format("2006-01-02"))
	}

	// Using Monday (Dec 30) for week number should give us week 1 of 2025
	// If we used Sunday (Dec 29), we'd get week 52 of 2024 - which would be wrong
	if result.Year != 2025 {
		t.Errorf("Year = %d, want 2025 (should use Monday's year, not Sunday's)", result.Year)
	}
	if result.WeekNum != 1 {
		t.Errorf("WeekNum = %d, want 1 (should use Monday's week number, not Sunday's)", result.WeekNum)
	}
}
