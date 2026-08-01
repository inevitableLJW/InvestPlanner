package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"investplanner/server/internal/config"
	"investplanner/server/internal/database"
	"investplanner/server/internal/httpapi"
	"investplanner/server/internal/repository"
	"investplanner/server/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		fmt.Println("migrations applied")
		return
	}
	app := service.New(repository.New(db), cfg.SessionTTL)
	router := httpapi.New(cfg, app)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	log.Printf("API listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
