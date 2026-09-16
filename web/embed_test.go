package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireBuiltAssets(t *testing.T) {
	t.Helper()
	if _, err := fs.Stat(StaticFS, "build/_app"); err != nil {
		t.Skip("frontend not built, run pnpm build")
	}
}

func textChunk(t *testing.T) string {
	t.Helper()
	var found string
	err := fs.WalkDir(StaticFS, "build/_app/immutable", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found == "" && !d.IsDir() && (strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".css")) {
			found = p
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, found, "no js or css chunk in build output")
	return "/" + strings.TrimPrefix(found, "build/")
}

func TestCachePolicy(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/_app/immutable/chunks/a.js", "public, max-age=31536000, immutable"},
		{"/", "no-cache"},
		{"/index.html", "no-cache"},
		{"/events.html", "no-cache"},
		{"/service-worker.js", "no-cache"},
		{"/_app/version.json", "no-cache"},
		{"/favicon.ico", "public, max-age=3600"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, cachePolicy(tt.path))
		})
	}
}

func TestCachePolicyMatrix(t *testing.T) {
	requireBuiltAssets(t)
	h := Handler()
	chunk := textChunk(t)

	tests := []struct {
		path   string
		want   string
		status int
	}{
		{"/", "no-cache", http.StatusOK},
		{"/monitors/whatever", "no-cache", http.StatusOK},
		{"/robots.txt", "public, max-age=3600", http.StatusOK},
		{"/service-worker.js", "no-cache", http.StatusOK},
		{"/_app/version.json", "no-cache", http.StatusOK},
		{chunk, "public, max-age=31536000, immutable", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.want, w.Header().Get("Cache-Control"))
		})
	}
}

func TestEncodingNegotiation(t *testing.T) {
	requireBuiltAssets(t)
	h := Handler()
	chunk := textChunk(t)

	tests := []struct {
		accept string
		want   string
	}{
		{"br", "br"},
		{"gzip", "gzip"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, chunk, nil)
			if tt.accept != "" {
				req.Header.Set("Accept-Encoding", tt.accept)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Header().Get("Content-Encoding"))
		})
	}
}

func TestETagRevalidation(t *testing.T) {
	requireBuiltAssets(t)
	h := Handler()

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, w.Code)
	etag := w.Header().Get("Etag")
	require.NotEmpty(t, etag)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusNotModified, w2.Code)
}

func TestMimePinning(t *testing.T) {
	requireBuiltAssets(t)
	h := Handler()

	tests := []struct {
		file string
		want string
	}{
		{"/robots.txt", "text/plain; charset=utf-8"},
		{"/favicon.ico", "image/x-icon"},
		{"/site.webmanifest", "application/manifest+json"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.file, nil))
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.want, w.Header().Get("Content-Type"))
		})
	}
}

func TestPrerenderedPageResolution(t *testing.T) {
	requireBuiltAssets(t)
	h := Handler()

	var page string
	err := fs.WalkDir(StaticFS, "build", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		base := path.Base(p)
		if page == "" && !d.IsDir() && strings.HasSuffix(base, ".html") && base != "index.html" {
			page = p
		}
		return nil
	})
	require.NoError(t, err)
	if page == "" {
		t.Skip("no prerendered pages in build output")
	}

	url := "/" + strings.TrimSuffix(strings.TrimPrefix(page, "build/"), ".html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}
