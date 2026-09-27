package media_test

import (
	"strings"
	"testing"

	"ardsoundsgrepper/internal/media"
)

func TestSanitizeTitle(t *testing.T) {
	cases := []struct{ name, title, want string }{
		{"Slash Doppelpunkt Fragezeichen", "Mio/das Eichhörnchen: Der Pilz?", "Mio-das Eichhörnchen- Der Pilz-"},
		{"Komma bleibt", "Mio, das Eichhörnchen: Der Pilz", "Mio, das Eichhörnchen- Der Pilz"},
		{"Whitespace und Punkte", "  Titel   mit Leerzeichen.. ", "Titel mit Leerzeichen"},
		{"Steuerzeichen", "Tab\tim Titel", "Tab-im Titel"},
		{"alle verbotenen", `a\b/c:d*e?f"g<h>i|j`, "a-b-c-d-e-f-g-h-i-j"},
		{"leer", " . ", "untitled"},
		{"zu lang", strings.Repeat("ä", 200), strings.Repeat("ä", 150)},
		{"geschütztes Leerzeichen", "a\u00a0b", "a b"},
		{"Zeilentrenner", "p\u2028q", "p q"},
		{"C1-Steuerzeichen", "t\u0085u", "t-u"},
		{"ideografisches Leerzeichen vorne", "\u3000lead", "lead"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := media.SanitizeTitle(tc.title); got != tc.want {
				t.Errorf("SanitizeTitle(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestFileName(t *testing.T) {
	want := "2026-09-25 - Im Kuscheltierkindergarten- Rückwärts gehen.mp3"
	if got := media.FileName("2026-09-25", "Im Kuscheltierkindergarten: Rückwärts gehen"); got != want {
		t.Errorf("FileName = %q, want %q", got, want)
	}
}
