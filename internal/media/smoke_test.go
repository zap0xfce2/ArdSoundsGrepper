//go:build smoke

package media_test

import (
	"context"
	"testing"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/media"
)

func TestSmokeRealAPIDownloadsNewestEpisode(t *testing.T) {
	ctx := context.Background()
	show, err := api.NewClient(api.DefaultEndpoint()).FetchShow(ctx, "urn:ard:show:daebe2a39366d28a")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	newest := media.MissingEpisodes(show, dir, false)[0]
	if err := media.NewDownloader().Download(ctx, newest.DownloadURL, media.EpisodePath(dir, newest), nil); err != nil {
		t.Errorf("Download: %v", err)
	}
}
