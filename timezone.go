package main

import (
	"fmt"
	"time"
)

var IST *time.Location

func init() {
	var err error
	IST, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		panic(fmt.Sprintf("failed to load IST timezone: %v", err))
	}
}