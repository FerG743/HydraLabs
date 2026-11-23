package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Suggest writes .env.suggested next to .env: the values the requirement needs, a suggestion where the crawl has one,
// and the pages the crawl knows. It never writes .env and never writes a secret: the tester fills in what is not here.
func (c Config) Suggest(requirement string) (string, error) {
	cr, err := c.loadCrawl()
	if err != nil {
		return "", err
	}
	envPath := filepath.Join(c.Home, ".env")
	content := buildSuggestion(computeNeeds(requirement, readEnvKeys(envPath), cr), cr, c.App, readEnvKeys(envPath))
	out := filepath.Join(c.Out, "env.suggested") // under out/: git ignores it, so it never blocks a crawl or a delivery
	if err := os.WriteFile(out, []byte(content), 0o600); err != nil {
		return "", err
	}
	return out, nil
}

// buildSuggestion: the file text. Keys already in .env are listed as set and never suggested again.
func buildSuggestion(needs []Need, cr *Crawl, app string, have map[string]bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Suggested values for %s. Copy the lines you need into .env, then check each one against the real test environment.\n", app)
	b.WriteString("# Never commit .env or this file, and never put a real password or token here: the tester types those in.\n\n")
	b.WriteString("# Pages the crawl knows (use these paths in the requirement):\n")
	for _, p := range crawlPaths(cr) {
		fmt.Fprintf(&b, "#   %s\n", p)
	}
	b.WriteString("\n")
	seen := map[string]bool{}
	for _, n := range needs {
		switch n.Kind {
		case "value":
			name := n.Name
			sugg := ""
			if n.Status == "not given" { // a text box the requirement gives no value for: name it after the field
				name = envName(n.Name)
				sugg = placeholderOf(cr, n.Name)
			}
			if seen[name] || have[name] {
				continue
			}
			seen[name] = true
			if n.Status == "not given" {
				fmt.Fprintf(&b, "# the requirement gives no value for %q: use [%s] in the requirement\n", n.Name, name)
			}
			fmt.Fprintf(&b, "%s=%s\n", name, sugg)
		case "options":
			name := envName(n.Name)
			if seen[name] || have[name] {
				continue
			}
			seen[name] = true
			fmt.Fprintf(&b, "# one of: %s\n%s=%s\n", strings.Join(n.Values, ", "), name, firstOf(n.Values))
		}
	}
	return b.String()
}

var nonName = regexp.MustCompile(`[^A-Z0-9]+`)
var fieldPrefix = regexp.MustCompile(`(?i)^(ingresa|escribe|selecciona|enter|type|select)\s+(el|la|los|las|the)?\s*`)

// envName: a field's name as a variable name ("Ingresa el número de orden" -> "NUMERO_DE_ORDEN").
func envName(field string) string {
	f := strings.TrimSpace(fieldPrefix.ReplaceAllString(strings.TrimSpace(field), ""))
	f = strings.NewReplacer("á", "A", "é", "E", "í", "I", "ó", "O", "ú", "U", "ñ", "N", "Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N").Replace(f)
	return strings.Trim(nonName.ReplaceAllString(strings.ToUpper(f), "_"), "_")
}

func placeholderOf(cr *Crawl, field string) string {
	if cr == nil {
		return ""
	}
	for _, p := range cr.Pages {
		for _, e := range p.Elements {
			if e.Name == field && e.Placeholder != "" {
				return e.Placeholder
			}
		}
	}
	return ""
}

func firstOf(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[0]
}

func crawlPaths(cr *Crawl) []string {
	if cr == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range cr.Pages {
		path := p.URL
		if i := strings.Index(path, "://"); i >= 0 {
			if j := strings.Index(path[i+3:], "/"); j >= 0 {
				path = path[i+3+j:]
			} else {
				path = "/"
			}
		}
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

// loadCrawl reads the project's crawl map (nil when there is none yet: the suggestion then has no pages).
func (c Config) loadCrawl() (*Crawl, error) {
	raw, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return nil, nil
	}
	cr := &Crawl{}
	if err := json.Unmarshal(raw, cr); err != nil {
		return nil, fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	return cr, nil
}
