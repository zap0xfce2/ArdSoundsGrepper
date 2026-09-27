package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

type abosState struct {
	cursor int
	fresh  map[string]int // URN → fehlende Folgen; fehlender Key = noch nicht gezählt
}

type aboCountMsg struct {
	urn   string
	fresh int
	err   error
}

type aboSyncMsg struct {
	sub  store.Subscription
	show api.Show
	err  error
}

// Neues Abo übernimmt den letzten Pfad der Sendung, sonst den Default.
func (a *App) subscribe(urn, title string) {
	target := a.deps.History.ShowPaths[urn]
	if target == "" {
		target = a.deps.Config.DefaultTarget(title)
	}
	if !a.deps.Config.Subscribe(store.Subscription{URN: urn, Title: title, TargetDir: target}) {
		a.setStatus("Bereits abonniert: " + title)
		return
	}
	if a.saveConfig() {
		a.setStatus(fmt.Sprintf("Abonniert: %s → %s (im Abo-Tab mit o ändern)", title, target))
	}
}

// fetchSubscriptions lädt alle Abos parallel im Hintergrund und baut je Abo eine Message.
func (a *App) fetchSubscriptions(build func(store.Subscription, api.Show, error) tea.Msg) tea.Cmd {
	client := a.deps.Client
	cmds := make([]tea.Cmd, 0, len(a.deps.Config.Subscriptions))
	for _, sub := range a.deps.Config.Subscriptions {
		cmds = append(cmds, func() tea.Msg {
			show, err := client.FetchShow(context.Background(), sub.URN)
			return build(sub, show, err)
		})
	}
	return tea.Batch(cmds...)
}

func (a *App) countAbos() tea.Cmd {
	cfg := a.deps.Config
	return a.fetchSubscriptions(func(sub store.Subscription, show api.Show, err error) tea.Msg {
		missing := media.MissingEpisodes(show, cfg.SubscriptionTarget(sub), cfg.IncludeTrailer)
		return aboCountMsg{urn: sub.URN, fresh: len(missing), err: err}
	})
}

func (a *App) applyAboCount(msg aboCountMsg) {
	if msg.err != nil {
		a.setError(msg.err)
		return
	}
	a.abos.fresh[msg.urn] = msg.fresh
}

func (a *App) syncAllAbos() tea.Cmd {
	a.setStatus("prüfe Abos …")
	return a.fetchSubscriptions(func(sub store.Subscription, show api.Show, err error) tea.Msg {
		return aboSyncMsg{sub: sub, show: show, err: err}
	})
}

func (a *App) applyAboSync(msg aboSyncMsg) tea.Cmd {
	if msg.err != nil {
		a.setError(msg.err)
		return nil
	}
	target := a.deps.Config.SubscriptionTarget(msg.sub)
	missing := media.MissingEpisodes(msg.show, target, a.deps.Config.IncludeTrailer)
	for _, ep := range missing {
		a.downloads.enqueue(msg.show.Title, ep, target)
	}
	a.abos.fresh[msg.sub.URN] = 0
	if len(missing) > 0 {
		a.tab = tabDownloads
	}
	return a.startNextDownload()
}

func (a *App) handleAbosKey(key string) tea.Cmd {
	subs := a.deps.Config.Subscriptions
	a.abos.cursor = moveCursor(a.abos.cursor, key, len(subs))
	switch key {
	case "s":
		return a.syncAllAbos()
	case "r":
		return a.countAbos()
	}
	if a.abos.cursor >= len(subs) {
		return nil
	}
	sub := subs[a.abos.cursor]
	switch key {
	case "enter":
		return a.loadShow(sub.URN)
	case "o":
		a.openPaths(sub.URN)
	case "d":
		a.deps.Config.Unsubscribe(sub.URN)
		if a.saveConfig() {
			a.setStatus("Abo entfernt: " + sub.Title)
		}
	}
	return nil
}

func (a *App) renderAbos() string {
	var b strings.Builder
	b.WriteString(a.styles.bold.Render("Abos") + "\n\n")
	subs := a.deps.Config.Subscriptions
	if len(subs) == 0 {
		return b.String() + a.styles.dim.Render("  Noch keine Abos – in der Suche mit a abonnieren.") + "\n"
	}
	b.WriteString(a.styles.dim.Render(fmt.Sprintf("  %s %s Ziel", fit("Sendung", 34), fit("Status", 9))) + "\n")
	cursor := min(a.abos.cursor, len(subs)-1)
	for i, sub := range subs {
		line := fmt.Sprintf("%s %s %s", fit(sub.Title, 34), a.freshLabel(sub.URN), a.styles.accent.Render(a.deps.Config.SubscriptionTarget(sub)))
		b.WriteString(a.row(i == cursor, line))
	}
	return b.String()
}

func (a *App) freshLabel(urn string) string {
	fresh, known := a.abos.fresh[urn]
	switch {
	case !known:
		return a.styles.dim.Render(fit("…", 9))
	case fresh == 0:
		return a.styles.ok.Render(fit("aktuell", 9))
	}
	return a.styles.warn.Render(fit(fmt.Sprintf("%d neu", fresh), 9))
}
