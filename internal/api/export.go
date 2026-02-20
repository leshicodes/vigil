package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateRandomName(ext string) string {
	adjectives := []string{"swift", "silent", "sneaky", "sleepy", "clever", "wild", "brave", "calm", "fierce", "happy", "lucky", "proud", "jumpy", "fuzzy"}
	animals := []string{"fox", "owl", "bear", "wolf", "hawk", "deer", "lynx", "hare", "puma", "frog", "moth", "crow", "cat", "dog"}

	adj := adjectives[rng.Intn(len(adjectives))]
	ani := animals[rng.Intn(len(animals))]
	return fmt.Sprintf("%s-%s.%s", adj, ani, ext)
}

func formatSRTTime(ms int) string {
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

type ExportRequest struct {
	IDs             []int64 `json:"ids"`
	Format          string  `json:"format"`           // "mp4" or "gif"
	IncludeTimecode bool    `json:"include_timecode"` // burn timestamp into video
}

func (s *Server) handleExportCaptures(w http.ResponseWriter, r *http.Request) {
	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if len(req.IDs) < 2 {
		http.Error(w, `{"error":"at least 2 captures required for export"}`, http.StatusBadRequest)
		return
	}

	captures, err := s.DB.GetCapturesByIDs(req.IDs)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch captures"}`, http.StatusInternalServerError)
		return
	}

	if len(captures) < 2 {
		http.Error(w, `{"error":"could not find enough matching captures"}`, http.StatusBadRequest)
		return
	}

	// Create a temporary directory for the ffmpeg processing
	tmpDir, err := os.MkdirTemp("", "vigil-export-*")
	if err != nil {
		http.Error(w, `{"error":"failed to create temp directory"}`, http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Copy all captures into the temp directory with sequential names for ffmpeg
	for i, cap := range captures {
		// Extract date and filename from cap.Filepath, ignoring the rest to handle moved data directories or cross-OS DB transfers
		parts := strings.Split(filepath.ToSlash(cap.Filepath), "/")
		if len(parts) < 2 {
			continue
		}
		filename := parts[len(parts)-1]
		date := parts[len(parts)-2]

		actualPath := filepath.Join(s.DataDir, "captures", date, filename)

		srcFile, err := os.Open(actualPath)
		if err != nil {
			fmt.Printf("[api] warning: failed to open capture file %s for export: %s\n", actualPath, err)
			continue
		}
		defer srcFile.Close()

		destPath := filepath.Join(tmpDir, fmt.Sprintf("img_%04d.jpg", i+1))
		destFile, err := os.Create(destPath)
		if err != nil {
			fmt.Printf("[api] warning: failed to create temp file %s: %s\n", destPath, err)
			continue
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			fmt.Printf("[api] warning: failed to copy to temp file %s: %s\n", destPath, err)
			continue
		}
	}

	format := req.Format
	if format != "gif" {
		format = "mp4" // default
	}

	if req.IncludeTimecode {
		var srtBuilder strings.Builder
		for i, cap := range captures {
			startTimeMs := i * 100
			endTimeMs := (i + 1) * 100

			startStr := formatSRTTime(startTimeMs)
			endStr := formatSRTTime(endTimeMs)
			displayTime := cap.Timestamp.Local().Format("2006-01-02 03:04 PM")

			srtBuilder.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, startStr, endStr, displayTime))
		}

		err = os.WriteFile(filepath.Join(tmpDir, "subs.srt"), []byte(srtBuilder.String()), 0644)
		if err != nil {
			fmt.Printf("[api] warning: failed to write subtitles file: %s\n", err)
		}
	}

	// Run ffmpeg to stitch the images
	outputPath := "output." + format
	inputPattern := "img_%04d.jpg"

	var cmdArgs []string
	var vfGraph string

	if format == "gif" {
		if req.IncludeTimecode {
			vfGraph = "subtitles=subs.srt,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"
		} else {
			vfGraph = "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"
		}
		cmdArgs = []string{
			"-y",
			"-framerate", "10",
			"-i", inputPattern,
			"-vf", vfGraph,
			"-loop", "0",
			outputPath,
		}
	} else {
		if req.IncludeTimecode {
			vfGraph = "subtitles=subs.srt"
		}
		cmdArgs = []string{
			"-y",
			"-framerate", "10",
			"-i", inputPattern,
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
		}
		if vfGraph != "" {
			cmdArgs = append(cmdArgs, "-vf", vfGraph)
		}
		cmdArgs = append(cmdArgs, outputPath)
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Dir = tmpDir // Run inside temp dir to simplify paths resolving

	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("[api] ffmpeg export error: %s\nOutput:\n%s\n", err, output)
		http.Error(w, `{"error":"failed to generate video"}`, http.StatusInternalServerError)
		return
	}

	// Open the generated file
	outFile, err := os.Open(filepath.Join(tmpDir, outputPath))
	if err != nil {
		http.Error(w, `{"error":"failed to read generated video"}`, http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	fileStat, err := outFile.Stat()
	if err != nil {
		http.Error(w, `{"error":"failed to stat generated video"}`, http.StatusInternalServerError)
		return
	}

	outFileName := generateRandomName(format)

	// Serve the exported file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outFileName))
	if format == "gif" {
		w.Header().Set("Content-Type", "image/gif")
	} else {
		w.Header().Set("Content-Type", "video/mp4")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(fileStat.Size(), 10))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, outFile); err != nil {
		fmt.Printf("[api] warning: failed to send export video to client: %s\n", err)
	}
}
