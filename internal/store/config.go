package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"ardsoundsgrepper/internal/media"
)

const DefaultTargetDir = "~/Music/ARD Sounds"

type Subscription struct {
	URN       string `toml:"urn"`
	Title     string `toml:"title"`
	TargetDir string `toml:"target_dir,omitempty"`
}

type Config struct {
	TargetDir      string         `toml:"target_dir"`
	IncludeTrailer bool           `toml:"include_trailer"`
	Subscriptions  []Subscription `toml:"subscriptions"`
}

func DefaultConfig() Config {
	return Config{TargetDir: DefaultTargetDir}
}

// LoadConfig liefert Defaults, wenn die Datei fehlt; ungültiges TOML ist ein Fehler.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("Config %s ist ungültig: %w", path, err)
	}
	return cfg, nil
}

func EncodeConfig(cfg Config) (string, error) {
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(cfg)
	return buf.String(), err
}

func SaveConfig(path string, cfg Config) error {
	text, err := EncodeConfig(cfg)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, []byte(text))
}

func (c Config) DefaultTarget(showTitle string) string {
	return filepath.Join(ExpandHome(c.TargetDir), media.SanitizeTitle(showTitle))
}

func (c Config) SubscriptionTarget(sub Subscription) string {
	if sub.TargetDir != "" {
		return ExpandHome(sub.TargetDir)
	}
	return c.DefaultTarget(sub.Title)
}

func (c *Config) Subscription(urn string) *Subscription {
	for i := range c.Subscriptions {
		if c.Subscriptions[i].URN == urn {
			return &c.Subscriptions[i]
		}
	}
	return nil
}

func (c *Config) Subscribe(sub Subscription) bool {
	if c.Subscription(sub.URN) != nil {
		return false
	}
	c.Subscriptions = append(c.Subscriptions, sub)
	return true
}

func (c *Config) Unsubscribe(urn string) {
	c.Subscriptions = slices.DeleteFunc(c.Subscriptions, func(s Subscription) bool { return s.URN == urn })
}

// MatchSubscriptions: exakte URN oder Titel-Teilstring (case-insensitive) für sync --abo.
func (c Config) MatchSubscriptions(query string) []Subscription {
	if sub := c.Subscription(query); sub != nil {
		return []Subscription{*sub}
	}
	needle := strings.ToLower(query)
	var matches []Subscription
	for _, sub := range c.Subscriptions {
		if strings.Contains(strings.ToLower(sub.Title), needle) {
			matches = append(matches, sub)
		}
	}
	return matches
}
