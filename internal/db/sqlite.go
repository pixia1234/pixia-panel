package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// Open opens a SQLite database and applies required PRAGMAs.
// The caller must register a SQLite driver (e.g. modernc.org/sqlite or mattn/go-sqlite3).
func Open(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("db path is empty")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return db, nil
}
