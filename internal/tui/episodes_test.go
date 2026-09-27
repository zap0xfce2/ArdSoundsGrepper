package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openSmall(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t)
	searchAndOpen(a)
	return a
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestEpisodesHideTrailerByDefault(t *testing.T) {
	if got := len(openSmall(t).episodes.visible()); got != 3 {
		t.Errorf("sichtbar = %d, want 3", got)
	}
}

func TestEpisodesTrailerToggle(t *testing.T) {
	a := openSmall(t)
	press(a, "t")
	if got := len(a.episodes.visible()); got != 4 {
		t.Errorf("sichtbar = %d, want 4", got)
	}
}

func TestEpisodesLiveFilter(t *testing.T) {
	a := openSmall(t)
	press(a, "/")
	typeText(a, "zwei")
	if got := len(a.episodes.visible()); got != 1 {
		t.Errorf("sichtbar = %d, want 1", got)
	}
}

func TestEpisodesFilterEscClears(t *testing.T) {
	a := openSmall(t)
	press(a, "/")
	typeText(a, "zwei")
	press(a, "esc")
	if a.episodes.filter.value != "" {
		t.Errorf("Filter = %q", a.episodes.filter.value)
	}
}

func TestEnterDownloadsCurrentEpisode(t *testing.T) {
	a := openSmall(t)
	press(a, "enter")
	if !fileExists(filepath.Join(a.episodes.target, "2026-09-25 - Folge Drei.mp3")) {
		t.Error("Folge Drei fehlt")
	}
}

func TestEnterWithMarksDownloadsOnlyMarked(t *testing.T) {
	a := openSmall(t)
	press(a, "down", "space", "enter")
	if fileExists(filepath.Join(a.episodes.target, "2026-09-25 - Folge Drei.mp3")) {
		t.Error("unmarkierte Folge geladen")
	}
}

func TestMarkAllMissingMarksDownloadable(t *testing.T) {
	a := openSmall(t)
	press(a, "A")
	if got := a.markedCount(); got != 2 {
		t.Errorf("markiert = %d, want 2", got)
	}
}

func TestBlockedEpisodeCannotBeMarked(t *testing.T) {
	a := openSmall(t)
	press(a, "down", "down", "space")
	if got := a.markedCount(); got != 0 {
		t.Errorf("markiert = %d", got)
	}
}

func TestEnterOnPresentEpisodeShowsHint(t *testing.T) {
	a := openSmall(t)
	press(a, "enter", "1", "enter")
	if !strings.Contains(a.status, "Nichts zu laden") {
		t.Errorf("Status = %q", a.status)
	}
}

func TestDownloadRemembersTarget(t *testing.T) {
	a := openSmall(t)
	press(a, "enter")
	if a.deps.History.RecentPaths[0] != a.episodes.target {
		t.Errorf("History = %v", a.deps.History.RecentPaths)
	}
}
