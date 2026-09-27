package api_test

import (
	"context"
	"reflect"
	"testing"

	"ardsoundsgrepper/internal/api"
	"ardsoundsgrepper/internal/apitest"
)

func search(t *testing.T, term string) ([]api.ShowHit, error) {
	t.Helper()
	srv := apitest.NewServer(t, apitest.MimiShow(), apitest.SmallShow())
	return api.NewClient(srv.Endpoint()).SearchShows(context.Background(), term)
}

func TestSearchShowsMapsHit(t *testing.T) {
	hits, _ := search(t, "mimi")
	want := []api.ShowHit{{URN: apitest.MimiURN, Title: "Mimi Sandmädchen", Sender: "rbb", EpisodeCount: 250}}
	if !reflect.DeepEqual(hits, want) {
		t.Errorf("hits = %+v, want %+v", hits, want)
	}
}

func TestSearchShowsWithoutMatchIsEmpty(t *testing.T) {
	if hits, _ := search(t, "xyzxyz"); len(hits) != 0 {
		t.Errorf("hits = %d, want 0", len(hits))
	}
}

func TestSearchShowsHandlesQuotes(t *testing.T) {
	if _, err := search(t, `"} ) \ Mimi`); err != nil {
		t.Errorf("err = %v", err)
	}
}
