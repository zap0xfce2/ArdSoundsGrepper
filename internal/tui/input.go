package tui

import tea "charm.land/bubbletea/v2"

type textInput struct {
	value  string
	active bool
}

// apply verarbeitet einen Tastendruck; Enter/Esc beenden die Eingabe.
func (t *textInput) apply(msg tea.KeyPressMsg) (submitted, cancelled bool) {
	switch msg.String() {
	case "enter":
		t.active = false
		return true, false
	case "esc":
		t.active = false
		return false, true
	case "backspace":
		if runes := []rune(t.value); len(runes) > 0 {
			t.value = string(runes[:len(runes)-1])
		}
		return false, false
	}
	t.value += msg.Text
	return false, false
}

func (a *App) textInputActive() bool {
	return a.search.input.active || a.episodes.filter.active || a.paths.input.active
}

func (a *App) handleTextKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case a.search.input.active:
		return a.handleSearchInput(msg)
	case a.episodes.filter.active:
		a.handleFilterInput(msg)
	case a.paths.input.active:
		a.handlePathInput(msg)
	}
	return nil
}

// Bubble Tea liefert Eingefügtes per Bracketed Paste als eigene Message, nicht als Tasten.
func (a *App) handlePaste(text string) {
	switch {
	case a.search.input.active:
		a.search.input.value += text
	case a.episodes.filter.active:
		a.episodes.filter.value += text
	case a.paths.input.active:
		a.paths.input.value += text
	}
}

// moveCursor bewegt eine Listenauswahl per Pfeil- oder vi-Taste und hält sie im gültigen Bereich.
func moveCursor(cursor int, key string, length int) int {
	switch key {
	case "down", "j":
		cursor++
	case "up", "k":
		cursor--
	}
	return max(0, min(cursor, length-1))
}
