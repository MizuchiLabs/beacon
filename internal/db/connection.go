// Package db provides functionality for interacting with the database
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mizuchilabs/sqlite-schema-diff/pkg/diff"
	"github.com/mizuchilabs/sqlite-schema-diff/pkg/parser"
)

//go:embed schemas/*.sql
var schemaFS embed.FS

const DBPath = "data/beacon.db"

// Open connects to the SQLite database and applies the schema. The pool is
// closed when ctx is cancelled.
func Open(ctx context.Context) (*Queries, error) {
	if err := os.MkdirAll(filepath.Dir(DBPath), 0o750); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	dataSource := fmt.Sprintf("file:%s?_txlock=immediate", filepath.ToSlash(DBPath))
	sqliteDB, err := sql.Open("sqlite", dataSource)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := setupSQLite(sqliteDB); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}

	if err := migrate(ctx, sqliteDB); err != nil {
		_ = sqliteDB.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = sqliteDB.Close()
	}()

	return New(sqliteDB), nil
}

// setupSQLite applies performance and safety pragmas.
func setupSQLite(db *sql.DB) error {
	pragmas := `
	PRAGMA busy_timeout = 5000;
	PRAGMA journal_mode = WAL;
	PRAGMA journal_size_limit = 200000000;
	PRAGMA synchronous = NORMAL;
	PRAGMA foreign_keys = ON;
	PRAGMA temp_store = MEMORY;
	PRAGMA mmap_size = 300000000;
	PRAGMA cache_size = -16000;`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		return fmt.Errorf("executing pragmas: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	return nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	parser.SetBaseFS(schemaFS)
	return diff.Apply(ctx, db, "schemas", diff.ApplyOptions{})
}
