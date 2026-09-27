package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"ardsoundsgrepper/internal/store"
)

const (
	colorDim        = "#6b6f86"
	colorAccent     = "#8aa4ff"
	colorOK         = "#7fd49a"
	colorWarn       = "#e8c46a"
	colorErr        = "#f07f8a"
	colorPink       = "#e29cd8"
	colorSelected   = "#2a3257"
	colorBackground = "#16161e"
	colorForeground = "#c8cbd9"
	listRows        = 20
	cursorGlyph     = "█"
)

const (
	hintSearchInput = "Enter suchen · Esc Liste"
	hintSearch      = "↑↓ wählen · Enter öffnen · a abonnieren · i neue Suche · ←→ Reiter · ? Hilfe"
	hintEpisodes    = "Enter laden · Leertaste markieren · A alle fehlenden · / filtern · t Trailer · o Ziel · Esc zurück"
	hintFilter      = "tippen filtert live · Enter übernehmen · Esc verwerfen"
	hintPaths       = "↑↓ wählen · Enter übernehmen · Tab bearbeiten · n neuer Pfad · x aus History entfernen · Esc schließen"
	hintPathInput   = "Pfad tippen · Enter übernehmen (wird bei Bedarf angelegt) · Esc abbrechen"
	hintDownloads   = "↑↓ wählen · x abbrechen · r erneut · ←→ Reiter"
	hintAbos        = "↑↓ wählen · Enter öffnen · o Ziel ändern · s alle prüfen & laden · r neu zählen · d entfernen"
	hintOverlay     = "Esc schließt"
)

const helpText = `Tasten
  1 2 3 / ← → / Tab   Reiter wechseln
  ↑ ↓ / j k           bewegen
  Enter               öffnen · Folgen laden · übernehmen
  Esc                 zurück / schließen
  i                   Suche: neuer Begriff
  a                   Suche: Sendung abonnieren
  Leertaste           Folge markieren
  A                   alle fehlenden markieren
  /                   Folgen filtern
  t                   Trailer ein/aus
  o                   Zielordner wählen (Folgen) · Abo-Ziel ändern (Abos)
  n / Tab / x         Pfad-Dialog: neuer Pfad / Eintrag bearbeiten / aus History entfernen
  x / r               Downloads: abbrechen / erneut
  s / r / d           Abos: alle laden / neu zählen / entfernen
  c                   Config anzeigen
  q                   beenden`

type styles struct {
	dim, accent, ok, warn, err, pink, bold, title, selected, tabOn, tabOff, badge lipgloss.Style
}

func newStyles() styles {
	fg := func(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
	onColor := func(hex string) lipgloss.Style {
		return lipgloss.NewStyle().Bold(true).Padding(0, 1).
			Foreground(lipgloss.Color(colorBackground)).Background(lipgloss.Color(hex))
	}
	return styles{
		dim: fg(colorDim), accent: fg(colorAccent), ok: fg(colorOK), warn: fg(colorWarn), err: fg(colorErr), pink: fg(colorPink),
		bold: lipgloss.NewStyle().Bold(true), title: fg(colorAccent).Bold(true),
		selected: lipgloss.NewStyle().Background(lipgloss.Color(colorSelected)),
		tabOn:    onColor(colorAccent), tabOff: fg(colorDim).Padding(0, 1), badge: onColor(colorPink),
	}
}

func (a *App) render() string {
	return a.renderHeader() + a.renderBody() + a.renderFooter()
}

func (a *App) renderBody() string {
	switch {
	case a.overlay == overlayPaths:
		return a.renderPaths()
	case a.overlay == overlayHelp:
		return helpText + "\n"
	case a.overlay == overlayConfig:
		return a.renderConfig()
	case a.overlay == overlayQuit:
		return a.styles.warn.Render("Downloads laufen – wirklich beenden? [j/N]") + "\n"
	case a.tab == tabSearch && a.inShow:
		return a.renderEpisodes()
	case a.tab == tabSearch:
		return a.renderSearch()
	case a.tab == tabAbos:
		return a.renderAbos()
	}
	return a.renderDownloads()
}

func (a *App) renderHeader() string {
	labels := []string{"1 Suche", fmt.Sprintf("2 Abos (%d)", len(a.deps.Config.Subscriptions)), a.downloadsTabLabel()}
	parts := []string{a.styles.badge.Render("ArdSoundsGrepper")}
	for i, label := range labels {
		style := a.styles.tabOff
		if tab(i) == a.tab {
			style = a.styles.tabOn
		}
		parts = append(parts, style.Render(label))
	}
	return strings.Join(parts, " ") + "\n" + a.rule() + "\n"
}

func (a *App) downloadsTabLabel() string {
	if active := a.activeDownloads(); active > 0 {
		return fmt.Sprintf("3 Downloads (%d)", active)
	}
	return "3 Downloads"
}

func (a *App) renderFooter() string {
	hint := a.currentHint()
	gap := max(1, a.width-lipgloss.Width(hint)-lipgloss.Width(a.deps.Version))
	status := a.styles.pink.Render(a.status)
	if a.statusErr {
		status = a.styles.err.Render(a.status)
	}
	return "\n" + a.rule() + "\n" + a.styles.dim.Render(hint+strings.Repeat(" ", gap)+a.deps.Version) + "\n" + status
}

func (a *App) currentHint() string {
	switch {
	case a.search.input.active:
		return hintSearchInput
	case a.episodes.filter.active:
		return hintFilter
	case a.paths.input.active:
		return hintPathInput
	case a.overlay == overlayPaths:
		return hintPaths
	case a.overlay != overlayNone:
		return hintOverlay
	case a.tab == tabSearch && a.inShow:
		return hintEpisodes
	case a.tab == tabSearch:
		return hintSearch
	case a.tab == tabAbos:
		return hintAbos
	}
	return hintDownloads
}

func (a *App) rule() string {
	return a.styles.dim.Render(strings.Repeat("─", max(0, a.width)))
}

func (a *App) row(selected bool, text string) string {
	if selected {
		return a.styles.selected.Render(a.styles.accent.Render("▸ ")+text) + "\n"
	}
	return "  " + text + "\n"
}

// window liefert den sichtbaren Ausschnitt einer Liste um den Cursor.
func window(cursor, length int) (start, end int) {
	start = max(0, min(cursor-listRows/2, length-listRows))
	return start, min(length, start+listRows)
}

func fit(text string, width int) string {
	runes := []rune(text)
	if len(runes) > width {
		return string(runes[:width-1]) + "…"
	}
	return text + strings.Repeat(" ", width-len(runes))
}

func (a *App) renderSearch() string {
	var b strings.Builder
	b.WriteString(a.styles.bold.Render("Sendung suchen") + "  " + a.styles.dim.Render("Suchbegriff, Sendungs-URL oder URN") + "\n\n")
	cursor := ""
	if a.search.input.active {
		cursor = cursorGlyph
	}
	b.WriteString(" " + a.styles.accent.Render("›") + " " + a.search.input.value + cursor + "\n\n")
	if a.search.loading {
		return b.String() + a.styles.dim.Render("   suche …") + "\n"
	}
	for i, hit := range a.search.hits {
		line := fmt.Sprintf("%2d  %s %s %4d Folgen%s", i+1, fit(hit.Title, 56), fit(hit.Sender, 12), hit.EpisodeCount, a.aboMarker(hit.URN))
		b.WriteString(a.row(!a.search.input.active && i == a.search.cursor, line))
	}
	return b.String()
}

func (a *App) aboMarker(urn string) string {
	if a.deps.Config.Subscription(urn) != nil {
		return a.styles.pink.Render(" ★")
	}
	return ""
}

func (a *App) renderConfig() string {
	text, err := store.EncodeConfig(a.deps.Config)
	if err != nil {
		text = err.Error()
	}
	return a.styles.title.Render("Config") + "  " + a.styles.dim.Render(a.deps.ConfigPath) + "\n\n" + text
}
