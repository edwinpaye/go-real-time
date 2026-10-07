package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticServer handles serving frontend projects organized under subpaths with SPA fallback support.
type StaticServer struct {
	baseDir string
	prefix  string
}

// NewStaticServer creates a static file handler for frontend distributions.
func NewStaticServer(prefix, baseDir string) *StaticServer {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &StaticServer{
		baseDir: baseDir,
		prefix:  prefix,
	}
}

func (s *StaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip prefix
	relPath := strings.TrimPrefix(r.URL.Path, s.prefix)
	relPath = strings.TrimPrefix(relPath, "/")

	// Prevent directory traversal
	cleaned := filepath.Clean(relPath)
	if strings.HasPrefix(cleaned, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	targetPath := filepath.Join(s.baseDir, cleaned)
	stat, err := os.Stat(targetPath)

	// If file exists and is not a directory, serve it
	if err == nil && !stat.IsDir() {
		s.serveFileWithHeaders(w, r, targetPath)
		return
	}

	// If it's a directory, look for index.html inside
	if err == nil && stat.IsDir() {
		indexPath := filepath.Join(targetPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			s.serveFileWithHeaders(w, r, indexPath)
			return
		}
	}

	// SPA Fallback: serve root index.html
	rootIndex := filepath.Join(s.baseDir, "index.html")
	if _, err := os.Stat(rootIndex); err == nil {
		s.serveFileWithHeaders(w, r, rootIndex)
		return
	}

	http.NotFound(w, r)
}

func (s *StaticServer) serveFileWithHeaders(w http.ResponseWriter, r *http.Request, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	}

	http.ServeFile(w, r, filePath)
}
