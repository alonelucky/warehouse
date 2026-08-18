// Package router assembles all routes and middleware chains.
package router

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"warehouse/internal/controller"
	"warehouse/pkg/web"
)

func Routes(c *controller.Controller, dist fs.FS) http.Handler {
	mux := http.NewServeMux()
	pub := []web.Middleware{web.Recoverer, web.Logger}

	mux.HandleFunc("GET /api/meta", web.Chain(pub...)(c.Meta))
	mux.HandleFunc("GET /api/stats", web.Chain(pub...)(c.Stats))

	mux.HandleFunc("GET /api/products", web.Chain(pub...)(c.ListProducts))
	mux.HandleFunc("POST /api/products", web.Chain(pub...)(c.CreateProduct))
	mux.HandleFunc("PUT /api/products/{id}", web.Chain(pub...)(c.UpdateProduct))

	mux.HandleFunc("GET /api/movements", web.Chain(pub...)(c.ListMovements))
	mux.HandleFunc("POST /api/movements", web.Chain(pub...)(c.AddMovement))
	mux.HandleFunc("POST /api/movements/batch", web.Chain(pub...)(c.AddMovementBatch))

	mux.HandleFunc("GET /api/locations", web.Chain(pub...)(c.ListLocations))
	mux.HandleFunc("GET /api/batches", web.Chain(pub...)(c.ListBatches))

	mux.HandleFunc("GET /api/audit", web.Chain(pub...)(c.ListAudit))

	if dist != nil {
		mux.Handle("/", spaHandler(dist))
	}
	return mux
}

// spaHandler serves the frontend build: actual files when the path has an
// extension, otherwise index.html so client-side routing keeps working.
func spaHandler(dist fs.FS) http.HandlerFunc {
	index := func() []byte {
		f, err := dist.Open("index.html")
		if err != nil {
			return nil
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		return b
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if strings.Contains(p, ".") {
			f, err := dist.Open(p)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, _ := f.Stat()
			if ct := mime.TypeByExtension(filepath.Ext(p)); ct != "" {
				w.Header().Set("Content-Type", ct)
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
			}
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			io.Copy(w, f)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(index)))
		w.Write(index)
	}
}
