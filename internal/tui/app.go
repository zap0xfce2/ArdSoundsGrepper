// Package tui ist die Bubble-Tea-Oberfläche; Fachlogik liegt in api, media und store.
package tui

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

type Deps struct {
	Client      *api.Client
	Downloader  *media.Downloader
	Config      store.Config
	ConfigErr   error // gesetzt, wenn config.toml ungültig ist → nie speichern
	ConfigPath  string
	History     store.History
	HistoryPath string
	LockPath    string
	Version     string
}

type tab int

const (
	tabSearch tab = iota
	tabAbos
	tabDownloads
	tabCount
)

type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayConfig
	overlayPaths
	overlayQuit
)

const defaultWidth = 100

type App struct {
	deps      Deps
	styles    styles
	tab       tab
	overlay   overlay
	status    string
	statusErr bool
	width     int
	search    searchState
	inShow    bool
	episodes  episodesState
	paths     pathsState
	downloads downloadsState
	abos      abosState
}

func New(deps Deps) *App {
	a := &App{deps: deps, styles: newStyles(), width: defaultWidth, abos: abosState{fresh: map[string]int{}}}
	// Mit Abos interessiert beim Start meist, was neu ist; ohne Abos gibt es nur die Suche.
	if len(deps.Config.Subscriptions) > 0 {
		a.tab = tabAbos
	} else {
		a.search.input.active = true
	}
	if deps.ConfigErr != nil {
		a.setError(fmt.Errorf("Config ungültig, Änderungen werden nicht gespeichert: %w", deps.ConfigErr))
	}
	return a
}

func Run(deps Deps) error {
	_, err := tea.NewProgram(New(deps)).Run()
	return err
}

func (a *App) Init() tea.Cmd {
	if a.tab == tabAbos {
		return a.countAbos()
	}
	return nil
}

func (a *App) View() tea.View {
	view := tea.NewView(a.render())
	view.AltScreen = true
	// Fester dunkler Hintergrund; das Terminal stellt seine Farben beim Beenden wieder her.
	view.BackgroundColor = lipgloss.Color(colorBackground)
	view.ForegroundColor = lipgloss.Color(colorForeground)
	return view
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return a, a.handleKey(msg)
	case tea.PasteMsg:
		a.handlePaste(msg.Content)
	case tea.WindowSizeMsg:
		a.width = msg.Width
	case searchResultMsg:
		a.applySearchResult(msg)
	case showLoadedMsg:
		a.openEpisodes(msg)
	case progressMsg:
		return a, a.applyProgress(msg)
	case downloadDoneMsg:
		return a, a.finishDownload(msg)
	case retryLockMsg:
		return a, a.startNextDownload()
	case aboCountMsg:
		a.applyAboCount(msg)
	case aboSyncMsg:
		return a, a.applyAboSync(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	a.setStatus("")
	if msg.String() == "ctrl+c" { // muss auch in Eingabefeldern beenden
		return a.requestQuit()
	}
	if a.textInputActive() {
		return a.handleTextKey(msg)
	}
	key := msg.String()
	if a.overlay != overlayNone {
		return a.handleOverlayKey(key)
	}
	if cmd, handled := a.handleGlobalKey(key); handled {
		return cmd
	}
	return a.handleTabKey(key)
}

func (a *App) handleTabKey(key string) tea.Cmd {
	switch {
	case a.tab == tabSearch && a.inShow:
		return a.handleEpisodesKey(key)
	case a.tab == tabSearch:
		return a.handleSearchKey(key)
	case a.tab == tabAbos:
		return a.handleAbosKey(key)
	}
	return a.handleDownloadsKey(key)
}

func (a *App) handleGlobalKey(key string) (tea.Cmd, bool) {
	switch key {
	case "1", "2", "3":
		return a.switchTab(tab(key[0] - '1')), true
	case "right", "tab":
		return a.switchTab((a.tab + 1) % tabCount), true
	case "left":
		return a.switchTab((a.tab + tabCount - 1) % tabCount), true
	case "?":
		a.overlay = overlayHelp
	case "c":
		a.overlay = overlayConfig
	case "q", "ctrl+c":
		return a.requestQuit(), true
	default:
		return nil, false
	}
	return nil, true
}

func (a *App) switchTab(target tab) tea.Cmd {
	a.tab = target
	// Vor der ersten Suche gibt es dort nichts anderes zu tun als zu tippen.
	if target == tabSearch && !a.inShow && a.search.query == "" {
		a.search.input.active = true
	}
	if target == tabAbos {
		return a.countAbos()
	}
	return nil
}

func (a *App) handleOverlayKey(key string) tea.Cmd {
	switch a.overlay {
	case overlayPaths:
		return a.handlePathsKey(key)
	case overlayQuit:
		return a.handleQuitKey(key)
	}
	if key == "esc" || key == "?" || key == "c" {
		a.overlay = overlayNone
	}
	return nil
}

func (a *App) requestQuit() tea.Cmd {
	if !a.downloads.busy() {
		return tea.Quit
	}
	a.overlay = overlayQuit
	return nil
}

// Beim Beenden erst den laufenden Download abbrechen lassen, damit seine .part gelöscht wird.
func (a *App) handleQuitKey(key string) tea.Cmd {
	a.overlay = overlayNone
	if key != "j" {
		return nil
	}
	a.downloads.quitting = true
	a.downloads.cancelAll()
	if a.downloads.running == nil {
		return tea.Quit
	}
	return nil
}

func (a *App) setStatus(text string) { a.status, a.statusErr = text, false }

func (a *App) setError(err error) {
	if err != nil {
		a.status, a.statusErr = err.Error(), true
	}
}

func (a *App) saveConfig() bool {
	if a.deps.ConfigErr != nil {
		a.setError(errors.New("Config ist ungültig – Änderung nicht gespeichert"))
		return false
	}
	if err := store.SaveConfig(a.deps.ConfigPath, a.deps.Config); err != nil {
		a.setError(err)
		return false
	}
	return true
}

func (a *App) saveHistory() {
	a.setError(store.SaveHistory(a.deps.HistoryPath, a.deps.History))
}
