package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/api"
)

type searchState struct {
	input   textInput
	query   string
	hits    []api.ShowHit
	cursor  int
	loading bool
}

type searchResultMsg struct {
	hits []api.ShowHit
	err  error
}

type showLoadedMsg struct {
	show api.Show
	err  error
}

func (a *App) handleSearchInput(msg tea.KeyPressMsg) tea.Cmd {
	if submitted, _ := a.search.input.apply(msg); !submitted {
		return nil
	}
	return a.runSearch(strings.TrimSpace(a.search.input.value))
}

// URL/URN/ID öffnen die Sendung direkt, alles andere wird gesucht.
func (a *App) runSearch(term string) tea.Cmd {
	if term == "" {
		return nil
	}
	a.search.query = term
	if id, ok := api.ParseShowID(term); ok {
		return a.loadShow(id)
	}
	a.search.loading = true
	client := a.deps.Client
	return func() tea.Msg {
		hits, err := client.SearchShows(context.Background(), term)
		return searchResultMsg{hits: hits, err: err}
	}
}

func (a *App) applySearchResult(msg searchResultMsg) {
	a.search.loading = false
	if msg.err != nil {
		a.setError(msg.err)
		return
	}
	a.search.hits, a.search.cursor = msg.hits, 0
	if len(msg.hits) == 0 {
		a.setStatus(fmt.Sprintf("Keine Sendung gefunden für „%s“", a.search.query))
	}
}

func (a *App) handleSearchKey(key string) tea.Cmd {
	a.search.cursor = moveCursor(a.search.cursor, key, len(a.search.hits))
	switch key {
	case "i", "/":
		a.search.input = textInput{active: true}
		return nil
	case "r":
		return a.runSearch(a.search.query)
	}
	if a.search.cursor >= len(a.search.hits) {
		return nil
	}
	hit := a.search.hits[a.search.cursor]
	switch key {
	case "enter":
		return a.loadShow(hit.URN)
	case "a":
		a.subscribe(hit.URN, hit.Title)
	}
	return nil
}

func (a *App) loadShow(id string) tea.Cmd {
	a.setStatus("lädt Sendung …")
	client := a.deps.Client
	return func() tea.Msg {
		show, err := client.FetchShow(context.Background(), id)
		return showLoadedMsg{show: show, err: err}
	}
}
