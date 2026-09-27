package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"ardsoundsgrepper/internal/store"
)

func TestTryLockSecondAttemptIsLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.lock")
	unlock, err := store.TryLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := store.TryLock(path); !errors.Is(err, store.ErrLocked) {
		t.Errorf("err = %v, want ErrLocked", err)
	}
}

func TestTryLockAfterUnlockSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.lock")
	unlock, _ := store.TryLock(path)
	unlock()
	if _, err := store.TryLock(path); err != nil {
		t.Errorf("err = %v", err)
	}
}
