package tui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/apitest"
	"ardsoundsgrepper/internal/store"
)

func TestDownloadFailureMarksItemFailed(t *testing.T) {
	a := newTestApp(t, apitest.BrokenShow())
	typeText(a, "kaputt")
	press(a, "enter", "enter", "enter")
	if a.downloads.items[0].state != dlFailed {
		t.Errorf("Status = %v", a.downloads.items[0].state)
	}
}

func TestRetryRequeuesItem(t *testing.T) {
	a := openSmall(t)
	press(a, "enter")
	item := a.downloads.items[0]
	_ = os.Remove(item.path())
	item.state = dlFailed
	press(a, "r")
	if item.state != dlDone {
		t.Errorf("Status = %v", item.state)
	}
}

func lockedApp(t *testing.T) *App {
	t.Helper()
	a := openSmall(t)
	unlock, err := store.TryLock(a.deps.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unlock)
	a.Update(keyMsg("enter")) // Tick-Command bewusst nicht ausführen
	return a
}

func TestLockedDownloadWaits(t *testing.T) {
	if got := lockedApp(t).downloads.items[0].state; got != dlWaiting {
		t.Errorf("Status = %v", got)
	}
}

func TestLockedDownloadShowsStatus(t *testing.T) {
	if got := lockedApp(t).status; !strings.Contains(got, "sync läuft gerade") {
		t.Errorf("Status = %q", got)
	}
}

func TestCancelWaitingItem(t *testing.T) {
	a := lockedApp(t)
	press(a, "x")
	if got := a.downloads.items[0].state; got != dlCancelled {
		t.Errorf("Status = %v", got)
	}
}

func TestQuitWithPendingDownloadsAsks(t *testing.T) {
	a := lockedApp(t)
	press(a, "q")
	if a.overlay != overlayQuit {
		t.Errorf("Overlay = %v", a.overlay)
	}
}

func TestQuitConfirmCancelsAndQuits(t *testing.T) {
	a := lockedApp(t)
	press(a, "q")
	_, cmd := a.Update(keyMsg("j"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("kein Quit")
	}
}

func TestRenderDownloadsUnknownLength(t *testing.T) {
	a := newTestApp(t)
	a.downloads.items = []*downloadItem{{state: dlDone, done: 1000, total: -1}}
	a.tab = tabDownloads
	if !strings.Contains(a.render(), "Gesamt") {
		t.Error("Gesamtzeile fehlt")
	}
}
