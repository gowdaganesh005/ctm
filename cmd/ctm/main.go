package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/analyzer"
	"github.com/ctm/cluster-time-machine/pkg/api"
	"github.com/ctm/cluster-time-machine/pkg/demo"
	"github.com/ctm/cluster-time-machine/pkg/recorder"
	"github.com/ctm/cluster-time-machine/pkg/store"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	dbPath := flag.String("db", "cluster_timemachine.db", "SQLite database file path")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (defaults to KUBECONFIG or ~/.kube/config)")
	staticDir := flag.String("static-dir", "web/dist", "Directory containing built frontend static files")
	enableDemo := flag.Bool("demo", true, "Seed demo scenarios if database is empty")
	flag.Parse()

	log.Printf("=========================================================")
	log.Printf("  Cluster Time Machine - Kubernetes Audit & Time Travel  ")
	log.Printf("=========================================================")
	log.Printf("Database: %s", *dbPath)

	// Initialize SQLite Database
	db, err := store.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Real-time SSE Hub
	hub := api.NewSSEHub()

	// Initialize RCA Analyzer Engine
	rcaEngine := analyzer.NewRCAEngine(db)

	// Broadcaster closure
	broadcaster := func(eventType string, payload any) {
		hub.Broadcast(api.EventType(eventType), payload)
	}

	// Initialize Recorder
	rec := recorder.NewRecorder(db, rcaEngine, *kubeconfig, broadcaster)

	// Initialize Simulator for Demo scenarios
	sim := demo.NewSimulator(rec, db)

	// Pre-seed demo scenarios if enabled
	if *enableDemo {
		if err := sim.SeedDemoData(); err != nil {
			log.Printf("[Warning] Failed to seed demo data: %v", err)
		} else {
			log.Printf("[Simulator] Demo scenarios and baseline telemetry seeded successfully.")
		}
	}

	// Start Kubernetes Watchers in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := rec.Start(ctx); err != nil {
		log.Printf("[Recorder] Watcher start failed: %v", err)
	}

	// Resolve static directory
	resolvedStaticDir := *staticDir
	if _, err := os.Stat(resolvedStaticDir); os.IsNotExist(err) {
		// Try relative to current working dir or binary
		cwd, _ := os.Getwd()
		candidate := filepath.Join(cwd, "web", "dist")
		if _, err := os.Stat(candidate); err == nil {
			resolvedStaticDir = candidate
		} else {
			resolvedStaticDir = "" // API only mode until frontend is built
		}
	}

	// Initialize API Server
	apiHandler := api.NewAPI(db, rcaEngine, rec, sim, hub)
	server := api.NewServer(apiHandler, hub, *port, resolvedStaticDir)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	fmt.Printf("\n>>> Cluster Time Machine is active at http://localhost:%d\n\n", *port)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down Cluster Time Machine...")
	rec.Stop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Stop(shutdownCtx)
	log.Println("Cluster Time Machine stopped gracefully.")
}
