// Package db provides functionality for interacting with the database
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mizuchilabs/sqlite-schema-diff/pkg/diff"
	"github.com/mizuchilabs/sqlite-schema-diff/pkg/parser"
	_ "modernc.org/sqlite"
)

//go:embed schemas/*.sql
var schemaFS embed.FS

var DBPath = "data/beacon.db"

type Connection struct {
	DB *sql.DB
	Q  *Queries
}

func NewConnection(ctx context.Context) *Connection {
	if err := os.MkdirAll("data", 0o750); err != nil {
		slog.Error("Failed to create data directory", "error", err)
	}

	dataSource := fmt.Sprintf("file:%s?_txlock=immediate", filepath.ToSlash(DBPath))
	sqliteDB, err := sql.Open("sqlite", dataSource)
	if err != nil {
		slog.Error("Failed to open database", "err", err)
		os.Exit(1)
	}

	if err := setupSQLite(sqliteDB); err != nil {
		slog.Error("Failed to configure database", "err", err)
		os.Exit(1)
	}
	migrate(ctx, sqliteDB)

	// Wait for shutdown signal
	go func() {
		<-ctx.Done()
		if err := sqliteDB.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	return &Connection{
		DB: sqliteDB,
		Q:  New(sqliteDB),
	}
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

func migrate(ctx context.Context, db *sql.DB) {
	parser.SetBaseFS(schemaFS)
	if err := diff.Apply(ctx, db, "schemas", diff.ApplyOptions{}); err != nil {
		slog.Error("failed to apply schema changes", "error", err)
		return
	}
}
