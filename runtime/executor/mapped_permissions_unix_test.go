//go:build darwin || linux

package main

import (
	"os"
	"testing"
)

func privateMappedFixtureFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal("cannot set fixture permissions")
	}
}
