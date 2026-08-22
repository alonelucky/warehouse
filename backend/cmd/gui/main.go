package main

import (
	"log"
	"os"

	"warehouse/internal/app"
	"warehouse/static"

	"github.com/opentoys/webview"
	"github.com/opentoys/webview/types"
)

func main() {
	operator := os.Getenv("WAREHOUSE_OPERATOR")
	if operator == "" {
		operator = "admin"
	}

	dbPath := "warehouse.db"
	if p := os.Getenv("WAREHOUSE_DB"); p != "" {
		dbPath = p
	}

	dist, err := static.FS()
	if err != nil {
		log.Fatalf("frontend assets: %v", err)
	}

	a, err := app.Run("127.0.0.1:0", dbPath, operator, dist)
	if err != nil {
		log.Fatal(err)
	}
	defer a.WaitShutdown()
	log.Printf("warehouse gui serving %s (db=%s, operator=%s)", a.URL, dbPath, operator)

	w, e := webview.New(webview.Options{
		Debug: true,
	})
	if e != nil {
		log.Fatalln(e)
	}

	defer w.Close()
	w.SetTitle("仓管家")
	w.SetSize(1024, 768, types.SizeNone)
	w.Navigate(a.URL)
	w.Run()
}
