package api

import (
	"regexp"
	"strings"
)

var (
	showURNPattern   = regexp.MustCompile(`urn:ard:show:[0-9a-f]+`)
	numericIDPattern = regexp.MustCompile(`^\d+$`)
)

// ParseShowID erkennt Sendungs-URL, URN oder numerische ID; alles andere ist ein Suchbegriff.
func ParseShowID(input string) (string, bool) {
	input = strings.TrimSpace(input)
	if numericIDPattern.MatchString(input) {
		return input, true
	}
	if urn := showURNPattern.FindString(input); urn != "" {
		return urn, true
	}
	return "", false
}
