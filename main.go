package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jeffgeiser/compass-dash/config"
	"github.com/jeffgeiser/compass-dash/server"
)

//go:embed all:frontend
var frontendEmbed embed.FS

func main() {
	var compassPath string
	flag.StringVar(&compassPath, "compass-path", "", "Path to your Compass folder (overrides saved config)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load config (%v); using defaults", err)
	}

	if compassPath != "" {
		cfg.CompassPath = compassPath
		if err := config.Save(cfg); err != nil {
			log.Printf("warning: could not save config: %v", err)
		}
	}

	// Sub the embed to strip the "frontend/" prefix so the server sees files directly.
	frontendFS, err := fs.Sub(frontendEmbed, "frontend")
	if err != nil {
		log.Fatalf("could not prepare frontend FS: %v", err)
	}

	srv := server.New(cfg, frontendFS)

	addr := srv.Addr()
	fmt.Printf("compass-dash listening on http://%s\n", addr)
	if cfg.CompassPath != "" {
		fmt.Printf("compass path: %s\n", cfg.CompassPath)
	} else {
		fmt.Printf("no compass path configured — open the dashboard and set one in Config\n")
	}
	fmt.Printf("open: http://%s\n", addr)

	// Run server in background; wait for signal
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// Ignore "server closed" error on graceful shutdown
			if err.Error() != "http: Server closed" {
				log.Fatalf("server error: %v", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nshutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
