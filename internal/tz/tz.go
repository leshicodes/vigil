package tz

import (
	"os"
	"sync"
	"time"

	"github.com/leshicodes/vigil/internal/logger"
)

var (
	loc     *time.Location
	locOnce sync.Once
)

// Location returns the *time.Location derived from the TZ environment
// variable. If TZ is empty the location defaults to UTC (matching the
// previous implicit behaviour inside Docker containers). The value is
// loaded once and cached for the lifetime of the process.
func Location() *time.Location {
	locOnce.Do(func() {
		name := os.Getenv("TZ")
		if name == "" {
			loc = time.UTC
			return
		}

		l, err := time.LoadLocation(name)
		if err != nil {
			logger.Warn("tz", "invalid TZ %q (%v) - falling back to UTC", name, err)
			loc = time.UTC
			return
		}
		loc = l
	})
	return loc
}

// Name returns a human-readable description of the resolved timezone,
// suitable for startup log messages.
func Name() string {
	l := Location()
	if l == time.UTC {
		if os.Getenv("TZ") == "" {
			return "UTC (default, set TZ to override)"
		}
		return "UTC"
	}
	return l.String() + " (from TZ env)"
}
