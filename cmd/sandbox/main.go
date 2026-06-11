package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
	commit    = "dev"
	buildTime = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\ncommit: %s\nbuildTime: %s\n", version, commit, buildTime)
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.SetFlags(log.Ldate | log.Ltime)

	cfg := loadConfig()
	log.Printf("Starting agent client (host=%s workspace=%s)", cfg.Host, cfg.Workspace)

	if err := os.MkdirAll(cfg.Workspace, 0755); err != nil {
		log.Fatalf("Failed to create workspace: %v", err)
	}

	client := NewAgentClient(cfg)
	client.Run(ctx)

	log.Println("Shutting down")
}
