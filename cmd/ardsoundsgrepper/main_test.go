package main

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"
)

func TestRunVersionPrintsVersion(t *testing.T) {
	var out bytes.Buffer
	run([]string{"--version"}, &out, io.Discard)
	if got := out.String(); got != version+"\n" {
		t.Errorf("Ausgabe = %q, want %q", got, version+"\n")
	}
}

func TestRunUnknownCommandIsUsageError(t *testing.T) {
	if code := run([]string{"quatsch"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("Exit-Code = %d, want %d", code, exitUsage)
	}
}

func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(dir, "state"))
}

func TestRunSyncWithoutAbosIsOK(t *testing.T) {
	isolate(t)
	if code := run([]string{"sync"}, io.Discard, io.Discard); code != exitOK {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunSyncUnknownFlagIsUsageError(t *testing.T) {
	isolate(t)
	if code := run([]string{"sync", "--quatsch"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunSyncUnknownAboIsUsageError(t *testing.T) {
	isolate(t)
	if code := run([]string{"sync", "--abo", "xyz"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("Exit = %d", code)
	}
}

func TestRunSyncPositionalArgIsUsageError(t *testing.T) {
	isolate(t)
	if code := run([]string{"sync", "mimi"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("Exit = %d", code)
	}
}
