package media_test

import (
	"os"
	"testing"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
)

var (
	loadable = api.Episode{ID: "1", Title: "Folge: Eins", PublishDate: "2026-09-25T00:01:00+02:00", DownloadURL: "http://x/1.mp3"}
	blocked  = api.Episode{ID: "2", Title: "Gesperrt", PublishDate: "2026-09-18T00:01:00+02:00"}
	trailer  = api.Episode{ID: "3", Title: "Sendung - Trailer", PublishDate: "2026-09-01T00:01:00+02:00", DownloadURL: "http://x/3.mp3"}
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStatusOfMissing(t *testing.T) {
	if got := media.StatusOf(t.TempDir(), loadable); got != media.StatusMissing {
		t.Errorf("Status = %v", got)
	}
}

func TestStatusOfPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, media.EpisodePath(dir, loadable), "ID3")
	if got := media.StatusOf(dir, loadable); got != media.StatusPresent {
		t.Errorf("Status = %v", got)
	}
}

func TestStatusOfEmptyFileIsMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, media.EpisodePath(dir, loadable), "")
	if got := media.StatusOf(dir, loadable); got != media.StatusMissing {
		t.Errorf("Status = %v", got)
	}
}

func TestStatusOfBlocked(t *testing.T) {
	if got := media.StatusOf(t.TempDir(), blocked); got != media.StatusBlocked {
		t.Errorf("Status = %v", got)
	}
}

func TestMissingEpisodesSkipsTrailerBlockedAndPresent(t *testing.T) {
	show := api.Show{Episodes: []api.Episode{loadable, blocked, trailer}}
	if got := len(media.MissingEpisodes(show, t.TempDir(), false)); got != 1 {
		t.Errorf("fehlend = %d, want 1", got)
	}
}

func TestMissingEpisodesIncludesTrailerOnRequest(t *testing.T) {
	show := api.Show{Episodes: []api.Episode{loadable, blocked, trailer}}
	if got := len(media.MissingEpisodes(show, t.TempDir(), true)); got != 2 {
		t.Errorf("fehlend = %d, want 2", got)
	}
}
