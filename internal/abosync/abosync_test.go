package abosync_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ardsoundsgrepper/internal/abosync"
	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/apitest"
	"ardsoundsgrepper/internal/media"
	"ardsoundsgrepper/internal/store"
)

type fixture struct {
	opts  abosync.Options
	out   *bytes.Buffer
	music string
}

func smallSub() store.Subscription {
	return store.Subscription{URN: apitest.SmallURN, Title: "Kleine Sendung"}
}
func brokenSub() store.Subscription {
	return store.Subscription{URN: apitest.BrokenURN, Title: "Kaputte Sendung"}
}

func newFixture(t *testing.T, subs ...store.Subscription) fixture {
	t.Helper()
	srv := apitest.NewServer(t, apitest.SmallShow(), apitest.BrokenShow())
	dir := t.TempDir()
	cfg := store.DefaultConfig()
	cfg.TargetDir = filepath.Join(dir, "music")
	cfg.Subscriptions = subs
	configPath := filepath.Join(dir, "config.toml")
	if err := store.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	return fixture{out: &out, music: cfg.TargetDir, opts: abosync.Options{
		Client: api.NewClient(srv.Endpoint()), Downloader: media.NewDownloader(),
		ConfigPath: configPath, LockPath: filepath.Join(dir, "sync.lock"), Version: "vTEST", Out: &out, Err: &out,
	}}
}

func (f fixture) run() int { return abosync.Run(context.Background(), f.opts) }

func TestRunDownloadsMissingEpisodes(t *testing.T) {
	if code := newFixture(t, smallSub()).run(); code != abosync.ExitOK {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunWritesFilesToDefaultTarget(t *testing.T) {
	f := newFixture(t, smallSub())
	f.run()
	if entries, _ := os.ReadDir(filepath.Join(f.music, "Kleine Sendung")); len(entries) != 2 {
		t.Errorf("Dateien = %d, want 2", len(entries))
	}
}

func TestRunSecondRunLoadsNothing(t *testing.T) {
	f := newFixture(t, smallSub())
	f.run()
	f.out.Reset()
	f.run()
	if !strings.Contains(f.out.String(), "0 geladen, 2 vorhanden") {
		t.Errorf("Log = %s", f.out)
	}
}

func TestRunUsesSubscriptionTarget(t *testing.T) {
	sub := smallSub()
	sub.TargetDir = filepath.Join(t.TempDir(), "eigenes Ziel")
	newFixture(t, sub).run()
	if entries, _ := os.ReadDir(sub.TargetDir); len(entries) != 2 {
		t.Errorf("Dateien = %d, want 2", len(entries))
	}
}

func TestRunLogsVersionFirst(t *testing.T) {
	f := newFixture(t, smallSub())
	f.run()
	if !strings.HasPrefix(f.out.String(), "ardsoundsgrepper vTEST") {
		t.Errorf("Log = %s", f.out)
	}
}

func TestRunAboFilterSelectsOnlyMatch(t *testing.T) {
	f := newFixture(t, smallSub(), brokenSub())
	f.opts.AboFilter = "kleine"
	if code := f.run(); code != abosync.ExitOK {
		t.Errorf("Exit = %d (kaputtes Abo mitgelaufen?)", code)
	}
}

func TestRunAmbiguousFilterIsUsageError(t *testing.T) {
	f := newFixture(t, smallSub(), brokenSub())
	f.opts.AboFilter = "sendung"
	if code := f.run(); code != abosync.ExitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunUnknownFilterIsUsageError(t *testing.T) {
	f := newFixture(t, smallSub())
	f.opts.AboFilter = "xyz"
	if code := f.run(); code != abosync.ExitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunLockedIsUsageError(t *testing.T) {
	f := newFixture(t, smallSub())
	unlock, _ := store.TryLock(f.opts.LockPath)
	defer unlock()
	if code := f.run(); code != abosync.ExitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunInvalidConfigIsUsageError(t *testing.T) {
	f := newFixture(t, smallSub())
	if err := os.WriteFile(f.opts.ConfigPath, []byte("target_dir = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := f.run(); code != abosync.ExitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunDownloadFailureExitsFailed(t *testing.T) {
	if code := newFixture(t, brokenSub()).run(); code != abosync.ExitFailed {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunUnknownShowExitsFailed(t *testing.T) {
	if code := newFixture(t, store.Subscription{URN: "urn:ard:show:0000", Title: "Weg"}).run(); code != abosync.ExitFailed {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunWithoutSubscriptionsIsOK(t *testing.T) {
	if code := newFixture(t).run(); code != abosync.ExitOK {
		t.Errorf("Exit = %d", code)
	}
}
