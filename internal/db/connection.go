// Package db provides functionality for interacting with the database
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Connection struct {
	mu sync.RWMutex
	db *sql.DB
	Q  *Queries
}

// NewConnection opens a SQLite connection.
func NewConnection(ctx context.Context, path string) *Connection {
	// Check path
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		log.Fatalf("failed to create db dir: %v", err)
	}

	dataSource := fmt.Sprintf("file:%s?_txlock=immediate", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	if err := setupSQLite(db); err != nil {
		log.Fatalf("failed to configure db: %v", err)
	}

	conn := &Connection{
		db: db,
		Q:  New(db),
	}
	conn.migrate()

	// Wait for shutdown signal
	go func() {
		<-ctx.Done()
		if err := db.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	return conn
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

func (c *Connection) Get() *sql.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

func (c *Connection) migrate() {
	goose.SetBaseFS(migrationFS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
	}
	if err := goose.Up(c.db, "migrations"); err != nil {
		log.Fatal(err)
	}
}
