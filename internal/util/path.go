package util

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

func ResolvePath(path string) string {
	basePath := "data"

	// If the provided path is absolute, return it as-is
	if filepath.IsAbs(path) {
		return path
	}

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		log.Printf("Warning: failed to create base directory: %v", err)
	}

	return filepath.Join(basePath, path)
}

// CopyFile copies a file from src to dst safely
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = srcFile.Close(); err != nil {
			slog.Error("failed to close source file", "error", err)
		}
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err = dstFile.Close(); err != nil {
			slog.Error("failed to close destination file", "error", err)
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
