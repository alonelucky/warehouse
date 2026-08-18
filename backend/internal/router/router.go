// Package router assembles all routes and middleware chains.
package router

import (
	"net/http"

	"warehouse/internal/controller"
	"warehouse/pkg/web"
)

func Routes(c *controller.Controller) http.Handler {
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

	return mux
}
