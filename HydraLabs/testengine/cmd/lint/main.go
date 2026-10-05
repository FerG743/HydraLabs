// lint checks a generated .feature (stdin) and prints {"ok":bool,"errors":[],"needs_input":[]}.
// It always exits 0 so an orchestrator can branch on "ok" and feed "errors"
// back to the LM. Checks are structural, not project-specific.
//
//	-min-then N       require at least N explicit "Entonces" steps (one per expected result)
//	-max-scenarios N  allow at most N scenarios/outlines (0 = unlimited)
//
//	go run ./cmd/lint -min-then 19 -max-scenarios 1 < case.feature
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	gherkin "github.com/cucumber/gherkin/go/v26"
	"github.com/cucumber/messages/go/v21"
)

type Result struct {
	OK         bool     `json:"ok"`
	Errors     []string `json:"errors"`
	NeedsInput []string `json:"needs_input"`
}

var (
	placeholder = regexp.MustCompile(`<([^<>\s][^<>]*)>`)
	// a credential literal after "contraseña:"/"password=" that is not a <placeholder>
	secret = regexp.MustCompile(`(?i)(contrase[ñn]a|password)\s*[:=]\s*[^<\s]`)
)

type Opts struct {
	MinThen, MaxScenarios int
	Lang                  string // expected "# language:" value; empty = "es"
}

func lint(src string, o Opts) Result {
	r := Result{Errors: []string{}, NeedsInput: []string{}}
	id := 0
	doc, err := gherkin.ParseGherkinDocument(strings.NewReader(src), func() string { id++; return fmt.Sprint(id) })
	if err != nil {
		r.Errors = append(r.Errors, "parse error: "+err.Error())
		return r
	}
	for _, c := range doc.Comments {
		if i := strings.Index(c.Text, "NEEDS-INPUT"); i >= 0 {
			r.NeedsInput = append(r.NeedsInput, strings.TrimSpace(c.Text[i:]))
		}
	}
	if doc.Feature == nil {
		r.Errors = append(r.Errors, "no Feature found (first line must be '# language: es', then 'Característica:')")
		return r
	}
	lang := o.Lang
	if lang == "" {
		lang = "es"
	}
	if doc.Feature.Language != lang {
		r.Errors = append(r.Errors, fmt.Sprintf("language is %q, expected '# language: %s' on the first line", doc.Feature.Language, lang))
	}

	scenarios := 0
	for _, ch := range doc.Feature.Children {
		sc := ch.Scenario
		if sc == nil {
			continue
		}
		scenarios++
		outcomes, params := 0, map[string]bool{}
		for _, st := range sc.Steps {
			if st.KeywordType == messages.StepKeywordType_OUTCOME {
				outcomes++
			}
			if secret.MatchString(st.Text) {
				r.Errors = append(r.Errors, fmt.Sprintf("%q: step contains a literal credential: use <usuario>/<contraseña>", sc.Name))
			}
			text := st.Text
			if st.DocString != nil {
				text += "\n" + st.DocString.Content
			}
			if st.DataTable != nil {
				for _, row := range st.DataTable.Rows {
					for _, c := range row.Cells {
						text += "\n" + c.Value
					}
				}
			}
			for _, m := range placeholder.FindAllStringSubmatch(text, -1) {
				params[m[1]] = true
			}
		}
		if len(sc.Steps) == 0 {
			r.Errors = append(r.Errors, fmt.Sprintf("%q: scenario has no steps", sc.Name))
		} else if outcomes == 0 {
			r.Errors = append(r.Errors, fmt.Sprintf("%q: scenario has no 'Entonces' step", sc.Name))
		} else if outcomes < o.MinThen {
			r.Errors = append(r.Errors, fmt.Sprintf("%q: has %d 'Entonces' steps but the case has %d expected results: write one 'Entonces' per expected result, right after its step", sc.Name, outcomes, o.MinThen))
		}
		if len(params) == 0 && len(sc.Examples) == 0 {
			continue
		}
		// every <param> used in steps must be a column of an Examples table
		cols := map[string]bool{}
		for _, ex := range sc.Examples {
			if ex.TableHeader == nil || len(ex.TableBody) == 0 {
				r.Errors = append(r.Errors, fmt.Sprintf("%q: Examples table needs a header and at least one row", sc.Name))
				continue
			}
			for _, c := range ex.TableHeader.Cells {
				cols[c.Value] = true
				if !params[c.Value] {
					r.Errors = append(r.Errors, fmt.Sprintf("%q: Ejemplos column %q is never used as <%s> in a step: use it or drop the column", sc.Name, c.Value, c.Value))
				}
			}
		}
		for p := range params {
			// credentials come from a secrets store at run time, not from Examples
			if !cols[p] && p != "usuario" && p != "contraseña" && p != "contrasena" {
				r.Errors = append(r.Errors, fmt.Sprintf("%q: placeholder <%s> has no column in Ejemplos", sc.Name, p))
			}
		}
	}
	if scenarios == 0 {
		r.Errors = append(r.Errors, "no Scenario found")
	}
	if o.MaxScenarios > 0 && scenarios > o.MaxScenarios {
		r.Errors = append(r.Errors, fmt.Sprintf("feature has %d scenarios, expected at most %d: merge them into one (use Esquema del escenario for data rows)", scenarios, o.MaxScenarios))
	}
	r.OK = len(r.Errors) == 0
	return r
}

func main() {
	var o Opts
	flag.StringVar(&o.Lang, "lang", "es", "expected Gherkin language code")
	flag.IntVar(&o.MinThen, "min-then", 0, "minimum explicit Entonces steps")
	flag.IntVar(&o.MaxScenarios, "max-scenarios", 0, "maximum scenarios (0 = unlimited)")
	flag.Parse()
	b, _ := io.ReadAll(os.Stdin)
	json.NewEncoder(os.Stdout).Encode(lint(string(b), o))
}
