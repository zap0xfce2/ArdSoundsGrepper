package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ardsoundsgrepper/internal/apitest"
	"ardsoundsgrepper/internal/store"
)

func TestRightArrowSwitchesToAbos(t *testing.T) {
	a := newTestApp(t)
	press(a, "esc", "right")
	if a.tab != tabAbos {
		t.Errorf("tab = %v", a.tab)
	}
}

func TestLeftArrowWrapsToDownloads(t *testing.T) {
	a := newTestApp(t)
	press(a, "esc", "left")
	if a.tab != tabDownloads {
		t.Errorf("tab = %v", a.tab)
	}
}

func TestNumberKeySelectsTab(t *testing.T) {
	a := newTestApp(t)
	press(a, "esc", "3")
	if a.tab != tabDownloads {
		t.Errorf("tab = %v", a.tab)
	}
}

func TestViewShowsVersion(t *testing.T) {
	if !strings.Contains(newTestApp(t).render(), "vTEST") {
		t.Error("Version fehlt")
	}
}

func TestNarrowWindowRendersWithoutPanic(t *testing.T) {
	a := newTestApp(t)
	a.Update(tea.WindowSizeMsg{Width: 5, Height: 5})
	if !strings.Contains(a.render(), "vTEST") {
		t.Error("Version fehlt")
	}
}

func TestQuitWithoutDownloadsQuits(t *testing.T) {
	a := newTestApp(t)
	press(a, "esc")
	_, cmd := a.Update(keyMsg("q"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("kein Quit")
	}
}

func TestInvalidConfigIsNeverSaved(t *testing.T) {
	a := newTestApp(t)
	a.deps.ConfigErr = errors.New("kaputt")
	typeText(a, "klein")
	press(a, "enter", "a")
	if _, err := os.Stat(a.deps.ConfigPath); err == nil {
		t.Error("Config wurde geschrieben")
	}
}

func TestPasteIntoSearchInput(t *testing.T) {
	a := newTestApp(t)
	a.Update(tea.PasteMsg{Content: "klein"})
	if a.search.input.value != "klein" {
		t.Errorf("Eingabe = %q", a.search.input.value)
	}
}

func TestCtrlCQuitsWhileTyping(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("kein Command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("kein Quit")
	}
}

func TestViewSetsDarkBackground(t *testing.T) {
	if got := newTestApp(t).View().BackgroundColor; got != lipgloss.Color(colorBackground) {
		t.Errorf("Hintergrund = %v", got)
	}
}

func appWithAbo(t *testing.T) *App {
	t.Helper()
	deps := newTestApp(t).deps
	deps.Config.Subscriptions = []store.Subscription{{URN: apitest.SmallURN, Title: "Kleine Sendung"}}
	return New(deps)
}

func TestStartWithoutAbosFocusesSearch(t *testing.T) {
	if a := newTestApp(t); a.tab != tabSearch || !a.search.input.active {
		t.Errorf("tab = %v, Suchfokus = %v", a.tab, a.search.input.active)
	}
}

func TestStartWithAbosOpensAbosTab(t *testing.T) {
	if got := appWithAbo(t).tab; got != tabAbos {
		t.Errorf("tab = %v", got)
	}
}

func TestStartWithAbosDoesNotFocusSearch(t *testing.T) {
	if appWithAbo(t).search.input.active {
		t.Error("Suchfeld hat Fokus")
	}
}

func TestStartWithAbosCountsFreshEpisodes(t *testing.T) {
	a := appWithAbo(t)
	drain(a, a.Init())
	if got := a.abos.fresh[apitest.SmallURN]; got != 2 {
		t.Errorf("neu = %d, want 2", got)
	}
}

func TestSwitchToSearchFocusesInputBeforeFirstSearch(t *testing.T) {
	a := appWithAbo(t)
	press(a, "1")
	if !a.search.input.active {
		t.Error("Suchfeld hat keinen Fokus")
	}
}

func TestSwitchToSearchKeepsResultsWithoutFocus(t *testing.T) {
	a := newTestApp(t)
	typeText(a, "klein")
	press(a, "enter", "2", "1")
	if a.search.input.active {
		t.Error("Suchfeld hat Fokus trotz vorhandener Treffer")
	}
}
