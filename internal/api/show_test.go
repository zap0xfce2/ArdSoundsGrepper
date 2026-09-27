package api_test

import (
	"context"
	"errors"
	"testing"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/apitest"
)

func fetchMimi(t *testing.T, id string) api.Show {
	t.Helper()
	srv := apitest.NewServer(t, apitest.MimiShow())
	show, err := api.NewClient(srv.Endpoint()).FetchShow(context.Background(), id)
	if err != nil {
		t.Fatalf("FetchShow: %v", err)
	}
	return show
}

func findEpisode(t *testing.T, show api.Show, id string) api.Episode {
	t.Helper()
	for _, ep := range show.Episodes {
		if ep.ID == id {
			return ep
		}
	}
	t.Fatalf("Folge %s fehlt", id)
	return api.Episode{}
}

func TestFetchShowReturnsOnlyPublishedUniqueEpisodes(t *testing.T) {
	if got := len(fetchMimi(t, apitest.MimiURN).Episodes); got != apitest.MimiPublishedCount {
		t.Errorf("Folgen = %d, want %d", got, apitest.MimiPublishedCount)
	}
}

func TestFetchShowAcceptsNumericID(t *testing.T) {
	if got := fetchMimi(t, apitest.MimiNumericID).Title; got != "Mimi Sandmädchen" {
		t.Errorf("Titel = %q", got)
	}
}

func TestFetchShowSortsNewestFirst(t *testing.T) {
	if got := fetchMimi(t, apitest.MimiURN).Episodes[0].Date(); got != "2026-09-25" {
		t.Errorf("neueste Folge = %s", got)
	}
}

func TestFetchShowOmitsFutureEpisode(t *testing.T) {
	for _, ep := range fetchMimi(t, apitest.MimiURN).Episodes {
		if ep.ID == "11271873" {
			t.Fatal("unveröffentlichte Folge enthalten")
		}
	}
}

func TestFetchShowBlockedEpisodeHasNoDownloadURL(t *testing.T) {
	if got := findEpisode(t, fetchMimi(t, apitest.MimiURN), "99999999").DownloadURL; got != "" {
		t.Errorf("DownloadURL = %q, want leer", got)
	}
}

func TestFetchShowDetectsTrailer(t *testing.T) {
	if !findEpisode(t, fetchMimi(t, apitest.MimiURN), "16251043").IsTrailer() {
		t.Error("Trailer nicht erkannt")
	}
}

func TestFetchShowUnknownShowIsErrShowNotFound(t *testing.T) {
	srv := apitest.NewServer(t, apitest.MimiShow())
	_, err := api.NewClient(srv.Endpoint()).FetchShow(context.Background(), "urn:ard:show:0000")
	if !errors.Is(err, api.ErrShowNotFound) {
		t.Errorf("err = %v, want ErrShowNotFound", err)
	}
}

func TestFetchShowUnreachableAPIFails(t *testing.T) {
	_, err := api.NewClient("http://127.0.0.1:1/graphql").FetchShow(context.Background(), apitest.MimiURN)
	if err == nil {
		t.Error("Fehler erwartet")
	}
}

func TestEpisodeDateRejectsInvalidDate(t *testing.T) {
	if got := (api.Episode{PublishDate: "../../../etc/x"}).Date(); got != "0000-00-00" {
		t.Errorf("Date = %q", got)
	}
}
