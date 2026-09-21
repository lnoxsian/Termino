package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.lnoxsian.termweb"
	"github.lnoxsian.termweb/internal/config"
	"github.lnoxsian.termweb/internal/server"
	"github.lnoxsian.termweb/internal/terminal"
)

const banner = `
╔═══════════════════════════════════════════════════════════╗
║   _____                   _                               ║
║  |_   _|__ _ __ _ __ ___ (_)_ __   ___                    ║
║    | |/ _ \ '__| '_ ` + "`" + ` _ \| | '_ \ / _ \                   ║
║    | |  __/ |  | | | | | | | | | | (_) |                  ║
║    |_|\___|_|  |_| |_| |_|_|_| |_|\___/                   ║
║                                                           ║
║  Termino                                                  ║
╚═══════════════════════════════════════════════════════════╝
`

var (
	Version   = "1.0.0"
	GitCommit = "none"
	BuildDate = "unknown"
)

func main() {
	var (
		flagHost    = flag.String("host", "", "Server bind host (default: 0.0.0.0)")
		flagPort    = flag.Int("port", 0, "Server bind port (default: 7681)")
		flagShell   = flag.String("shell", "", "Default shell (default: $SHELL or /bin/bash)")
		flagTheme   = flag.String("theme", "", "Default theme: classic, green, amber, blue, monochrome")
		flagConfig  = flag.String("config", "", "Path to config.toml")
		flagVersion = flag.Bool("version", false, "Print version information and exit")
	)
	flag.Parse()

	if *flagVersion {
		fmt.Printf("Termino v%s (commit: %s, date: %s)\n", Version, GitCommit, BuildDate)
		return
	}

	// Load configuration file if specified or if config.toml exists
	configFile := *flagConfig
	if configFile == "" {
		if _, err := os.Stat("config.toml"); err == nil {
			configFile = "config.toml"
		}
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("[termino] error loading config: %v", err)
	}

	// CLI flags override config file
	if *flagHost != "" {
		cfg.Server.Host = *flagHost
	}
	if *flagPort != 0 {
		cfg.Server.Port = *flagPort
	}
	if *flagShell != "" {
		cfg.Terminal.Shell = *flagShell
	}
	if *flagTheme != "" {
		cfg.Desktop.Theme = *flagTheme
	}

	// Initialize terminal session manager
	manager := terminal.NewManager(cfg.Terminal.Shell)

	// Initialize HTTP/WS server
	srv, err := server.NewServer(cfg, manager, termweb.WebFS)
	if err != nil {
		log.Fatalf("[termino] failed to initialize server: %v", err)
	}

	fmt.Print(banner)
	fmt.Printf(" [>] Version:        v%s\n", Version)
	fmt.Printf(" [>] Server URL:     http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf(" [>] Default Shell:  %s\n", cfg.Terminal.Shell)
	fmt.Printf(" [>] Active Theme:   %s\n", cfg.Desktop.Theme)
	fmt.Printf(" [>] Font Size:      %dpx\n", cfg.Terminal.FontSize)
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println(" Press Ctrl+C to terminate Termino")

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("[termino] server fatal error: %v", err)
	case sig := <-sigChan:
		log.Printf("\n[termino] received signal %s, exiting...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[termino] error during shutdown: %v", err)
		}
	}
}
