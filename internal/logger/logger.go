package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents a logging severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel = LevelInfo
	mu           sync.RWMutex
	stdLogger    = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
)

// SetLevel sets the global log level from a string.
// Accepted values: "DEBUG", "INFO", "WARN", "ERROR" (case-insensitive).
// Defaults to INFO if unrecognized or empty.
func SetLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		currentLevel = LevelDebug
	case "INFO", "":
		currentLevel = LevelInfo
	case "WARN", "WARNING":
		currentLevel = LevelWarn
	case "ERROR":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}
}

// GetLevel returns the current log level as a string.
func GetLevel() string {
	mu.RLock()
	defer mu.RUnlock()

	switch currentLevel {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

func logf(level Level, tag string, component string, format string, args ...any) {
	mu.RLock()
	threshold := currentLevel
	mu.RUnlock()

	if level < threshold {
		return
	}

	msg := fmt.Sprintf(format, args...)
	// Use Output with calldepth=3 so Lshortfile shows the caller's file, not logger.go.
	stdLogger.Output(3, fmt.Sprintf("[%s] [%s] %s", tag, component, msg))
}

// Debug logs a message at DEBUG level. Use for command args, raw output, internal state.
func Debug(component string, format string, args ...any) {
	logf(LevelDebug, "DEBUG", component, format, args...)
}

// Info logs a message at INFO level. Use for startup, successful operations, milestones.
func Info(component string, format string, args ...any) {
	logf(LevelInfo, "INFO", component, format, args...)
}

// Warn logs a message at WARN level. Use for recoverable issues, suspicious state.
func Warn(component string, format string, args ...any) {
	logf(LevelWarn, "WARN", component, format, args...)
}

// Error logs a message at ERROR level. Use for failures that prevent an operation.
func Error(component string, format string, args ...any) {
	logf(LevelError, "ERROR", component, format, args...)
}
