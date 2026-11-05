package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A case's test data from the board: the values in the test repo's apps/<App>/data/<case>_data.csv, each shown next to
// the step that types or picks it (the renderer's steps.json). Saving writes that CSV and commits it, so every run
// (Run, execute, deliver, the team's CI) uses the new value; a value kept out of the repo goes to ~/.hydra.env and the
// CSV names it (ENV:NAME), the way a person would do it by hand.

type DataStep struct {
	N      string `json:"n"`
	Text   string `json:"text"`
	Column string `json:"column,omitempty"`
	Value  string `json:"value,omitempty"` // "" for an ENV: value (only its name is shown)
	Env    string `json:"env,omitempty"`   // the env var the CSV points at
	EnvSet bool   `json:"envSet"`          // it has a value on this machine
}

type CaseData struct {
	Key       string     `json:"key"`
	Delivered bool       `json:"delivered"` // the data CSV is in the test repo: values can be changed here
	File      string     `json:"file"`
	Steps     []DataStep `json:"steps"`
	Note      string     `json:"note,omitempty"`
}

func (c Config) dataRel(key string) string {
	return filepath.Join("apps", c.App, "data", token(key)+"_data.csv")
}

// readData: the CSV's header and the row of this case (nil if absent).
func readData(path, key string) (header, row []string, all [][]string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}
	all, err = csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil || len(all) == 0 {
		return nil, nil, nil, fmt.Errorf("%s: %v", path, err)
	}
	for _, r := range all[1:] {
		if len(r) > 0 && r[0] == key {
			row = r
		}
	}
	return all[0], row, all, nil
}

func (c Config) caseData(key string) CaseData {
	d := CaseData{Key: key, File: filepath.ToSlash(c.dataRel(key)), Steps: []DataStep{}}
	var steps []StepMap
	if b, err := os.ReadFile(filepath.Join(c.bundleDir(key), "steps.json")); err == nil {
		json.Unmarshal(b, &steps)
	}
	if len(steps) == 0 {
		d.Note = "this case has not been built yet (or was built before steps were recorded): Poll it first"
		return d
	}
	path := filepath.Join(c.Home, c.dataRel(key))
	d.Delivered = fileExists(path)
	if !d.Delivered { // the values are still the case's own: changing them is changing the case
		path = filepath.Join(c.bundleDir(key), c.dataRel(key))
		d.Note = "not delivered yet: the values come from the case's steps; change them there (Write a case), or Deliver first"
	}
	header, row, _, _ := readData(path, key)
	val := map[string]string{}
	for i, col := range header {
		if i > 0 && row != nil && i < len(row) {
			val[col] = row[i]
		}
	}
	for _, s := range steps {
		ds := DataStep{N: s.N, Text: s.Text, Column: s.Column}
		if v, ok := val[s.Column]; ok && s.Column != "" {
			if name, isEnv := strings.CutPrefix(v, "ENV:"); isEnv {
				ds.Env, ds.EnvSet = name, os.Getenv(name) != ""
			} else {
				ds.Value = v
			}
		}
		d.Steps = append(d.Steps, ds)
	}
	return d
}

type DataValue struct {
	Value string `json:"value"`
	Env   bool   `json:"env"` // keep it out of the repo: ~/.hydra.env, the CSV says ENV:NAME
}

var envChars = regexp.MustCompile(`[^A-Z0-9]+`)

// saveCaseData writes the values into the test repo's CSV (and ~/.hydra.env) and commits the CSV on the base branch.
func (c Config) saveCaseData(key string, vals map[string]DataValue) error {
	if !caseKey.MatchString(key) {
		return fmt.Errorf("%q is not a case key", key)
	}
	repo, rel := c.Home, c.dataRel(key)
	path := filepath.Join(repo, rel)
	if !fileExists(path) {
		return fmt.Errorf("%s has no data in the test repo yet: change the values in its steps, or Deliver it first", key)
	}
	if b, _ := git(repo, "branch", "--show-current"); b != c.BaseBranch {
		return fmt.Errorf("the test repo is on %q, not %q (a delivery running?): try again when it is back", b, c.BaseBranch)
	}
	header, row, all, err := readData(path, key)
	if err != nil {
		return err
	}
	if row == nil {
		return fmt.Errorf("%s has no row in %s", key, rel)
	}
	secrets := map[string]string{}
	for col, v := range vals {
		i := indexOf(header, col)
		if i < 1 {
			return fmt.Errorf("%s does not use a value called %q", key, col)
		}
		if strings.ContainsAny(v.Value, "\r\n") {
			return fmt.Errorf("%s: a value cannot span lines", col)
		}
		for len(row) <= i {
			row = append(row, "")
		}
		switch {
		case v.Env:
			name := "HYDRA_" + envChars.ReplaceAllString(strings.ToUpper(key+"_"+col), "_")
			if v.Value != "" { // empty: keep the value already on this machine, only (re)point the CSV at it
				secrets[name] = v.Value
			}
			row[i] = "ENV:" + name
		case strings.HasPrefix(v.Value, "ENV:"):
			row[i] = v.Value // a person naming a variable they set themselves
		default:
			row[i] = v.Value
		}
	}
	for n, r := range all {
		if n > 0 && len(r) > 0 && r[0] == key {
			all[n] = row
		}
	}
	if len(secrets) > 0 {
		if err := saveSecrets(expand(hydraEnvPath), secrets); err != nil {
			return err
		}
	}
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	w.WriteAll(all)
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return err
	}
	if _, err := git(repo, "add", rel); err != nil {
		return err
	}
	if st, _ := git(repo, "status", "--porcelain", "--", rel); st == "" {
		return nil // nothing changed in the file (only a value in ~/.hydra.env)
	}
	_, err = git(repo, append(gitIdentity(repo), "commit", "-q", "-m", "data "+key+": new values from the board", "--", rel)...)
	return err
}

func indexOf(xs []string, x string) int {
	for i, y := range xs {
		if y == x {
			return i
		}
	}
	return -1
}
