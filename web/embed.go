// Package web embeds the SvelteKit build output and serves it as a single page app.
package web

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/vearutop/statigz"
	"github.com/vearutop/statigz/brotli"
)

//go:generate pnpm install
//go:generate pnpm build
//go:embed all:build
var StaticFS embed.FS

// Handler serves the embedded SPA.
func Handler() http.Handler {
	if dev := os.Getenv("BEACON_DEV_UI"); dev != "" {
		return devProxy(dev)
	}
	pinMimeTypes()
	files := statigz.FileServer(StaticFS, brotli.AddEncoding, statigz.FSPrefix("build"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p != "/" && !exists(p) {
			if exists(p + ".html") {
				p += ".html"
			} else {
				p = "/"
			}
		}
		r.URL.Path = p
		w.Header().Set("Cache-Control", cachePolicy(p))
		files.ServeHTTP(w, r)
	})
}

func devProxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		panic("BEACON_DEV_UI is not a url: " + target)
	}
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(u)
			r.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, "No Vite at "+target+", run pnpm dev in web. "+err.Error(), http.StatusBadGateway)
		},
	}
}

// pinMimeTypes keeps content types host independent.
func pinMimeTypes() {
	_ = mime.AddExtensionType(".txt", "text/plain; charset=utf-8")
	_ = mime.AddExtensionType(".ico", "image/x-icon")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
	_ = mime.AddExtensionType(".woff", "font/woff")
	_ = mime.AddExtensionType(".woff2", "font/woff2")
}

func exists(p string) bool {
	st, err := fs.Stat(StaticFS, path.Join("build", p))
	return err == nil && !st.IsDir()
}

func cachePolicy(p string) string {
	switch {
	case strings.HasPrefix(p, "/_app/immutable/"):
		return "public, max-age=31536000, immutable"
	case p == "/" || strings.HasSuffix(p, ".html") || p == "/service-worker.js" || p == "/_app/version.json":
		return "no-cache"
	default:
		return "public, max-age=3600"
	}
}
