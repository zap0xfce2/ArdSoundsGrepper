// Package media kümmert sich um Dateinamen, Episoden-Status und Downloads.
package media

import (
	"regexp"
	"strings"
)

const (
	maxTitleRunes = 150
	untitledName  = "untitled"
	fileExtension = ".mp3"
)

// Muss identisch zu ardsounds-dl.sh bleiben, sonst werden vorhandene Downloads nicht erkannt.
// jq (Oniguruma) behandelt [:cntrl:] und \s als Unicode-Klassen, daher \p{Cc} und \p{Z}.
var (
	forbiddenChars = regexp.MustCompile(`[\\/:*?"<>|\p{Cc}]`)
	whitespaceRuns = regexp.MustCompile(`[\s\p{Z}]+`)
)

// SanitizeTitle macht einen Folgentitel dateisystemtauglich.
func SanitizeTitle(title string) string {
	cleaned := forbiddenChars.ReplaceAllString(title, "-")
	cleaned = whitespaceRuns.ReplaceAllString(cleaned, " ")
	cleaned = strings.Trim(cleaned, " .")
	if runes := []rune(cleaned); len(runes) > maxTitleRunes {
		cleaned = string(runes[:maxTitleRunes])
	}
	if cleaned == "" {
		return untitledName
	}
	return cleaned
}

// FileName liefert "YYYY-MM-DD - <Titel>.mp3".
func FileName(date, title string) string {
	return date + " - " + SanitizeTitle(title) + fileExtension
}
