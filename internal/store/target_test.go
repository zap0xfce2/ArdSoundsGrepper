package store_test

import (
	"testing"

	"ardsoundsgrepper/internal/store"
)

func TestResolveTarget(t *testing.T) {
	cfg := store.Config{TargetDir: "/music", Subscriptions: []store.Subscription{
		{URN: "urn:abo", Title: "Abo", TargetDir: "/abo"}, {URN: "urn:leer", Title: "Leer"},
	}}
	full := store.History{RecentPaths: []string{"/recent"}, ShowPaths: map[string]string{"urn:abo": "/show", "urn:x": "/show", "urn:leer": "/show"}}
	cases := []struct {
		name string
		hist store.History
		urn  string
		want string
	}{
		{"Abo-Ziel gewinnt", full, "urn:abo", "/abo"},
		{"Abo ohne target_dir nutzt Default wie sync", full, "urn:leer", "/music/Leer"},
		{"Pfad der Sendung", full, "urn:x", "/show"},
		{"zuletzt genutzt", full, "urn:neu", "/recent"},
		{"Default", store.History{}, "urn:neu", "/music/Neue Sendung"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := store.ResolveTarget(cfg, tc.hist, tc.urn, "Neue Sendung"); got != tc.want {
				t.Errorf("Ziel = %s, want %s", got, tc.want)
			}
		})
	}
}
