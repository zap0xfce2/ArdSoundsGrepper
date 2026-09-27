package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/apitest"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

var specialKeys = map[string]rune{
	"enter": tea.KeyEnter, "esc": tea.KeyEscape, "space": tea.KeySpace, "backspace": tea.KeyBackspace,
	"left": tea.KeyLeft, "right": tea.KeyRight, "up": tea.KeyUp, "down": tea.KeyDown, "tab": tea.KeyTab,
}

func newTestApp(t *testing.T, shows ...apitest.Show) *App {
	t.Helper()
	if len(shows) == 0 {
		shows = []apitest.Show{apitest.SmallShow()}
	}
	srv := apitest.NewServer(t, shows...)
	dir := t.TempDir()
	cfg := store.DefaultConfig()
	cfg.TargetDir = filepath.Join(dir, "music")
	return New(Deps{
		Client: api.NewClient(srv.Endpoint()), Downloader: media.NewDownloader(), Config: cfg,
		ConfigPath: filepath.Join(dir, "config.toml"), HistoryPath: filepath.Join(dir, "history.json"),
		LockPath: filepath.Join(dir, "sync.lock"), Version: "vTEST",
	})
}

func keyMsg(key string) tea.KeyPressMsg {
	if code, ok := specialKeys[key]; ok {
		msg := tea.KeyPressMsg{Code: code}
		if code == tea.KeySpace {
			msg.Text = " "
		}
		return msg
	}
	return tea.KeyPressMsg{Code: []rune(key)[0], Text: key}
}

// press schickt Tasten und führt entstehende Commands synchron aus (inkl. Downloads).
func press(a *App, keys ...string) {
	for _, key := range keys {
		_, cmd := a.Update(keyMsg(key))
		drain(a, cmd)
	}
}

func typeText(a *App, text string) {
	for _, r := range text {
		press(a, string(r))
	}
}

func drain(a *App, cmd tea.Cmd) {
	for queue := []tea.Cmd{cmd}; len(queue) > 0; {
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}
		msg := next()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		if msg != nil {
			_, follow := a.Update(msg)
			queue = append(queue, follow)
		}
	}
}

// searchAndOpen sucht "klein" und öffnet den ersten Treffer.
func searchAndOpen(a *App) {
	typeText(a, "klein")
	press(a, "enter", "enter")
}
