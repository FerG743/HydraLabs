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

// A case key copied into a draft's name is dropped: keys come from the save step.
func TestDraftNameDropsACopiedCaseKey(t *testing.T) {
	for _, name := range []string{"PQA-3 Enter account value", "PQA-6: Tarjetas full flow"} {
		if got := (modelCase{Name: name}).draft().Name; strings.Contains(got, "PQA-") {
			t.Fatalf("key kept in %q: %q", name, got)
		}
	}
	if got := (modelCase{Name: "Tarjetas shows Agregar"}).draft().Name; got != "Tarjetas shows Agregar" {
		t.Fatalf("plain name changed: %q", got)
	}
}

// An expected error is refused unless the requirement describes an error.
func TestInventedErrorIsRefused(t *testing.T) {
	ex := "An error toast or alert is shown to the user."
	if got := unrequested(ex, "the user clicks procesar and the account is processed."); len(got) != 1 {
		t.Fatalf("invented error accepted: %v", got)
	}
	if got := unrequested(ex, "if the account is wrong, an error message is shown."); len(got) != 0 {
		t.Fatalf("described error refused: %v", got)
	}
	if got := unrequested(`"Procesar" is enabled.`, "the user clicks procesar."); len(got) != 0 {
		t.Fatalf("plain expectation refused: %v", got)
	}
}

// The exploration's observed changes reach the prompt, and an expected state nobody observed is refused.
func TestObservedStatesReachTheCheck(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "explore"), 0o755)
	os.WriteFile(filepath.Join(dir, "explore", "Tarjetas.json"), []byte(`{"path":"/Tarjetas","candidates":[{"after_action":"select \"Selecciona la acción\" = DESBLOQUEO","control":"Procesar","field":"enabled","from":false,"to":true}]}`), 0o644)
	o := observedStates(dir, "/Tarjetas")
	if len(o) != 1 || !strings.Contains(o.prompt(), "Procesar") {
		t.Fatalf("observed: %v", o)
	}
	if got := o.unobserved(`"Procesar" is disabled before the account is added and enabled after.`); len(got) != 0 {
		t.Fatalf("observed change refused: %v", got)
	}
	if got := o.unobserved(`"Agregar" is disabled before the account is added and enabled after.`); len(got) != 1 {
		t.Fatalf("unobserved change accepted: %v", got)
	}
	if got := observedStates(dir, "/Otra").unobserved(`"Agregar" is disabled before x and enabled after.`); len(got) != 0 {
		t.Fatalf("with nothing observed the check must not refuse: %v", got)
	}
}

// An observed enable change becomes an expectation only in a draft whose steps cause it, in the renderer's form.
func TestObservedChangeBecomesAnExpectationForTheDraftThatCausesIt(t *testing.T) {
	o := observedPage{{AfterAction: `select "Selecciona la acción" = DESBLOQUEO`, Control: "Procesar", Field: "enabled", From: false, To: true}}
	got := o.expectationsFor([]string{"Navigate to `/Tarjetas`.", "Select Selecciona la acción `DESBLOQUEO`.", `Click "Agregar".`}, map[string]string{"procesar": "button"})
	if len(got) != 1 || !strings.HasPrefix(got[0], `"Procesar" is disabled before`) || !strings.HasSuffix(got[0], "enabled after.") {
		t.Fatalf("expectation: %v", got)
	}
	if n := len(o.expectationsFor([]string{"Navigate to `/Tarjetas`."}, map[string]string{"procesar": "button"})); n != 0 {
		t.Fatalf("a draft that does not take the action got an expectation: %d", n)
	}
	if !strings.Contains(got[0], "before") {
		t.Fatal("renderer needs the before/after form")
	}
}

// The page path from the requirement is lowercase; the exploration file keeps the app's own casing.
func TestObservedStatesMatchThePageIgnoringCase(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "explore"), 0o755)
	os.WriteFile(filepath.Join(dir, "explore", "Tarjetas.json"), []byte(`{"path":"/Tarjetas","candidates":[{"after_action":"click \"Agregar\"","control":"Procesar","field":"enabled","from":false,"to":true}]}`), 0o644)
	if got := observedStates(dir, "/tarjetas"); len(got) != 1 {
		t.Fatalf("lowercase page path found nothing: %v", got)
	}
}

// Only buttons get generated state expectations: the renderer checks a button's state, not a text field's.
func TestOnlyButtonsGetGeneratedStateExpectations(t *testing.T) {
	o := observedPage{{AfterAction: `click "Agregar"`, Control: "Ingresa la cuenta", Field: "enabled", From: true, To: false}}
	if n := len(o.expectationsFor([]string{`Click "Agregar".`}, map[string]string{"ingresa la cuenta": "textbox"})); n != 0 {
		t.Fatalf("a text field got a state expectation: %d", n)
	}
}

// A navigate step without its slash is the app's page; a state expectation with no between text is refused.
func TestNavigateAndEmptyBetweenAreFixedOrRefused(t *testing.T) {
	if m := navigateName.FindStringSubmatch("Navigate to Tarjetas."); m == nil || m[1] != "Tarjetas" {
		t.Fatalf("navigate not read: %v", m)
	}
	if !emptyBefore.MatchString(`"Procesar" is disabled before  and enabled after.`) {
		t.Fatal("empty between not caught")
	}
	if !emptyBefore.MatchString(`"Procesar" is disabled before after click Agregar and enabled after.`) {
		t.Fatal("empty between with 'after' not caught")
	}
	if emptyBefore.MatchString(`"Procesar" is disabled before select "X" = Y and enabled after.`) {
		t.Fatal("a real between was refused")
	}
}

// The needs list names the .env values, the dropdown options and the table title a requirement depends on.
func TestNeedsListNamesWhatTheTesterMustSupply(t *testing.T) {
	cr := &Crawl{Pages: []struct {
		URL      string
		Elements []CrawlElement
	}{{URL: "http://x/cobro_ordenes", Elements: []CrawlElement{
		{Role: "combobox", Name: "Ingresa la tienda", Options: []string{"LIVR", "SBB"}},
	}}}}
	req := "Page: /cobro_ordenes\nEscoger la orden [ORDER_NUMBER] en la tienda Ingresa la tienda y cobrarla.\nRevisar que la orden aparezca en la tabla \"Un vistazo a tus ordenes\"."
	needs := computeNeeds(req, map[string]bool{"ORDER_NUMBER": true}, cr)
	var values, options, tables int
	for _, n := range needs {
		switch n.Kind {
		case "value":
			values++
			if n.Name == "ORDER_NUMBER" && n.Status != "defined in .env" {
				t.Fatalf("defined value reported as %q", n.Status)
			}
		case "options":
			options++
			if len(n.Values) != 2 {
				t.Fatalf("options: %v", n.Values)
			}
		case "table":
			tables++
		}
	}
	if values != 1 || options != 1 || tables != 1 {
		t.Fatalf("needs: %+v", needs)
	}
	if got := computeNeeds("Escoger la orden [STORE_X].", map[string]bool{}, nil); len(got) != 1 || got[0].Status != "missing from .env" {
		t.Fatalf("missing value not reported: %+v", got)
	}
}

// The suggestion file: names from the fields, the page's first option for a dropdown, the crawl's placeholder as a
// suggestion for a text box, and nothing for a value that is already set in .env.
func TestSuggestionSuggestsWithoutOverwriting(t *testing.T) {
	cr := &Crawl{Pages: []struct {
		URL      string
		Elements []CrawlElement
	}{{URL: "http://x/cobro_ordenes", Elements: []CrawlElement{
		{Role: "textbox", Name: "Ingresa el número de orden", Placeholder: "123456"},
		{Role: "combobox", Name: "Ingresa la tienda", Options: []string{"LIVR", "SBB"}},
	}}}}
	needs := computeNeeds("Page: /cobro_ordenes\nEscoger una orden y cobrarla en la tienda Ingresa la tienda.", map[string]bool{}, cr)
	out := buildSuggestion(needs, cr, "PortalQAdev", map[string]bool{"TIENDA": true})
	for _, want := range []string{"NUMERO_DE_ORDEN=123456", "/cobro_ordenes"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "TIENDA=") {
		t.Fatal("a value already set in .env was suggested again")
	}
	if got := envName("Ingresa la cuenta"); got != "CUENTA" {
		t.Fatalf("envName: %q", got)
	}
}
