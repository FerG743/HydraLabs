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

// Input is one field or dropdown on the crawled pages: the name the case uses, what it offers, and a suggestion.
type Input struct {
	Name        string   `json:"name"`
	Field       string   `json:"field"` // "textbox" | "combobox"
	EnvName     string   `json:"envName"`
	Page        string   `json:"page"`
	Group       string   `json:"group"` // the tab the input sits under, or "" for the page itself
	Labelled    bool     `json:"labelled"`
	Options     []string `json:"options,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
	InEnv       bool     `json:"inEnv"`
	HandEntered bool     `json:"handEntered,omitempty"` // the options were typed in by a person, not crawled
	Incomplete  string   `json:"incomplete,omitempty"`  // why a person needs to look at this input; "" when complete
}

// incompleteReason: a dropdown with no options, or a text box the crawl took its value for as its name.
func incompleteReason(in Input) string {
	if in.Field == "combobox" && len(in.Options) == 0 {
		return "no options: the crawl could not open it; type them in the options box"
	}
	if in.Field == "textbox" && !in.Labelled {
		return "no label: the crawl took its value as its name; name it in the requirement or skip it"
	}
	return ""
}

// labelLike: a real label has a word in it. "01", "1033030823" and "303030…" are values the crawl took as names.
var wordRun = regexp.MustCompile(`\p{L}{3,}`)

func labelLike(name string) bool {
	return wordRun.MatchString(name)
}

// inputsOf: every named text box and dropdown in the crawl, one entry each, with the .env name it would be saved under.
func inputsOf(cr *Crawl, have map[string]bool) []Input {
	return inputsOfHand(cr, have, nil)
}

// inputsOfHand is inputsOf with the options a person typed in for dropdowns the crawl could not open.
func inputsOfHand(cr *Crawl, have map[string]bool, hand map[string][]string) []Input {
	seen := map[string]bool{}
	var out []Input
	for _, p := range cr.Pages {
		page := pagePath(p.URL)
		for _, e := range p.Elements {
			key := page + "|" + e.Under + "|" + e.Name
			if e.Name == "" || (e.Role != "textbox" && e.Role != "combobox") || seen[key] {
				continue
			}
			seen[key] = true
			in := Input{Name: e.Name, Field: e.Role, EnvName: envName(e.Name), Options: e.Options, Page: page, Group: e.Under, Labelled: labelLike(e.Name)}
			if e.Role == "combobox" && len(in.Options) == 0 && len(hand[e.Name]) > 0 {
				in.Options, in.HandEntered = hand[e.Name], true
			}
			if e.Role == "combobox" && len(e.Options) > 0 {
				in.Suggestion = e.Options[0]
			} else {
				in.Suggestion = e.Placeholder
			}
			in.InEnv = have[in.EnvName]
			in.Incomplete = incompleteReason(in)
			out = append(out, in)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

var envNameOK = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// saveEnvValue appends NAME=value to the project's .env. It refuses a name that is already set (a value is never overwritten),
// a name that is not a plain variable name, and a value that would break the file's line format.
func (c Config) saveEnvValue(name, value string) error {
	if !envNameOK.MatchString(name) {
		return fmt.Errorf("%q is not a variable name: use capitals, digits and underscores, like QA_CARD", name)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("a value has to be one line")
	}
	path := filepath.Join(c.Home, ".env")
	if readEnvKeys(path)[name] {
		return fmt.Errorf("%s is already set in .env: edit it by hand if you want another value", name)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
	return err
}

// pagePath: the path of a crawled page's URL ("http://host/Tarjetas" -> "/Tarjetas").
func pagePath(rawURL string) string {
	if i := strings.Index(rawURL, "://"); i >= 0 {
		if j := strings.Index(rawURL[i+3:], "/"); j >= 0 {
			return rawURL[i+3+j:]
		}
		return "/"
	}
	return rawURL
}

// handOptionsPath: the options a person typed in for dropdowns the crawl could not open. Under out/: it is never committed.
func (c Config) handOptionsPath() string { return filepath.Join(c.Out, "hand-options.json") }

func loadHandOptions(path string) map[string][]string {
	out := map[string][]string{}
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &out)
	}
	return out
}

// saveHandOption stores one dropdown's options (one per line in the form). Blank lines are dropped, and a line break is
// the only separator, so an option may contain commas.
func (c Config) saveHandOption(name string, options []string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("the dropdown's name is missing")
	}
	var clean []string
	for _, o := range options {
		if o = strings.TrimSpace(o); o != "" {
			clean = append(clean, o)
		}
	}
	if len(clean) == 0 {
		return fmt.Errorf("give at least one option, one per line")
	}
	if len(clean) > 200 {
		return fmt.Errorf("more than 200 options: check the list")
	}
	all := loadHandOptions(c.handOptionsPath())
	all[name] = clean
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(c.handOptionsPath()), 0o755)
	return os.WriteFile(c.handOptionsPath(), b, 0o600)
}
