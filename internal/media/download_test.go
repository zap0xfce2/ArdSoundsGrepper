package media_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ardsoundsgrepper/internal/apitest"
	"ardsoundsgrepper/internal/media"
)

const shortStall = 50 * time.Millisecond

// stallingServer liefert einen MP3-Anfang und hängt dann, bis der Client abbricht.
func stallingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(apitest.MP3Body()[:1024])
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadWritesMP3IntoNewDirectory(t *testing.T) {
	srv := apitest.NewServer(t)
	target := filepath.Join(t.TempDir(), "neu", "a.mp3")
	if err := media.NewDownloader().Download(context.Background(), srv.URL+"/cdn/a.mp3", target, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(target); !bytes.Equal(got, apitest.MP3Body()) {
		t.Error("Inhalt weicht ab")
	}
}

func TestDownloadRejectsHTML(t *testing.T) {
	srv := apitest.NewServer(t)
	err := media.NewDownloader().Download(context.Background(), srv.URL+"/cdn/html", filepath.Join(t.TempDir(), "a.mp3"), nil)
	if !errors.Is(err, media.ErrNotMP3) {
		t.Errorf("err = %v, want ErrNotMP3", err)
	}
}

func TestDownloadRejectedLeavesNoFiles(t *testing.T) {
	srv := apitest.NewServer(t)
	dir := t.TempDir()
	_ = media.NewDownloader().Download(context.Background(), srv.URL+"/cdn/html", filepath.Join(dir, "a.mp3"), nil)
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("Reste: %v", entries)
	}
}

func TestDownloadReportsProgress(t *testing.T) {
	srv := apitest.NewServer(t)
	var last int64
	_ = media.NewDownloader().Download(context.Background(), srv.URL+"/cdn/a.mp3", filepath.Join(t.TempDir(), "a.mp3"),
		func(done, _ int64) { last = done })
	if last != int64(len(apitest.MP3Body())) {
		t.Errorf("Fortschritt = %d", last)
	}
}

func TestDownloadStallFails(t *testing.T) {
	downloader := media.NewDownloader()
	downloader.StallTimeout = shortStall
	err := downloader.Download(context.Background(), stallingServer(t).URL, filepath.Join(t.TempDir(), "a.mp3"), nil)
	if !errors.Is(err, media.ErrStalled) {
		t.Errorf("err = %v, want ErrStalled", err)
	}
}

func cancelledDownload(t *testing.T) (string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(shortStall, cancel)
	target := filepath.Join(t.TempDir(), "a.mp3")
	return target, media.NewDownloader().Download(ctx, stallingServer(t).URL, target, nil)
}

func TestDownloadCancelReturnsCanceled(t *testing.T) {
	if _, err := cancelledDownload(t); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestDownloadCancelLeavesNoPart(t *testing.T) {
	target, _ := cancelledDownload(t)
	if _, err := os.Stat(target + ".part"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf(".part existiert noch: %v", err)
	}
}

func TestDownloadSkipsExistingFile(t *testing.T) {
	srv := apitest.NewServer(t)
	target := filepath.Join(t.TempDir(), "a.mp3")
	writeFile(t, target, "vorhanden")
	_ = media.NewDownloader().Download(context.Background(), srv.URL+"/cdn/a.mp3", target, nil)
	if got, _ := os.ReadFile(target); string(got) != "vorhanden" {
		t.Error("vorhandene Datei überschrieben")
	}
}
