package media

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"ardsoundsgrepper/internal/api"
)

const (
	partSuffix          = ".part"
	magicPeekSize       = 3
	defaultStallTimeout = 30 * time.Second
	dirPerm             = 0o755
	mpegSyncMask        = 0xE0
)

var (
	ErrNotMP3  = errors.New("Antwort ist keine MP3")
	ErrStalled = errors.New("Download hängt, keine Daten mehr")
)

type ProgressFunc func(done, total int64)

type Downloader struct {
	HTTP         *http.Client
	StallTimeout time.Duration
}

// Kein Gesamt-Timeout am Client: große Dateien dürfen dauern, nur Stillstand bricht ab.
func NewDownloader() *Downloader {
	return &Downloader{HTTP: &http.Client{}, StallTimeout: defaultStallTimeout}
}

// Download lädt über <target>.part und benennt erst danach um, damit nie halbe MP3s entstehen.
func (d *Downloader) Download(ctx context.Context, url, target string, progress ProgressFunc) error {
	if isNonEmptyFile(target) {
		return nil // z. B. inzwischen von sync geladen – nie überschreiben
	}
	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return fmt.Errorf("Zielordner kann nicht angelegt werden: %w", err)
	}
	part := target + partSuffix
	if err := d.fetchToFile(ctx, url, part, progress); err != nil {
		_ = os.Remove(part)
		return err
	}
	return os.Rename(part, target)
}

func (d *Downloader) fetchToFile(ctx context.Context, url, path string, progress ProgressFunc) error {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var stalled atomic.Bool
	watchdog := time.AfterFunc(d.StallTimeout, func() { stalled.Store(true); cancel() })
	defer watchdog.Stop()
	err := d.stream(streamCtx, url, path, func(done, total int64) {
		watchdog.Reset(d.StallTimeout)
		if progress != nil {
			progress(done, total)
		}
	})
	switch {
	case stalled.Load():
		return ErrStalled
	case ctx.Err() != nil:
		return ctx.Err()
	}
	return err
}

func (d *Downloader) stream(ctx context.Context, url, path string, progress ProgressFunc) error {
	resp, err := d.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body := bufio.NewReader(resp.Body)
	if head, _ := body.Peek(magicPeekSize); !looksLikeMP3(head) {
		return ErrNotMP3
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, &progressReader{reader: body, total: resp.ContentLength, onRead: progress}); err != nil {
		return err
	}
	return file.Close()
}

func (d *Downloader) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", api.UserAgent)
	resp, err := d.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, fmt.Errorf("CDN antwortet mit HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

// looksLikeMP3 prüft ID3-Header oder MPEG-Frame-Sync (0xFF 0xEx).
func looksLikeMP3(head []byte) bool {
	if bytes.HasPrefix(head, []byte("ID3")) {
		return true
	}
	return len(head) >= 2 && head[0] == 0xFF && head[1]&mpegSyncMask == mpegSyncMask
}

type progressReader struct {
	reader io.Reader
	done   int64
	total  int64
	onRead ProgressFunc
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.done += int64(n)
		p.onRead(p.done, p.total)
	}
	return n, err
}
