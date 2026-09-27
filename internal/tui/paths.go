package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/store"
)

// pathsState: Pfad-Dialog für die Folgenansicht (aboURN leer) oder für ein Abo-Ziel.
type pathsState struct {
	cursor int
	input  textInput
	aboURN string
}

func (a *App) openPaths(aboURN string) {
	a.overlay = overlayPaths
	a.paths = pathsState{aboURN: aboURN}
}

func (a *App) handlePathsKey(key string) tea.Cmd {
	recent := a.deps.History.RecentPaths
	a.paths.cursor = moveCursor(a.paths.cursor, key, len(recent)+1)
	onNewRow := a.paths.cursor == len(recent)
	switch {
	case key == "esc":
		a.overlay = overlayNone
	case key == "n" || (key == "enter" && onNewRow):
		a.paths.cursor = len(recent)
		a.paths.input = textInput{active: true}
	case key == "tab" && !onNewRow: // Vorschlag aus der History als Ausgangspunkt bearbeiten
		a.paths.input = textInput{value: recent[a.paths.cursor], active: true}
		a.paths.cursor = len(recent)
	case key == "enter":
		a.applyTarget(recent[a.paths.cursor])
	case key == "x" && !onNewRow:
		a.deps.History.Forget(recent[a.paths.cursor])
		a.saveHistory()
	}
	return nil
}

func (a *App) handlePathInput(msg tea.KeyPressMsg) {
	submitted, _ := a.paths.input.apply(msg)
	path := strings.TrimSpace(a.paths.input.value)
	if !submitted || path == "" {
		return
	}
	// Relative Pfade würden bei Cron-sync relativ zu dessen Arbeitsverzeichnis landen.
	path = store.ExpandHome(path)
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	a.applyTarget(path)
}

// applyTarget setzt das Ziel der Folgenansicht oder – im Abo-Modus – das feste Abo-Ziel.
func (a *App) applyTarget(path string) {
	a.overlay = overlayNone
	if a.paths.aboURN == "" {
		a.episodes.target = path
		a.setStatus("Ziel: " + path + " (wird bei Bedarf angelegt)")
		return
	}
	sub := a.deps.Config.Subscription(a.paths.aboURN)
	if sub == nil {
		return
	}
	sub.TargetDir = path
	if a.saveConfig() {
		a.setStatus(fmt.Sprintf("Abo-Ziel: %s → %s", sub.Title, path))
	}
	a.deps.History.Remember(sub.URN, path)
	a.saveHistory()
}

func (a *App) renderPaths() string {
	title, current := "Zielordner wählen", a.episodes.target
	if sub := a.deps.Config.Subscription(a.paths.aboURN); sub != nil {
		title, current = "Abo-Ziel für "+sub.Title, a.deps.Config.SubscriptionTarget(*sub)
	}
	var b strings.Builder
	b.WriteString(a.styles.title.Render(title) + "  " +
		a.styles.dim.Render(fmt.Sprintf("zuletzt verwendet (max. %d)", store.MaxRecentPaths)) + "\n")
	b.WriteString(a.styles.accent.Render(a.pathContext()) + "\n\n")
	recent := a.deps.History.RecentPaths
	for i, path := range recent {
		mark := ""
		if path == current {
			mark = a.styles.ok.Render(" ● aktuell")
		}
		b.WriteString(a.row(i == a.paths.cursor, fmt.Sprintf("%2d  %s%s", i+1, path, mark)))
	}
	newRow := a.styles.accent.Render(" +") + "  neuer Pfad …"
	if a.paths.input.active {
		newRow = a.styles.accent.Render(" +") + "  " + a.paths.input.value + cursorGlyph
	}
	b.WriteString(a.row(a.paths.cursor == len(recent), newRow))
	return b.String()
}

// pathContext nennt, wofür der Pfad gilt: Abo, markierte Folgen oder die Folge unter dem Cursor.
func (a *App) pathContext() string {
	if sub := a.deps.Config.Subscription(a.paths.aboURN); sub != nil {
		return "Abo: " + sub.Title + " (gilt für alle künftigen Downloads dieses Abos)"
	}
	e := &a.episodes
	if marked := a.markedCount(); marked > 0 {
		return fmt.Sprintf("Sendung: %s · %d markierte Folge(n)", e.show.Title, marked)
	}
	if list := e.visible(); e.cursor < len(list) {
		ep := list[e.cursor]
		return fmt.Sprintf("Sendung: %s · Folge: %s – %s", e.show.Title, ep.Date(), ep.Title)
	}
	return "Sendung: " + e.show.Title
}
