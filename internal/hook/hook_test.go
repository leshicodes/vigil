package hook

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func writeScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)

	if runtime.GOOS == "windows" {
		// On Windows, write a .bat file instead.
		p += ".bat"
		if err := os.WriteFile(p, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}

	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+content), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunSuccess(t *testing.T) {
	dir := t.TempDir()

	var script string
	if runtime.GOOS == "windows" {
		script = writeScript(t, dir, "ok", "@echo off\necho hello from hook\n")
	} else {
		script = writeScript(t, dir, "ok", "echo hello from hook\n")
	}

	r := Run(script, "/tmp/test.jpg", 5*time.Second)
	if r.Status != "success" {
		t.Fatalf("expected success, got %s: %s", r.Status, r.Output)
	}
	if r.Output == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRunSkipped(t *testing.T) {
	r := Run("", "/tmp/test.jpg", 5*time.Second)
	if r.Status != "skipped" {
		t.Fatalf("expected skipped, got %s", r.Status)
	}
}

func TestRunError(t *testing.T) {
	dir := t.TempDir()

	var script string
	if runtime.GOOS == "windows" {
		script = writeScript(t, dir, "fail", "@echo off\nexit /b 1\n")
	} else {
		script = writeScript(t, dir, "fail", "exit 1\n")
	}

	r := Run(script, "/tmp/test.jpg", 5*time.Second)
	if r.Status != "error" {
		t.Fatalf("expected error, got %s", r.Status)
	}
}

func TestRunMissingScript(t *testing.T) {
	r := Run("/nonexistent/script.sh", "/tmp/test.jpg", 5*time.Second)
	if r.Status != "error" {
		t.Fatalf("expected error, got %s", r.Status)
	}
}

func TestRunTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("timeout test not reliable on Windows")
	}

	dir := t.TempDir()
	script := writeScript(t, dir, "slow", "sleep 10\n")

	r := Run(script, "/tmp/test.jpg", 500*time.Millisecond)
	if r.Status != "error" {
		t.Fatalf("expected error (timeout), got %s", r.Status)
	}
}
