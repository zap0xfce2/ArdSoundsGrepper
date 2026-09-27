package media

import (
	"os"
	"path/filepath"

	"ardsoundsgrepper/internal/api"
)

type Status int

const (
	StatusMissing Status = iota
	StatusPresent
	StatusBlocked
)

func EpisodePath(dir string, ep api.Episode) string {
	return filepath.Join(dir, FileName(ep.Date(), ep.Title))
}

// StatusOf wird immer frisch aus Dateisystem und API berechnet, nie gespeichert.
func StatusOf(dir string, ep api.Episode) Status {
	if isNonEmptyFile(EpisodePath(dir, ep)) {
		return StatusPresent
	}
	if ep.DownloadURL == "" {
		return StatusBlocked
	}
	return StatusMissing
}

func isNonEmptyFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

// MissingEpisodes ist die gemeinsame Regel für sync, Abo-Zähler und TUI.
func MissingEpisodes(show api.Show, dir string, includeTrailer bool) []api.Episode {
	var missing []api.Episode
	for _, ep := range show.Episodes {
		if ep.IsTrailer() && !includeTrailer {
			continue
		}
		if StatusOf(dir, ep) == StatusMissing {
			missing = append(missing, ep)
		}
	}
	return missing
}
