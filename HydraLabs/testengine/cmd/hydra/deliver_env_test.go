package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The test process gets the project's .env values it does not already have, and never overrides one it has.
func TestDotenvOnlyAddsWhatTheProcessDoesNotHave(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("# note\nSAMPLE_VALUE=placeholder\nBASE_URL=http://wrong\n"), 0o644)
	got := dotenvOnly(p, []string{"BASE_URL=http://127.0.0.1:3000"})
	if len(got) != 1 || got[0] != "SAMPLE_VALUE=placeholder" {
		t.Fatalf("got %v", got)
	}
}
