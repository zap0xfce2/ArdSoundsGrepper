package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"ardsoundsgrepper/internal/store"
)

func TestConfigPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	if got, _ := store.ConfigPath(); got != "/xdg/config/ardsoundsgrepper/config.toml" {
		t.Errorf("ConfigPath = %s", got)
	}
}

func TestHistoryPathUsesXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	if got, _ := store.HistoryPath(); got != "/xdg/state/ardsoundsgrepper/history.json" {
		t.Errorf("HistoryPath = %s", got)
	}
}

func TestLockPathFallsBackToLocalState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	if got, _ := store.LockPath(); got != filepath.Join(home, ".local/state/ardsoundsgrepper/sync.lock") {
		t.Errorf("LockPath = %s", got)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := store.ExpandHome("~/Music"); got != filepath.Join(home, "Music") {
		t.Errorf("ExpandHome = %s", got)
	}
}

func TestExpandHomeKeepsAbsolutePath(t *testing.T) {
	if got := store.ExpandHome("/srv/~x"); got != "/srv/~x" {
		t.Errorf("ExpandHome = %s", got)
	}
}
