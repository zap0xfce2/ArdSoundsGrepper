package store_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"ardsoundsgrepper/internal/store"
)

func TestRememberPutsPathFirst(t *testing.T) {
	h := store.History{RecentPaths: []string{"/a", "/b"}}
	h.Remember("", "/c")
	if h.RecentPaths[0] != "/c" {
		t.Errorf("RecentPaths = %v", h.RecentPaths)
	}
}

func TestRememberDeduplicates(t *testing.T) {
	h := store.History{RecentPaths: []string{"/a", "/b"}}
	h.Remember("", "/b")
	if !reflect.DeepEqual(h.RecentPaths, []string{"/b", "/a"}) {
		t.Errorf("RecentPaths = %v", h.RecentPaths)
	}
}

func TestRememberCapsAtMax(t *testing.T) {
	var h store.History
	for i := range store.MaxRecentPaths + 3 {
		h.Remember("", fmt.Sprintf("/p%d", i))
	}
	if len(h.RecentPaths) != store.MaxRecentPaths {
		t.Errorf("Länge = %d", len(h.RecentPaths))
	}
}

func TestRememberStoresShowPath(t *testing.T) {
	var h store.History
	h.Remember("urn:x", "/x")
	if h.ShowPaths["urn:x"] != "/x" {
		t.Errorf("ShowPaths = %v", h.ShowPaths)
	}
}

func TestForgetRemovesPath(t *testing.T) {
	h := store.History{RecentPaths: []string{"/a", "/b"}}
	h.Forget("/a")
	if !reflect.DeepEqual(h.RecentPaths, []string{"/b"}) {
		t.Errorf("RecentPaths = %v", h.RecentPaths)
	}
}

func TestHistoryRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "history.json")
	want := store.History{RecentPaths: []string{"/a"}, ShowPaths: map[string]string{"urn:x": "/a"}}
	if err := store.SaveHistory(path, want); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.LoadHistory(path); !reflect.DeepEqual(got, want) {
		t.Errorf("geladen = %+v", got)
	}
}

func TestLoadHistoryMissingIsEmpty(t *testing.T) {
	if h, _ := store.LoadHistory(filepath.Join(t.TempDir(), "fehlt.json")); len(h.RecentPaths) != 0 {
		t.Errorf("History = %+v", h)
	}
}
