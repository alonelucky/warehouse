// Package app wires the store, service, controller, router and HTTP server
// together for both the server and webview GUI entry points.
package app

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"warehouse/internal/controller"
	"warehouse/internal/router"
	"warehouse/internal/service"
	"warehouse/internal/store"
)

type App struct {
	Store  *store.Store
	Server *http.Server
	URL    string
}

// Run opens the store (creating it when missing), starts the HTTP server and
// returns an App whose URL holds the bound address. addr "127.0.0.1:0" picks
// an available port, which the GUI mode relies on.
func Run(addr, dbPath, operator string, dist fs.FS) (*App, error) {
	st, err := store.Open(dbPath, operator)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	svc := service.New(st)
	ctl := controller.New(svc)
	server := &http.Server{Handler: router.Routes(ctl, dist)}

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("http server: %v", err)
		}
	}()

	return &App{
		Store:  st,
		Server: server,
		URL:    "http://" + ln.Addr().String(),
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.Store.Close()
	return a.Server.Shutdown(ctx)
}

func (a *App) WaitShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
