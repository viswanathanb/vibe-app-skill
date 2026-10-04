package server

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"example.com/app/internal/httpx"
)

// registerSPA serves the built React app from dir and falls back to index.html for
// client-side routes. Unknown /api/* paths always return a JSON 404.
func registerSPA(r *gin.Engine, dir string) {
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if dir == "" || strings.HasPrefix(p, "/api/") || p == "/api" ||
			(c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) {
			httpx.Fail(c, httpx.NotFound("route not found"))
			return
		}

		root := http.Dir(dir) // http.Dir rejects path traversal
		clean := path.Clean("/" + p)
		if f, err := root.Open(clean); err == nil {
			info, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !info.IsDir() {
				if strings.HasPrefix(clean, "/assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				http.FileServer(root).ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.Header("Cache-Control", "no-cache")
		c.File(filepath.Join(dir, "index.html"))
	})
}
