package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

type episodesState struct {
	show           api.Show
	marked         map[string]bool // Episoden-ID → markiert
	cursor         int
	filter         textInput
	includeTrailer bool
	target         string
}

func (a *App) openEpisodes(msg showLoadedMsg) {
	if msg.err != nil {
		a.setError(msg.err)
		return
	}
	a.setStatus("")
	a.episodes = episodesState{
		show: msg.show, marked: map[string]bool{}, includeTrailer: a.deps.Config.IncludeTrailer,
		target: store.ResolveTarget(a.deps.Config, a.deps.History, msg.show.URN, msg.show.Title),
	}
	a.tab, a.inShow = tabSearch, true
}

func (e *episodesState) visible() []api.Episode {
	needle := strings.ToLower(e.filter.value)
	var list []api.Episode
	for _, ep := range e.show.Episodes {
		if ep.IsTrailer() && !e.includeTrailer {
			continue
		}
		if strings.Contains(strings.ToLower(ep.Title), needle) {
			list = append(list, ep)
		}
	}
	return list
}

func (a *App) handleEpisodesKey(key string) tea.Cmd {
	list := a.episodes.visible()
	a.episodes.cursor = moveCursor(a.episodes.cursor, key, len(list))
	switch key {
	case "space":
		a.toggleMark(list)
	case "A":
		a.markAllMissing(list)
	case "/":
		a.episodes.filter.active = true
	case "t":
		a.episodes.includeTrailer = !a.episodes.includeTrailer
	case "o":
		a.openPaths("")
	case "r":
		return a.loadShow(a.episodes.show.URN)
	case "esc":
		a.inShow = false
	case "enter", "d":
		return a.queueEpisodes(list)
	}
	return nil
}

func (a *App) handleFilterInput(msg tea.KeyPressMsg) {
	if _, cancelled := a.episodes.filter.apply(msg); cancelled {
		a.episodes.filter.value = ""
	}
	a.episodes.cursor = 0
}

func (a *App) episodeStatus(ep api.Episode) media.Status {
	return media.StatusOf(a.episodes.target, ep)
}

func (a *App) toggleMark(list []api.Episode) {
	if a.episodes.cursor >= len(list) {
		return
	}
	ep := list[a.episodes.cursor]
	if a.episodeStatus(ep) != media.StatusBlocked {
		a.episodes.marked[ep.ID] = !a.episodes.marked[ep.ID]
	}
}

func (a *App) markAllMissing(list []api.Episode) {
	for _, ep := range list {
		a.episodes.marked[ep.ID] = a.episodeStatus(ep) == media.StatusMissing
	}
}

func (a *App) markedCount() int {
	count := 0
	for _, marked := range a.episodes.marked {
		if marked {
			count++
		}
	}
	return count
}

// Ohne Markierung wird die Folge unter dem Cursor geladen, sofern sie fehlt.
func (a *App) pickedEpisodes(list []api.Episode) []api.Episode {
	var picked []api.Episode
	for _, ep := range a.episodes.show.Episodes {
		if a.episodes.marked[ep.ID] {
			picked = append(picked, ep)
		}
	}
	if len(picked) > 0 || a.episodes.cursor >= len(list) {
		return picked
	}
	if current := list[a.episodes.cursor]; a.episodeStatus(current) == media.StatusMissing {
		return []api.Episode{current}
	}
	return nil
}

func (a *App) queueEpisodes(list []api.Episode) tea.Cmd {
	picked := a.pickedEpisodes(list)
	if len(picked) == 0 {
		a.setStatus("Nichts zu laden – Folge ist vorhanden oder gesperrt. Leertaste markiert.")
		return nil
	}
	for _, ep := range picked {
		a.downloads.enqueue(a.episodes.show.Title, ep, a.episodes.target)
	}
	a.episodes.marked = map[string]bool{}
	a.deps.History.Remember(a.episodes.show.URN, a.episodes.target)
	a.saveHistory()
	a.tab = tabDownloads
	a.setStatus(fmt.Sprintf("%d Folge(n) eingereiht → %s", len(picked), a.episodes.target))
	return a.startNextDownload()
}

func (a *App) renderEpisodes() string {
	e := &a.episodes
	list := e.visible()
	cursor := max(0, min(e.cursor, len(list)-1))
	var b strings.Builder
	b.WriteString(a.styles.bold.Render(e.show.Title) + "  " +
		a.styles.dim.Render(fmt.Sprintf("%d Folgen · %d markiert", len(e.show.Episodes), a.markedCount())) + "\n")
	b.WriteString(a.episodesInfoLine() + "\n\n")
	start, end := window(cursor, len(list))
	for i := start; i < end; i++ {
		b.WriteString(a.row(i == cursor, a.episodeLine(list[i])))
	}
	if len(list) == 0 {
		b.WriteString(a.styles.dim.Render("  (keine Folgen passen zum Filter)") + "\n")
	}
	return b.String()
}

func (a *App) episodesInfoLine() string {
	e := a.episodes
	trailer := a.styles.dim.Render("aus")
	if e.includeTrailer {
		trailer = a.styles.ok.Render("an")
	}
	filter := e.filter.value
	switch {
	case e.filter.active:
		filter += cursorGlyph
	case filter == "":
		filter = "–"
	}
	return fmt.Sprintf("%s %s %s   %s %s   %s %s",
		a.styles.dim.Render("Ziel:"), a.styles.accent.Render(e.target), a.styles.dim.Render("(o)"),
		a.styles.dim.Render("Filter:"), filter, a.styles.dim.Render("Trailer:"), trailer)
}

func (a *App) episodeLine(ep api.Episode) string {
	box := "[ ]"
	if a.episodes.marked[ep.ID] {
		box = a.styles.accent.Render("[x]")
	}
	return fmt.Sprintf("%s  %s  %s  %s %s", box, ep.Date(), formatDuration(ep.DurationSeconds),
		a.statusLabel(a.episodeStatus(ep)), fit(ep.Title, 58))
}

func (a *App) statusLabel(status media.Status) string {
	switch status {
	case media.StatusPresent:
		return a.styles.ok.Render("✓ vorhanden ")
	case media.StatusBlocked:
		return a.styles.err.Render("⊘ gesperrt  ")
	}
	return a.styles.warn.Render("○ fehlt     ")
}

func formatDuration(seconds int) string {
	return fmt.Sprintf("%3d:%02d", seconds/60, seconds%60)
}
