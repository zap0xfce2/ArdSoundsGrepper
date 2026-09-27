package tui

import (
	"strings"
	"testing"

	"ardsoundsgrepper/internal/apitest"
)

func TestSearchShowsHits(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter")
	if len(a.search.hits) != 1 {
		t.Errorf("Treffer = %d", len(a.search.hits))
	}
}

func TestSearchWithoutHitsShowsMessage(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "xyz")
	press(a, "enter")
	if !strings.Contains(a.status, "Keine Sendung gefunden") {
		t.Errorf("Status = %q", a.status)
	}
}

func TestSearchEnterOpensShow(t *testing.T) {
	a := newTestApp(t)
	searchAndOpen(a)
	if a.episodes.show.URN != apitest.SmallURN {
		t.Errorf("Sendung = %q", a.episodes.show.URN)
	}
}

func TestSearchURNOpensShowDirectly(t *testing.T) {
	a := newTestApp(t)
	typeText(a, apitest.SmallURN)
	press(a, "enter")
	if !a.inShow {
		t.Error("Sendung nicht geöffnet")
	}
}

func TestSubscribeUsesDefaultTarget(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "a")
	want := a.deps.Config.DefaultTarget("Kleine Sendung")
	if got := a.deps.Config.Subscriptions[0].TargetDir; got != want {
		t.Errorf("Ziel = %s, want %s", got, want)
	}
}
