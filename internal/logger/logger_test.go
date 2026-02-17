package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"DEBUG", "DEBUG"},
		{"debug", "DEBUG"},
		{"INFO", "INFO"},
		{"", "INFO"},
		{"WARN", "WARN"},
		{"WARNING", "WARN"},
		{"ERROR", "ERROR"},
		{"nonsense", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevel(tt.input)
			if got := GetLevel(); got != tt.want {
				t.Errorf("SetLevel(%q): GetLevel() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	stdLogger = log.New(&buf, "", 0) // No flags for clean test output.

	SetLevel("WARN")
	Debug("test", "should not appear")
	Info("test", "should not appear")
	Warn("test", "should appear")
	Error("test", "also appears")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Errorf("DEBUG/INFO messages should be filtered at WARN level, got: %s", output)
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("WARN message should appear, got: %s", output)
	}
	if !strings.Contains(output, "also appears") {
		t.Errorf("ERROR message should appear, got: %s", output)
	}

	// Verify format includes tags.
	if !strings.Contains(output, "[WARN] [test]") {
		t.Errorf("expected [WARN] [test] tag, got: %s", output)
	}

	// Reset.
	SetLevel("INFO")
}
