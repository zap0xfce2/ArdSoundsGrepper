package tui

import (
	"os"
	"testing"

	"ardsoundsgrepper/internal/apitest"
)

func subscribedApp(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "a", "2")
	return a
}

func TestAbosTabCountsFreshEpisodes(t *testing.T) {
	if got := subscribedApp(t).abos.fresh[apitest.SmallURN]; got != 2 {
		t.Errorf("neu = %d, want 2", got)
	}
}

func TestAbosSyncDownloadsIntoAboTarget(t *testing.T) {
	a := subscribedApp(t)
	press(a, "s")
	entries, _ := os.ReadDir(a.deps.Config.SubscriptionTarget(a.deps.Config.Subscriptions[0]))
	if len(entries) != 2 {
		t.Errorf("Dateien = %d, want 2", len(entries))
	}
}

func TestAbosRemove(t *testing.T) {
	a := subscribedApp(t)
	press(a, "d")
	if len(a.deps.Config.Subscriptions) != 0 {
		t.Error("Abo noch da")
	}
}

func TestAbosEnterOpensShow(t *testing.T) {
	a := subscribedApp(t)
	press(a, "enter")
	if !a.inShow || a.tab != tabSearch {
		t.Error("Sendung nicht geöffnet")
	}
}

func TestAbosOpenShowPreselectsAboTarget(t *testing.T) {
	a := subscribedApp(t)
	press(a, "enter")
	if a.episodes.target != a.deps.Config.SubscriptionTarget(a.deps.Config.Subscriptions[0]) {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}
