// Command mycocycle is the runnable entry point for the MycoCycle grow-bag
// transfer gate HTTP backend. It opens the SQLite WAL store, seeds the catalog,
// wires the service facade, serves the JSON API and shuts down gracefully.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mycocycle-growbag-transfer-gate/internal/api"
	"mycocycle-growbag-transfer-gate/internal/config"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
	"mycocycle-growbag-transfer-gate/internal/store"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	cat, err := st.LoadCatalog()
	if err != nil {
		log.Fatalf("load catalog: %v", err)
	}

	svc := service.New(st, cat, domain.NewClock())
	if report, err := svc.Recover(); err == nil {
		log.Printf("recovered state: tasks=%d leases=%d pending_retries=%d decisions=%d",
			report.Tasks, report.ActiveLeases, report.PendingDeviceRetries, report.Decisions)
	}

	srv := api.New(svc, api.WithRequestLogging(cfg.LogRequests))
	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: srv,
	}

	go func() {
		log.Printf("MycoCycle grow-bag transfer gate listening on %s (db=%s)", cfg.Addr, cfg.DBPath)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Printf("shutting down (timeout %s)", cfg.Shutdown)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
