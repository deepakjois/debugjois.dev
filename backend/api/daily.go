package main

import (
	"encoding/base64"
	"fmt"

	"github.com/matthalp/go-meridian/v2/timezones/cet"
)

type dailyResponse struct {
	Title    string `json:"title"`
	Contents string `json:"contents"`
}

func todayStringInCET() string {
	return cet.Now().Format("2006-01-02")
}

func currentTimestampInCET() string {
	return cet.Now().Format("2006-01-02 15:04:05")
}

func validateDailyTitle(title, currentDate string) error {
	if title != fmt.Sprintf("%s.md", currentDate) {
		return fmt.Errorf("title must match current date %s.md", currentDate)
	}

	return nil
}

func encodeDailyContents(contents string) string {
	return base64.StdEncoding.EncodeToString([]byte(contents))
}
