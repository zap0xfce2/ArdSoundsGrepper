package store

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

const lockFilePerm = 0o644

var ErrLocked = errors.New("läuft bereits")

// TryLock verhindert, dass TUI und Cron-sync gleichzeitig dieselben .part-Dateien schreiben.
func TryLock(path string) (unlock func(), err error) {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockFilePerm)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}, nil
}
