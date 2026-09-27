package api_test

import (
	"testing"

	"ardsoundsgrepper/internal/api"
)

func TestParseShowID(t *testing.T) {
	const urn = "urn:ard:show:daebe2a39366d28a"
	cases := []struct {
		name, input, want string
		ok                bool
	}{
		{"URL", "https://www.ardsounds.de/sendung/mimi-sandmaedchen/" + urn + "/", urn, true},
		{"URN", urn, urn, true},
		{"numerisch", "16240993", "16240993", true},
		{"Episode", "https://www.ardsounds.de/episode/urn:ard:episode:cc3900daf3771a8c/", "", false},
		{"Suchbegriff", "Mimi", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := api.ParseShowID(tc.input)
			if got != tc.want || ok != tc.ok {
				t.Errorf("ParseShowID(%q) = %q, %v; want %q, %v", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}
