package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func openWithHistory(t *testing.T, paths ...string) *App {
	t.Helper()
	a := newTestApp(t)
	a.deps.History.RecentPaths = paths
	searchAndOpen(a)
	return a
}

func TestPathPickerSelectsRecentPath(t *testing.T) {
	a := openWithHistory(t, "/a", "/b")
	press(a, "o", "down", "enter")
	if a.episodes.target != "/b" {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}

func TestPathPickerNewPath(t *testing.T) {
	a := openWithHistory(t, "/a")
	press(a, "o", "n")
	typeText(a, "/neu/ziel")
	press(a, "enter")
	if a.episodes.target != "/neu/ziel" {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}

func TestPathPickerExpandsTilde(t *testing.T) {
	a := openWithHistory(t)
	press(a, "o", "n")
	typeText(a, "~/abc")
	press(a, "enter")
	home, _ := os.UserHomeDir()
	if a.episodes.target != filepath.Join(home, "abc") {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}

func TestPathPickerRemovesEntry(t *testing.T) {
	a := openWithHistory(t, "/a", "/b")
	press(a, "o", "x")
	if !reflect.DeepEqual(a.deps.History.RecentPaths, []string{"/b"}) {
		t.Errorf("History = %v", a.deps.History.RecentPaths)
	}
}

func TestPathPickerSetsAboTarget(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "a", "2", "o", "n")
	typeText(a, "/abo/ziel")
	press(a, "enter")
	if got := a.deps.Config.Subscriptions[0].TargetDir; got != "/abo/ziel" {
		t.Errorf("Abo-Ziel = %s", got)
	}
}

func TestSpontaneousTargetDoesNotChangeAboTarget(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "a", "enter", "o", "n")
	typeText(a, filepath.Join(t.TempDir(), "spontan"))
	press(a, "enter", "enter")
	if got := a.deps.Config.Subscriptions[0].TargetDir; got != a.deps.Config.DefaultTarget("Kleine Sendung") {
		t.Errorf("Abo-Ziel = %s", got)
	}
}

func TestPathPickerMakesRelativePathAbsolute(t *testing.T) {
	a := openWithHistory(t)
	press(a, "o", "n")
	typeText(a, "rel/ziel")
	press(a, "enter")
	if !filepath.IsAbs(a.episodes.target) {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}

func TestPathDialogNamesCurrentEpisode(t *testing.T) {
	a := openWithHistory(t)
	press(a, "o")
	if !strings.Contains(a.render(), "Folge: 2026-09-25 – Folge Drei") {
		t.Error("aktuelle Folge fehlt im Pfad-Dialog")
	}
}

func TestPathDialogNamesMarkedCount(t *testing.T) {
	a := openWithHistory(t)
	press(a, "A", "o")
	if !strings.Contains(a.render(), "2 markierte Folge(n)") {
		t.Error("Anzahl markierter Folgen fehlt im Pfad-Dialog")
	}
}

func TestPathDialogNamesAbo(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "a", "2", "o")
	if !strings.Contains(a.render(), "Abo: Kleine Sendung") {
		t.Error("Abo fehlt im Pfad-Dialog")
	}
}

func TestPathPickerKeepsSpacesAndDownloadsThere(t *testing.T) {
	a := openWithHistory(t)
	target := filepath.Join(t.TempDir(), "Mein Ordner", "Kinder Hörspiele")
	press(a, "o", "n")
	typeText(a, "  "+target+" ")
	press(a, "enter", "enter")
	if !fileExists(filepath.Join(target, "2026-09-25 - Folge Drei.mp3")) {
		t.Errorf("Datei fehlt in %q", target)
	}
}

func TestPathPickerTabPrefillsSelectedPath(t *testing.T) {
	a := openWithHistory(t, "/a", "/b")
	press(a, "o", "down", "tab")
	if !a.paths.input.active || a.paths.input.value != "/b" {
		t.Errorf("Eingabe = %q (aktiv %v)", a.paths.input.value, a.paths.input.active)
	}
}

func TestPathPickerTabEditedPathBecomesTarget(t *testing.T) {
	a := openWithHistory(t, "/musik")
	press(a, "o", "tab")
	typeText(a, "/Mimi")
	press(a, "enter")
	if a.episodes.target != "/musik/Mimi" {
		t.Errorf("Ziel = %s", a.episodes.target)
	}
}
