package main

import (
	_ "embed"
	"fmt"
	"net/http"
)

// The command reference, shown on the board. Embedded so the board works from any folder.
//
//go:embed help.html
var helpPage string

func helpView() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, helpPage)
	})
}
