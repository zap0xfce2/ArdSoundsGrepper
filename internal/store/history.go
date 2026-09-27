package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
)

const MaxRecentPaths = 10

// History ist Zustand, keine Config: zuletzt genutzte Zielordner und der letzte Ordner je Sendung.
type History struct {
	RecentPaths []string          `json:"recent_paths"`
	ShowPaths   map[string]string `json:"show_paths"`
}

func LoadHistory(path string) (History, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return History{}, nil
	}
	if err != nil {
		return History{}, err
	}
	var h History
	if err := json.Unmarshal(data, &h); err != nil {
		return History{}, fmt.Errorf("History %s ist ungültig: %w", path, err)
	}
	return h, nil
}

func SaveHistory(path string, h History) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data)
}

// Remember setzt path nach vorne (ohne Duplikate, max. 10) und merkt ihn für die Sendung urn.
func (h *History) Remember(urn, path string) {
	others := slices.DeleteFunc(slices.Clone(h.RecentPaths), func(p string) bool { return p == path })
	h.RecentPaths = append([]string{path}, others...)
	if len(h.RecentPaths) > MaxRecentPaths {
		h.RecentPaths = h.RecentPaths[:MaxRecentPaths]
	}
	if urn == "" {
		return
	}
	if h.ShowPaths == nil {
		h.ShowPaths = map[string]string{}
	}
	h.ShowPaths[urn] = path
}

func (h *History) Forget(path string) {
	h.RecentPaths = slices.DeleteFunc(h.RecentPaths, func(p string) bool { return p == path })
}
