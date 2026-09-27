package store

import (
	"os"
	"path/filepath"
)

const dirPerm = 0o755

// writeFileAtomic schreibt über Temp-Datei + Rename, damit ein Abbruch nie eine halbe Datei hinterlässt.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // nach erfolgreichem Rename ein No-op
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
