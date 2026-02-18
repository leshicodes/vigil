package tz

import (
	"os"
	"sync"
	"testing"
	"time"
)

// reset clears the cached location so each test starts fresh.
func reset() {
	locOnce = sync.Once{}
	loc = nil
}

func TestLocation_DefaultUTC(t *testing.T) {
	reset()
	os.Unsetenv("TZ")

	l := Location()
	if l != time.UTC {
		t.Fatalf("expected UTC, got %v", l)
	}
}

func TestLocation_ValidTZ(t *testing.T) {
	reset()
	os.Setenv("TZ", "America/Chicago")
	defer os.Unsetenv("TZ")

	l := Location()
	if l.String() != "America/Chicago" {
		t.Fatalf("expected America/Chicago, got %v", l)
	}
}

func TestLocation_InvalidTZ(t *testing.T) {
	reset()
	os.Setenv("TZ", "Fake/Zone")
	defer os.Unsetenv("TZ")

	l := Location()
	if l != time.UTC {
		t.Fatalf("expected UTC fallback for invalid TZ, got %v", l)
	}
}

func TestName_Default(t *testing.T) {
	reset()
	os.Unsetenv("TZ")

	n := Name()
	if n != "UTC (default, set TZ to override)" {
		t.Fatalf("unexpected name: %s", n)
	}
}

func TestName_WithTZ(t *testing.T) {
	reset()
	os.Setenv("TZ", "America/Chicago")
	defer os.Unsetenv("TZ")

	n := Name()
	if n != "America/Chicago (from TZ env)" {
		t.Fatalf("unexpected name: %s", n)
	}
}
