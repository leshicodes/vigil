package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	// Configuration
	tzName := "America/Chicago"
	dataDir := "data"
	dbPath := filepath.Join(dataDir, "vigil.db")

	// Load timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Fatalf("failed to load location %q: %v", tzName, err)
	}
	fmt.Printf("Using timezone: %s\n", loc)

	// Open DB
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// List all captures
	rows, err := db.Query("SELECT id, timestamp FROM capture_log")
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	type update struct {
		id    int64
		newTS string
	}
	var updates []update

	for rows.Next() {
		var id int64
		var tsStr string
		if err := rows.Scan(&id, &tsStr); err != nil {
			log.Fatal(err)
		}

		// Parse existing timestamp
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			log.Printf("check id=%d: failed to parse time %q: %v", id, tsStr, err)
			continue
		}

		// Check if it's already local (has offset -06:00 or similar)
		// If it ends in 'Z', it's UTC.
		// Or if .Location() is UTC.

		// We want to force it to use the configured location.
		// Note: ts.In(loc) changes the location but keeps the instant.
		localTS := ts.In(loc)
		newTSStr := localTS.Format(time.RFC3339)

		if tsStr != newTSStr {
			updates = append(updates, update{
				id:    id,
				newTS: newTSStr,
			})
		}
	}
	rows.Close()

	if len(updates) == 0 {
		fmt.Println("No timestamps need migration.")
		return
	}

	fmt.Printf("Found %d timestamps to migrate.\n", len(updates))

	// Execute updates
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range updates {
		_, err := tx.Exec("UPDATE capture_log SET timestamp = ? WHERE id = ?", u.newTS, u.id)
		if err != nil {
			tx.Rollback()
			log.Fatalf("FAIL id=%d: db update: %v", u.id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Successfully migrated %d timestamps.\n", len(updates))
}
