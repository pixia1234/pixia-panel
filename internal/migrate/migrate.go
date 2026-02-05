package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Apply executes all .sql files in the given directory in lexicographic order.
func Apply(db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	sort.Strings(files)

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.TrimSpace(string(content)) == "" {
			continue
		}

		if err := execSQL(db, path, content); err != nil {
			return err
		}
	}

	return nil
}

func execSQL(db *sql.DB, path string, content []byte) error {
	_, err := db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("apply migration %s: %w", filepath.Base(path), err)
	}
	return nil
}
