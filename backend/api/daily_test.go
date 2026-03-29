package main

import (
	"testing"
)

func TestValidateDailyTitle(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		currentDate string
		wantErr     string
	}{
		{name: "valid", title: "2026-03-12.md", currentDate: "2026-03-12"},
		{name: "invalid format", title: "2026-03-12", currentDate: "2026-03-12", wantErr: "title must match current date 2026-03-12.md"},
		{name: "wrong date", title: "2026-03-11.md", currentDate: "2026-03-12", wantErr: "title must match current date 2026-03-12.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDailyTitle(tt.title, tt.currentDate)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}
