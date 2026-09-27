// Package abosync lädt headless (Cron/Docker) die fehlenden Folgen der Abos.
package abosync

import (
	"context"
	"fmt"
	"io"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

const (
	ExitOK     = 0
	ExitFailed = 1 // mind. ein Download oder Abo fehlgeschlagen
	ExitUsage  = 2 // Eingabe-, Config- oder Lock-Fehler, nichts wurde versucht
)

type Options struct {
	AboFilter  string
	Client     *api.Client
	Downloader *media.Downloader
	ConfigPath string
	LockPath   string
	Version    string
	Out        io.Writer
	Err        io.Writer
}

type result struct{ loaded, present, failed int }

func (r result) add(other result) result {
	return result{r.loaded + other.loaded, r.present + other.present, r.failed + other.failed}
}

func Run(ctx context.Context, o Options) int {
	fmt.Fprintf(o.Out, "ardsoundsgrepper %s – sync\n", o.Version)
	unlock, err := store.TryLock(o.LockPath)
	if err != nil {
		fmt.Fprintf(o.Err, "Fehler: sync %v\n", err)
		return ExitUsage
	}
	defer unlock()
	cfg, subs, err := o.selectSubscriptions()
	if err != nil {
		fmt.Fprintf(o.Err, "Fehler: %v\n", err)
		return ExitUsage
	}
	var total result
	for _, sub := range subs {
		total = total.add(o.syncSubscription(ctx, cfg, sub))
	}
	fmt.Fprintf(o.Out, "fertig: %d geladen, %d vorhanden, %d Fehler\n", total.loaded, total.present, total.failed)
	if total.failed > 0 {
		return ExitFailed
	}
	return ExitOK
}

func (o Options) selectSubscriptions() (store.Config, []store.Subscription, error) {
	cfg, err := store.LoadConfig(o.ConfigPath)
	if err != nil {
		return store.Config{}, nil, err
	}
	if o.AboFilter == "" {
		return cfg, cfg.Subscriptions, nil
	}
	matches := cfg.MatchSubscriptions(o.AboFilter)
	switch len(matches) {
	case 0:
		return cfg, nil, fmt.Errorf("kein Abo passt zu %q", o.AboFilter)
	case 1:
		return cfg, matches, nil
	}
	return cfg, nil, fmt.Errorf("%q ist mehrdeutig (%d Abos)", o.AboFilter, len(matches))
}

// Ein Fehler bei einem Abo stoppt die übrigen nicht.
func (o Options) syncSubscription(ctx context.Context, cfg store.Config, sub store.Subscription) result {
	show, err := o.Client.FetchShow(ctx, sub.URN)
	if err != nil {
		fmt.Fprintf(o.Err, "FEHLER:    %s: %v\n", sub.Title, err)
		return result{failed: 1}
	}
	dir := cfg.SubscriptionTarget(sub)
	res := result{present: countPresent(show, dir)}
	for _, ep := range media.MissingEpisodes(show, dir, cfg.IncludeTrailer) {
		res = res.add(o.download(ctx, dir, ep))
	}
	fmt.Fprintf(o.Out, "%s: %d geladen, %d vorhanden, %d Fehler → %s\n", sub.Title, res.loaded, res.present, res.failed, dir)
	return res
}

func (o Options) download(ctx context.Context, dir string, ep api.Episode) result {
	name := media.FileName(ep.Date(), ep.Title)
	if err := o.Downloader.Download(ctx, ep.DownloadURL, media.EpisodePath(dir, ep), nil); err != nil {
		fmt.Fprintf(o.Err, "FEHLER:    %s: %v\n", name, err)
		return result{failed: 1}
	}
	fmt.Fprintf(o.Out, "geladen:   %s\n", name)
	return result{loaded: 1}
}

func countPresent(show api.Show, dir string) int {
	present := 0
	for _, ep := range show.Episodes {
		if media.StatusOf(dir, ep) == media.StatusPresent {
			present++
		}
	}
	return present
}
