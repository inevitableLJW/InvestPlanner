package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"investplanner/server/internal/config"
	"investplanner/server/internal/database"
	"investplanner/server/internal/httpapi"
	"investplanner/server/internal/marketdata"
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
	store := repository.New(db)
	app := service.New(store, cfg.SessionTTL)
	app.QDII = &service.QDIIService{Store: store, Now: time.Now, FreshFor: 24 * time.Hour}
	app.FX = &service.FXService{Store: store, Now: time.Now, Enabled: cfg.FXAssistantEnabled}
	snapshots := &marketdata.SnapshotService{Store: store, Now: time.Now, FreshFor: map[string]time.Duration{"purchase_limit": 12 * time.Hour, "etf_quote": 15 * time.Minute, "fx_rate": 36 * time.Hour}}
	collector := &marketdata.Collector{Store: store, Snapshots: snapshots, AKTools: marketdata.NewAKToolsClient(cfg.AKToolsBaseURL, 45*time.Second), PurchaseDetails: marketdata.NewEastMoneyPurchaseClient(), Interval: 6 * time.Hour, Concurrency: 2, Now: time.Now}
	if cfg.TushareToken != "" {
		collector.Tushare = marketdata.NewTushareClient(cfg.TushareToken, 5*time.Second)
	}
	if cfg.ChinaMoneyURL != "" {
		collector.ChinaMoney = marketdata.NewChinaMoneyClient(cfg.ChinaMoneyURL, 5*time.Second)
	}
	go collector.Run(context.Background())
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
