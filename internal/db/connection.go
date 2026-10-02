// Package db provides functionality for interacting with the database
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mizuchilabs/sqlite-schema-diff/pkg/diff"
)

//go:embed schemas/*.sql
var schemaFS embed.FS

// pragmas live in the DSN so every connection gets them, not just the first.
const pragmas = "_txlock=immediate" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=journal_size_limit(200000000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=mmap_size(300000000)" +
	"&_pragma=cache_size(-16000)"

// Open connects to beacon.db inside dataDir and applies the schema. The pool
// is closed when ctx is cancelled.
func Open(ctx context.Context, dataDir string) (*Queries, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	path := filepath.Join(dataDir, "beacon.db")
	backup := ""
	if _, err := os.Stat(path); err == nil {
		backup = path + ".backup"
	}

	sqliteDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?"+pragmas)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	sqliteDB.SetMaxOpenConns(1)

	if err := migrate(ctx, sqliteDB, backup); err != nil {
		_ = sqliteDB.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	q := New(sqliteDB)
	if err := q.BackfillRollups(ctx); err != nil {
		_ = sqliteDB.Close()
		return nil, fmt.Errorf("backfilling rollups: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = sqliteDB.Close()
	}()

	return q, nil
}

// migrate applies the embedded schema. A backup is only written when a path is
// given and the schema actually changes.
func migrate(ctx context.Context, db *sql.DB, backupPath string) error {
	fsys, err := fs.Sub(schemaFS, "schemas")
	if err != nil {
		return err
	}
	_, err = diff.Apply(ctx, db, fsys, diff.ApplyOptions{BackupPath: backupPath})
	return err
}
