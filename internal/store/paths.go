// Package store verwaltet Config, Pfad-History und Lock unter den XDG-Verzeichnissen.
package store

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	AppName        = "ardsoundsgrepper"
	configFile     = "config.toml"
	historyFile    = "history.json"
	lockFile       = "sync.lock"
	configHomeEnv  = "XDG_CONFIG_HOME"
	stateHomeEnv   = "XDG_STATE_HOME"
	configFallback = ".config"
	stateFallback  = ".local/state"
)

func ConfigPath() (string, error)  { return appFile(configHomeEnv, configFallback, configFile) }
func HistoryPath() (string, error) { return appFile(stateHomeEnv, stateFallback, historyFile) }
func LockPath() (string, error)    { return appFile(stateHomeEnv, stateFallback, lockFile) }

func appFile(env, fallback, name string) (string, error) {
	base := os.Getenv(env)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, fallback)
	}
	return filepath.Join(base, AppName, name), nil
}

// ExpandHome löst "~/" auf; gespeichert wird weiter mit "~", damit die Config lesbar bleibt.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
