// Command ardsoundsgrepper lädt Folgen von ARD-Sounds-Sendungen per TUI oder headless per sync.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"ardsoundsgrepper/internal/abosync"
	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
	"ardsoundsgrepper/internal/tui"
)

// Wird beim Build per -ldflags "-X main.version=vYYMMDDhhmm" gesetzt.
var version = "vYYMMDDhhmm"

const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
)

const usageText = `Nutzung:
  ardsoundsgrepper                   startet die TUI
  ardsoundsgrepper sync [--abo X]    lädt fehlende Folgen aller Abos (oder nur Abo X)
  ardsoundsgrepper --version         zeigt die Version`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runTUI(stderr)
	}
	switch args[0] {
	case "--version":
		fmt.Fprintln(stdout, version)
		return exitOK
	case "sync":
		return runSync(args[1:], stdout, stderr)
	}
	fmt.Fprintln(stderr, usageText)
	return exitUsage
}

func runSync(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	abo := flags.String("abo", "", "nur dieses Abo (URN oder Titel-Teilstring)")
	if err := flags.Parse(args); err != nil {
		return abosync.ExitUsage
	}
	if flags.NArg() > 0 { // z. B. "sync mimi" ohne --abo würde sonst alle Abos laden
		fmt.Fprintln(stderr, usageText)
		return abosync.ExitUsage
	}
	paths, err := resolvePaths()
	if err != nil {
		fmt.Fprintf(stderr, "Fehler: %v\n", err)
		return abosync.ExitUsage
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return abosync.Run(ctx, abosync.Options{
		AboFilter: *abo, Client: api.NewClient(api.DefaultEndpoint()), Downloader: media.NewDownloader(),
		ConfigPath: paths.config, LockPath: paths.lock, Version: version, Out: stdout, Err: stderr,
	})
}

type appPaths struct{ config, history, lock string }

func resolvePaths() (appPaths, error) {
	config, err := store.ConfigPath()
	if err != nil {
		return appPaths{}, err
	}
	history, err := store.HistoryPath()
	if err != nil {
		return appPaths{}, err
	}
	lock, err := store.LockPath()
	return appPaths{config: config, history: history, lock: lock}, err
}

func runTUI(stderr io.Writer) int {
	paths, err := resolvePaths()
	if err != nil {
		fmt.Fprintf(stderr, "Fehler: %v\n", err)
		return exitUsage
	}
	cfg, cfgErr := store.LoadConfig(paths.config)
	if cfgErr != nil {
		cfg = store.DefaultConfig()
	}
	// Die History ist kein Nutzerinhalt: ist sie kaputt, startet die TUI mit leerer History.
	history, _ := store.LoadHistory(paths.history)
	err = tui.Run(tui.Deps{
		Client: api.NewClient(api.DefaultEndpoint()), Downloader: media.NewDownloader(),
		Config: cfg, ConfigErr: cfgErr, ConfigPath: paths.config,
		History: history, HistoryPath: paths.history, LockPath: paths.lock, Version: version,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Fehler: %v\n", err)
		return exitFailed
	}
	return exitOK
}
