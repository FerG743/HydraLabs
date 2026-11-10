package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelpPageIsServedAndLinkedFromTheBoard(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/help/", http.StripPrefix("/help", helpView()))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/help/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Hydra Command Reference") {
		t.Fatalf("help page not served: %d", rec.Code)
	}
	if !strings.Contains(boardPage, `src="help/"`) {
		t.Fatal("the board does not embed the help page")
	}
}
