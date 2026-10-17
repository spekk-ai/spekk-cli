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
	// git runs the credential helper as "agent-client git-credential get".
	if len(os.Args) > 1 && os.Args[1] == "git-credential" {
		os.Exit(runGitCredential(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
	}

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
	switch status, check := checkClaudeCode(); check {
	case claudeCodeCurrent:
		log.Print(status)
	case claudeCodeUnreadable:
		log.Printf("WARNING: %s Turns run with no version check.", status)
	case claudeCodeOld:
		log.Printf("WARNING: %s Every turn is refused until then.", status)
	}

	if claudeAuthState() == claudeAuthSignInRequired {
		log.Print("WARNING: Claude Code is not signed in. Every turn is refused until a person signs in on this sandbox.")
	}

	if err := os.MkdirAll(cfg.Workspace, 0755); err != nil {
		log.Fatalf("Failed to create workspace: %v", err)
	}

	client := NewAgentClient(cfg)
	client.Run(ctx)

	log.Println("Shutting down")
}
