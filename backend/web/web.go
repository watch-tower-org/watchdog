package web

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

func Register(r *gin.Engine, apiPrefix string) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, apiPrefix) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"code":    http.StatusNotFound,
				"error":   "The requested resource was not found",
			})
			return
		}
		serveFromFS(c, sub, c.Request.URL.Path)
	})
}

func serveFromFS(c *gin.Context, sub fs.FS, reqPath string) {
	name := strings.TrimPrefix(reqPath, "/")
	if name == "" {
		name = "index.html"
	}

	f, err := sub.Open(name)
	if err != nil {
		f, err = sub.Open("index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		name = "index.html"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		c.Header("Content-Type", ct)
	}
	http.ServeContent(c.Writer, c.Request, filepath.Base(name), time.Time{}, bytes.NewReader(data))
}
