package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Local cases: for teams without Jira. One Markdown file per case in the app's repo, cases/<KEY>.md, written with the
// same sections as a Jira description, so it goes through the same parser (ToCase) and everything after is unchanged.
//
//	# Sign in with a valid user           <- the case name
//	labels: automate                      <- optional; automate (default) or execute
//
//	- Preconditions: The QA user exists.
//	- Steps:
//	  1. Navigate to /login.
//	  2. Enter the user and the password and click "Entrar".
//	- Expected result:
//	  - The home page shows "Orders".

const localCasesReadme = `# cases/

Test cases for Hydra, one file per case: ` + "`<KEY>.md`" + ` (e.g. ` + "`SHOP-1.md`" + `; the name is the case id, so keep it once used).
Hydra picks up new and changed files on every ` + "`hydra poll`" + `; this README is ignored.

` + "```markdown" + `
# Sign in with a valid user
labels: automate

- Preconditions: The QA user exists.
- Steps:
  1. Navigate to /login.
  2. Enter the user and the password and click "Entrar".
- Expected result:
  - The home page shows "Orders".
` + "```" + `

` + "`labels: execute`" + ` only runs the test already stored for that case; ` + "`automate`" + ` (the default) builds it.
Sections can also be in Spanish (Precondiciones, Pasos, Resultado esperado). ` + "`Preconditions: Same as SHOP-1`" + ` reuses another case's.
`

var (
	caseKey    = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[A-Z0-9-]+$`)
	listItem   = regexp.MustCompile(`^(\s*)(?:[-*+]|\d+[.)])\s+(.*)$`)
	codeSpan   = regexp.MustCompile("`([^`]*)`")
	labelsLine = regexp.MustCompile(`(?i)^(labels|etiquetas)\s*:\s*(.*)$`)
)

func (c Config) localCaseDir() string { return filepath.Join(c.Home, "cases") }

// localIssues reads cases/*.md as issues (only: just that key). A file whose name is not a case key is an error,
// said once, rather than a case that silently never runs.
func localIssues(dir, only string) ([]Issue, error) {
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	var out []Issue
	for _, f := range files {
		key := strings.ToUpper(strings.TrimSuffix(filepath.Base(f), ".md"))
		if key == "README" || (only != "" && key != only) {
			continue
		}
		if !caseKey.MatchString(key) {
			return nil, fmt.Errorf("%s: name it <KEY>.md, e.g. SHOP-1.md", f)
		}
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		is := mdIssue(string(b))
		is.Key = key
		out = append(out, is)
	}
	if only != "" && len(out) == 0 {
		return nil, fmt.Errorf("no case %s in %s", only, dir)
	}
	return out, nil
}

// mdIssue turns the Markdown into the HTML shape ToCase reads from Jira: one <ul> of labeled items, lists nested.
func mdIssue(md string) Issue {
	is := Issue{Labels: []string{"automate"}}
	var b strings.Builder
	depth := []int{} // indents of the open lists
	closeTo := func(n int) {
		for len(depth) > n {
			depth = depth[:len(depth)-1]
			b.WriteString("</li></ul>")
		}
	}
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		switch m := listItem.FindStringSubmatch(line); {
		case strings.HasPrefix(t, "# ") && is.Summary == "":
			is.Summary = strings.TrimSpace(t[2:])
		case labelsLine.MatchString(t) && len(depth) == 0:
			is.Labels = nil
			for _, l := range strings.Split(labelsLine.FindStringSubmatch(t)[2], ",") {
				if l = strings.TrimSpace(l); l != "" {
					is.Labels = append(is.Labels, l)
				}
			}
		case m != nil:
			ind := len(strings.ReplaceAll(m[1], "\t", "    "))
			for len(depth) > 0 && ind < depth[len(depth)-1] {
				closeTo(len(depth) - 1)
			}
			if len(depth) > 0 && ind == depth[len(depth)-1] {
				b.WriteString("</li>")
			} else {
				depth = append(depth, ind)
				b.WriteString("<ul>")
			}
			b.WriteString("<li>" + inline(m[2]))
		case t != "" && len(depth) > 0: // a wrapped line continues the item
			b.WriteString(" " + inline(t))
		}
	}
	closeTo(0)
	is.HTML = b.String()
	return is
}

func inline(s string) string {
	return codeSpan.ReplaceAllString(html.EscapeString(s), "<code>$1</code>")
}
