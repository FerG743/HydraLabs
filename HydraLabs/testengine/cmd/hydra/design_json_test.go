package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The model sometimes drops the final "}" of its answer; the draft must still read.
func TestAnswerMissingItsFinalBraceStillParses(t *testing.T) {
	answer := "```json\n" + `{"cases":[{"name":"a","precondition":"p","steps":[{"action":"navigate","target":"","value":"/T"}],"expected":[{"kind":"appears_in","target":"Agregar","between":""}]}]` + "\n```"
	var out struct{ Cases []modelCase }
	if err := json.Unmarshal([]byte(closeJSON(answer)), &out); err != nil || len(out.Cases) != 1 {
		t.Fatalf("err %v, cases %d, text %q", err, len(out.Cases), closeJSON(answer))
	}
}

// A visible control becomes a check the renderer knows (the "appears in" text check).
func TestControlVisibleRendersAsATextCheck(t *testing.T) {
	m := modelCase{Name: "a", Expected: []struct{ Kind, Target, Between string }{{Kind: "control_visible", Target: "Agregar"}}}
	d := m.draft()
	if len(d.Expected) != 1 || !strings.Contains(d.Expected[0], `appears in "Agregar".`) {
		t.Fatalf("got %q", d.Expected)
	}
}

// The model learns from the profile whether the app has a sign-in; it is not hard-coded in the rules.
func TestAccessNoteFollowsTheProfilesLogin(t *testing.T) {
	if got := (Config{}).accessNote(); !strings.Contains(got, "no sign-in") {
		t.Fatalf("no login: %q", got)
	}
	if got := (Config{LoginFile: "x.json", LoginPath: "/login"}).accessNote(); !strings.Contains(got, "/login") {
		t.Fatalf("login: %q", got)
	}
}

// The app's default starting state comes from the profile; a case only writes its own when it asks for one.
func TestDefaultPreconditionFollowsTheProfile(t *testing.T) {
	if got := (Config{}).defaultPrecondition(); got != "No sign-in is needed." {
		t.Fatalf("no login: %q", got)
	}
	if got := (Config{LoginFile: "x.json"}).defaultPrecondition(); got != "The user is signed in." {
		t.Fatalf("login: %q", got)
	}
}

// A field is named by the field alone: the model sometimes copies the instruction ("Ingresa la cuenta").
func TestFieldNameDropsTheInstruction(t *testing.T) {
	for in, want := range map[string]string{"Ingresa la cuenta": "cuenta", "Enter order number": "order number", "cuenta": "cuenta"} {
		if got := fieldName(in); got != want {
			t.Fatalf("%q -> %q, want %q", in, got, want)
		}
	}
}

// A draft may point at a variable in .env (ENV:QA_CARD) but never carry a secret.
func TestEnvPointerIsAllowedAndSecretsAreNot(t *testing.T) {
	if !isEnvPointer("ENV:QA_CARD") {
		t.Fatal("ENV:QA_CARD should be a pointer")
	}
	for _, v := range []string{"not-a-pointer-value", "ENV:qa card", "env:qa_card"} {
		if isEnvPointer(v) {
			t.Fatalf("%q should not pass", v)
		}
	}
}

// A value in brackets must be a variable in .env; .env values themselves are never read.
func TestBracketValuesNeedASourceInEnv(t *testing.T) {
	envKeys := map[string]bool{"SAMPLE_VALUE": true}
	if got := unsourced("Enter [SAMPLE_VALUE] in the account field.", envKeys); len(got) != 0 {
		t.Fatalf("defined name rejected: %v", got)
	}
	if got := unsourced("Enter [cuenta] in the account field.", envKeys); len(got) != 1 {
		t.Fatalf("undefined name accepted: %v", got)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	os.WriteFile(p, []byte("# comment\nBASE_URL=http://x\nSAMPLE_VALUE=placeholder\n"), 0o644)
	keys := readEnvKeys(p)
	if !keys["SAMPLE_VALUE"] || !keys["BASE_URL"] || keys["not-a-pointer-value"] || keys["# comment"] {
		t.Fatalf("keys: %v", keys)
	}
}

// Names in a step must be on the page; the crawl's names match ignoring case and accents.
func TestStepNamesMustBeOnThePage(t *testing.T) {
	names := []string{"Ingresa la cuenta", "Agregar", "Procesar", "Selecciona la acción"}
	for step, want := range map[string]int{
		"Enter cuenta `[SAMPLE_VALUE]`.":            0,
		"Select Selecciona la accion `DESBLOQUEO`.": 0,
		"Click \"Agregar\".":                        0,
		"Verify \"Procesar\" is enabled.":           0,
		"Click \"Guardar\".":                        1,
		"Enter tarjeta `[SAMPLE_VALUE]`.":           1,
	} {
		if got := unmatched(step, names); len(got) != want {
			t.Fatalf("%q: %d problems, want %d: %v", step, len(got), want, got)
		}
	}
}

// The planning call returns its text as the plan; an empty plan is an error, not a silent empty draft.
func TestPlanCallReturnsPlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "  1. Add the account.  "}}}, "usage": map[string]any{"total_tokens": 42}})
	}))
	defer srv.Close()
	c := Config{LMURL: srv.URL, Model: "m"}
	plan, tokens, err := c.askPlan("req", "no sign-in", "page", "", 2)
	if err != nil || plan != "1. Add the account." || tokens != 42 {
		t.Fatalf("plan %q tokens %d err %v", plan, tokens, err)
	}
	if planText("") != "" || !strings.Contains(planText("x"), "x") {
		t.Fatal("planText")
	}
}

// A click on a text box or dropdown is refused; a click on a button, or an enter or select, is fine.
func TestClickingAFieldIsRefused(t *testing.T) {
	roles := map[string]string{"ingresa la cuenta": "textbox", "selecciona la accion": "combobox", "agregar": "button"}
	if got := fieldClicks(`Click "Ingresa la cuenta".`, roles); len(got) != 1 {
		t.Fatalf("text box click accepted: %v", got)
	}
	if got := fieldClicks(`Click "Selecciona la acción".`, roles); len(got) != 1 {
		t.Fatalf("dropdown click accepted: %v", got)
	}
	for _, ok := range []string{`Click "Agregar".`, "Enter cuenta `[SAMPLE_VALUE]`."} {
		if got := fieldClicks(ok, roles); len(got) != 0 {
			t.Fatalf("%q refused: %v", ok, got)
		}
	}
}
