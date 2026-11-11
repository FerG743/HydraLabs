package main

import (
	"strings"
	"testing"
)

func TestMapSaysWhichProjectWhenNoneIsGiven(t *testing.T) {
	err := Config{}.printMap(false, "", false)
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("want a hint to pass --project, got %v", err)
	}
}
