package logging

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in    string
		want  slog.Level
		isErr bool
	}{
		{"", slog.LevelInfo, false},
		{"info", slog.LevelInfo, false},
		{"INFO", slog.LevelInfo, false},
		{"debug", slog.LevelDebug, false},
		{"warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"trace", slog.LevelInfo, true},
	}
	for _, tc := range tests {
		got, err := ParseLevel(tc.in)
		if tc.isErr {
			if err == nil {
				t.Fatalf("ParseLevel(%q): want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseLevel(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
