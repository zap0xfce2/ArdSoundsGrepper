package store_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ardsoundsgrepper/internal/store"
)

func sampleConfig() store.Config {
	return store.Config{TargetDir: "/music", Subscriptions: []store.Subscription{
		{URN: "urn:ard:show:a", Title: "Mimi Sandmädchen", TargetDir: "/music/Mimi"},
		{URN: "urn:ard:show:b", Title: "Mimi und Mio"},
		{URN: "urn:ard:show:c", Title: "11KM"},
	}}
}

func TestLoadConfigMissingFileReturnsDefault(t *testing.T) {
	cfg, _ := store.LoadConfig(filepath.Join(t.TempDir(), "fehlt.toml"))
	if cfg.TargetDir != store.DefaultTargetDir {
		t.Errorf("TargetDir = %s", cfg.TargetDir)
	}
}

func TestSaveAndLoadConfigRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := store.SaveConfig(path, sampleConfig()); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.LoadConfig(path); !reflect.DeepEqual(got, sampleConfig()) {
		t.Errorf("geladen = %+v", got)
	}
}

func TestLoadConfigInvalidReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("target_dir = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadConfig(path); err == nil {
		t.Error("Fehler erwartet")
	}
}

func TestMatchSubscriptions(t *testing.T) {
	cases := []struct {
		name, query string
		want        int
	}{
		{"URN exakt", "urn:ard:show:c", 1},
		{"Titel eindeutig", "sandmädchen", 1},
		{"Titel mehrdeutig", "MIMI", 2},
		{"kein Treffer", "xyz", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(sampleConfig().MatchSubscriptions(tc.query)); got != tc.want {
				t.Errorf("Treffer = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSubscribeRejectsDuplicate(t *testing.T) {
	cfg := sampleConfig()
	if cfg.Subscribe(store.Subscription{URN: "urn:ard:show:a"}) {
		t.Error("Duplikat angenommen")
	}
}

func TestUnsubscribeRemoves(t *testing.T) {
	cfg := sampleConfig()
	cfg.Unsubscribe("urn:ard:show:a")
	if cfg.Subscription("urn:ard:show:a") != nil {
		t.Error("Abo noch vorhanden")
	}
}

func TestSubscriptionTargetUsesOwnTarget(t *testing.T) {
	cfg := sampleConfig()
	if got := cfg.SubscriptionTarget(cfg.Subscriptions[0]); got != "/music/Mimi" {
		t.Errorf("Ziel = %s", got)
	}
}

func TestSubscriptionTargetWithoutTargetDirFallsBackToDefault(t *testing.T) {
	cfg := sampleConfig()
	if got := cfg.SubscriptionTarget(cfg.Subscriptions[1]); got != "/music/Mimi und Mio" {
		t.Errorf("Ziel = %s", got)
	}
}
