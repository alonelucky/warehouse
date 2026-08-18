package main

import (
	"flag"
	"log"
	"os"

	"warehouse/internal/app"
	"warehouse/static"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "warehouse.db", "sqlite database file path")
	flag.Parse()

	operator := os.Getenv("WAREHOUSE_OPERATOR")
	if operator == "" {
		operator = "admin"
	}

	dist, err := static.FS()
	if err != nil {
		log.Fatalf("frontend assets: %v", err)
	}
	a, err := app.Run(*addr, *dbPath, operator, dist)
	if err != nil {
		log.Fatal(err)
	}
	defer a.WaitShutdown()

	log.Printf("warehouse listening on %s (db=%s, operator=%s)", a.URL, *dbPath, operator)
	select {}
}
