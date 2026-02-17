package hook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTimeout is applied when no explicit timeout is set.
const DefaultTimeout = 60 * time.Second

// Result holds the output of a hook execution.
type Result struct {
	Output string
	Status string // "success", "error", "skipped"
}

// Run executes the hook script at scriptPath, passing imagePath as the
// first argument. It enforces a timeout and captures stdout/stderr.
// If scriptPath is empty, it returns a "skipped" result.
//
// For .py files, it automatically prepends "python" to the command.
func Run(scriptPath, imagePath string, timeout time.Duration) Result {
	if scriptPath == "" {
		return Result{Status: "skipped"}
	}

	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Determine how to invoke the script.
	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".py":
		cmd = exec.CommandContext(ctx, findPython(), scriptPath, imagePath)
	case ".rb":
		cmd = exec.CommandContext(ctx, "ruby", scriptPath, imagePath)
	case ".js":
		cmd = exec.CommandContext(ctx, "node", scriptPath, imagePath)
	default:
		cmd = exec.CommandContext(ctx, scriptPath, imagePath)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	combined := stdout.String()
	if stderr.Len() > 0 {
		combined += "\n--- stderr ---\n" + stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return Result{
			Output: fmt.Sprintf("hook timed out after %s\n%s", timeout, combined),
			Status: "error",
		}
	}

	if err != nil {
		return Result{
			Output: fmt.Sprintf("hook failed: %v\n%s", err, combined),
			Status: "error",
		}
	}

	return Result{
		Output: combined,
		Status: "success",
	}
}

// findPython returns the path to the Python interpreter, preferring a local
// .venv if one exists next to the vigil binary.
func findPython() string {
	// Check for a local venv (Windows then Linux layout).
	for _, candidate := range []string{
		filepath.Join(".venv", "Scripts", "python.exe"),
		filepath.Join(".venv", "bin", "python3"),
		filepath.Join(".venv", "bin", "python"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "python"
}
