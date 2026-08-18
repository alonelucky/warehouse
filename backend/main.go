package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"warehouse/internal/controller"
	"warehouse/internal/router"
	"warehouse/internal/service"
	"warehouse/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "warehouse.db", "sqlite database file path")
	flag.Parse()

	operator := os.Getenv("WAREHOUSE_OPERATOR")
	if operator == "" {
		operator = "admin"
	}

	st, err := store.Open(*dbPath, operator)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	ctl := controller.New(svc)
	log.Printf("warehouse listening on %s (db=%s, operator=%s)", *addr, *dbPath, operator)
	log.Fatal(http.ListenAndServe(*addr, router.Routes(ctl)))
}
