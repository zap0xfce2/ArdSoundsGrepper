package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

const (
	lockRetryInterval = 10 * time.Second
	progressBuffer    = 64
	bytesPerMB        = 1_000_000
	totalBarWidth     = 50
	itemBarWidth      = 16
)

type downloadState int

const (
	dlWaiting downloadState = iota
	dlRunning
	dlDone
	dlFailed
	dlCancelled
)

func (s downloadState) label() string {
	switch s {
	case dlWaiting:
		return "wartet"
	case dlRunning:
		return "lädt"
	case dlDone:
		return "fertig"
	case dlFailed:
		return "fehler"
	}
	return "abgebrochen"
}

type downloadItem struct {
	id      int
	show    string
	episode api.Episode
	target  string
	state   downloadState
	done    int64
	total   int64
	started time.Time
	err     error
	cancel  context.CancelFunc
}

func (d *downloadItem) path() string { return media.EpisodePath(d.target, d.episode) }

func (d *downloadItem) pending() bool { return d.state == dlWaiting || d.state == dlRunning }

func (d *downloadItem) bytesPerSecond() float64 {
	seconds := time.Since(d.started).Seconds()
	if seconds <= 0 {
		return 0
	}
	return float64(d.done) / seconds
}

type downloadsState struct {
	items    []*downloadItem
	cursor   int
	nextID   int
	running  *downloadItem
	unlock   func()
	events   <-chan tea.Msg
	quitting bool
}

type progressMsg struct {
	id          int
	done, total int64
}

type downloadDoneMsg struct {
	id  int
	err error
}

type retryLockMsg struct{}

// enqueue ignoriert Folgen, die schon in der Warteschlange stehen.
func (d *downloadsState) enqueue(show string, ep api.Episode, target string) {
	item := &downloadItem{show: show, episode: ep, target: target}
	for _, existing := range d.items {
		if existing.pending() && existing.path() == item.path() {
			return
		}
	}
	d.nextID++
	item.id = d.nextID
	d.items = append(d.items, item)
}

func (d *downloadsState) busy() bool {
	for _, item := range d.items {
		if item.pending() {
			return true
		}
	}
	return false
}

func (d *downloadsState) cancelAll() {
	for _, item := range d.items {
		if item.state == dlWaiting {
			item.state = dlCancelled
		}
	}
	if d.running != nil {
		d.running.cancel()
	}
}

func (a *App) activeDownloads() int {
	count := 0
	for _, item := range a.downloads.items {
		if item.pending() {
			count++
		}
	}
	return count
}

// Sequenziell wie im Script; der Lock wird nur während eines Downloads gehalten.
func (a *App) startNextDownload() tea.Cmd {
	if a.downloads.running != nil {
		return nil
	}
	var next *downloadItem
	for _, item := range a.downloads.items {
		if item.state == dlWaiting {
			next = item
			break
		}
	}
	if next == nil {
		return nil
	}
	unlock, err := store.TryLock(a.deps.LockPath)
	if errors.Is(err, store.ErrLocked) {
		a.setStatus("sync läuft gerade – Downloads warten")
		return tea.Tick(lockRetryInterval, func(time.Time) tea.Msg { return retryLockMsg{} })
	}
	if err != nil {
		a.setError(err)
		return nil
	}
	a.downloads.unlock = unlock
	return a.launch(next)
}

func (a *App) launch(item *downloadItem) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	item.state, item.cancel, item.started = dlRunning, cancel, time.Now()
	events := make(chan tea.Msg, progressBuffer)
	a.downloads.running, a.downloads.events = item, events
	downloader, url, path, id := a.deps.Downloader, item.episode.DownloadURL, item.path(), item.id
	go func() {
		err := downloader.Download(ctx, url, path, func(done, total int64) {
			select { // Fortschritt darf verworfen werden, das Ende nicht
			case events <- progressMsg{id: id, done: done, total: total}:
			default:
			}
		})
		events <- downloadDoneMsg{id: id, err: err}
		close(events)
	}()
	return waitForEvent(events)
}

func waitForEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

func (a *App) applyProgress(msg progressMsg) tea.Cmd {
	if item := a.downloads.running; item != nil && item.id == msg.id {
		item.done, item.total = msg.done, msg.total
	}
	return waitForEvent(a.downloads.events)
}

func (a *App) finishDownload(msg downloadDoneMsg) tea.Cmd {
	item := a.downloads.running
	a.downloads.running = nil
	if a.downloads.unlock != nil {
		a.downloads.unlock()
		a.downloads.unlock = nil
	}
	if item != nil {
		item.state, item.err = finalState(msg.err), msg.err
	}
	if a.downloads.quitting {
		return tea.Quit
	}
	return a.startNextDownload()
}

func finalState(err error) downloadState {
	switch {
	case err == nil:
		return dlDone
	case errors.Is(err, context.Canceled):
		return dlCancelled
	}
	return dlFailed
}

func (a *App) handleDownloadsKey(key string) tea.Cmd {
	a.downloads.cursor = moveCursor(a.downloads.cursor, key, len(a.downloads.items))
	if a.downloads.cursor >= len(a.downloads.items) {
		return nil
	}
	item := a.downloads.items[a.downloads.cursor]
	switch key {
	case "x":
		a.cancelItem(item)
	case "r":
		return a.retryItem(item)
	}
	return nil
}

func (a *App) cancelItem(item *downloadItem) {
	switch item.state {
	case dlRunning:
		item.cancel()
	case dlWaiting:
		item.state = dlCancelled
	}
}

func (a *App) retryItem(item *downloadItem) tea.Cmd {
	if item.state != dlFailed && item.state != dlCancelled {
		return nil
	}
	item.state, item.err, item.done = dlWaiting, nil, 0
	return a.startNextDownload()
}

func (a *App) renderDownloads() string {
	items := a.downloads.items
	if len(items) == 0 {
		return a.styles.dim.Render("Keine Downloads. Folgen in einer Sendung markieren und Enter drücken.") + "\n"
	}
	var b strings.Builder
	b.WriteString(a.renderTotal() + "\n\n")
	cursor := min(a.downloads.cursor, len(items)-1)
	start, end := window(cursor, len(items))
	for i := start; i < end; i++ {
		b.WriteString(a.row(i == cursor, a.downloadLine(items[i])))
	}
	b.WriteString("\n" + a.styles.dim.Render("Ziel:") + " " + a.styles.accent.Render(items[cursor].target) + "\n")
	return b.String()
}

func (a *App) renderTotal() string {
	var done, total int64
	for _, item := range a.downloads.items {
		if item.state != dlCancelled {
			done += item.done
			total += max(item.total, item.done) // Content-Length kann -1 sein
		}
	}
	line := fmt.Sprintf("%s  %s  %.0f/%.0f MB", a.styles.bold.Render("Gesamt"), a.bar(done, total, totalBarWidth),
		float64(done)/bytesPerMB, float64(total)/bytesPerMB)
	run := a.downloads.running
	if run == nil || run.bytesPerSecond() <= 0 {
		return line
	}
	remaining := float64(total-done) / run.bytesPerSecond()
	return line + fmt.Sprintf("  %s  %s", a.styles.accent.Render(fmt.Sprintf("%.1f MB/s", run.bytesPerSecond()/bytesPerMB)),
		a.styles.dim.Render(fmt.Sprintf("noch ~%ds", int(remaining))))
}

func (a *App) downloadLine(item *downloadItem) string {
	label := item.state.label()
	if item.state == dlFailed && item.err != nil {
		label += ": " + item.err.Error()
	}
	return fmt.Sprintf("%s %s %s %s", fit(item.show, 16), fit(item.episode.Date()+" - "+item.episode.Title, 44),
		a.bar(item.done, item.total, itemBarWidth), a.stateStyle(item.state).Render(fit(label, 40)))
}

func (a *App) stateStyle(state downloadState) lipgloss.Style {
	switch state {
	case dlDone:
		return a.styles.ok
	case dlRunning:
		return a.styles.accent
	case dlWaiting:
		return a.styles.dim
	}
	return a.styles.err
}

func (a *App) bar(done, total int64, width int) string {
	full := 0
	if total > 0 {
		full = min(width, int(float64(width)*float64(done)/float64(total)))
	}
	return a.styles.accent.Render(strings.Repeat("█", full)) + a.styles.dim.Render(strings.Repeat("░", width-full))
}
