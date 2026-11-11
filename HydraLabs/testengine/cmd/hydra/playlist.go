package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Playlists: an ordered flow of stored cases that run as one, with values passed forward (testengine/PLAYLISTS.md).
// A flow file lives in the app's repo, at hydra/playlists/<name>.json. Every case of the flow is rendered into one
// folder, so the flow shares one locators file, and tools/jira/playlist.py checks and runs it.

var playlistName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func (c Config) playlistFile(name string) (string, error) {
	if !playlistName.MatchString(name) {
		return "", fmt.Errorf("%q is not a playlist name (lowercase letters, digits and dashes)", name)
	}
	return c.durable("playlists", name+".json"), nil
}

// playlistNames lists the flows in the app's repo, for the board.
func (c Config) playlistNames() []string {
	files, _ := filepath.Glob(c.durable("playlists", "*.json"))
	var names []string
	for _, f := range files {
		if n := strings.TrimSuffix(filepath.Base(f), ".json"); playlistName.MatchString(n) {
			names = append(names, n)
		}
	}
	return names
}

// savePlaylist writes a new flow of ids, in order, to hydra/playlists/<name>.json and commits it, as the case editor
// does. It never overwrites: a flow someone wrote by hand is edited in the repo, not replaced from the board.
func (c Config) savePlaylist(name string, ids []string) error {
	path, err := c.playlistFile(name)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("add at least one case to the flow")
	}
	stored, err := c.loadCases()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, x := range stored {
		known[x.ID] = true
	}
	type step struct {
		Case string `json:"case"`
	}
	flow := struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Stop  bool   `json:"stopOnFailure"`
		Steps []step `json:"steps"`
	}{ID: name, Name: name, Stop: true}
	for _, id := range ids {
		if !known[id] {
			return fmt.Errorf("case %s is not stored in this app", id)
		}
		flow.Steps = append(flow.Steps, step{id})
	}
	repo := c.Home
	if b, _ := git(repo, "branch", "--show-current"); b != c.BaseBranch {
		return fmt.Errorf("the test repo is on %q, not %q (a delivery running?): try again when it is back", b, c.BaseBranch)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists: pick another name", name)
		}
		return err
	}
	b, _ := json.MarshalIndent(flow, "", "  ")
	_, werr := f.Write(append(b, '\n'))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	rel, _ := filepath.Rel(repo, path)
	_, err = git(repo, "add", rel)
	if err == nil {
		_, err = git(repo, append(gitIdentity(repo), "commit", "-q", "-m", "playlist "+name, "--", rel)...)
	}
	if err != nil {
		git(repo, "rm", "--cached", "-q", "--ignore-unmatch", "--", rel) // unstage it too
		os.Remove(path)                                                  // so the name is free to try again
	}
	return err
}

// buildPlaylist renders the flow's cases into out/playlists/<name>/ and writes the cases file playlist.py checks against.
func (c Config) buildPlaylist(name string) (folder, casesFile, flowFile string, err error) {
	if flowFile, err = c.playlistFile(name); err != nil {
		return "", "", "", err
	}
	b, err := os.ReadFile(flowFile)
	if err != nil {
		return "", "", "", err
	}
	var flow struct {
		Steps []struct {
			Case string `json:"case"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(b, &flow); err != nil {
		return "", "", "", fmt.Errorf("%s: %w", flowFile, err)
	}
	stored, err := c.loadCases()
	if err != nil {
		return "", "", "", err
	}
	byID := map[string]*Case{}
	for _, x := range stored {
		byID[x.ID] = x
	}
	type entry struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	var cases []entry
	var ids []string
	inFlow := map[string]bool{}
	for _, s := range flow.Steps {
		x, ok := byID[s.Case]
		if !ok {
			return "", "", "", fmt.Errorf("%s: case %s is not stored in this app", name, s.Case)
		}
		cases = append(cases, entry{x.ID, x.Name, "ready"})
		if !inFlow[x.ID] {
			inFlow[x.ID] = true
			ids = append(ids, x.ID)
		}
	}
	if len(ids) == 0 {
		return "", "", "", fmt.Errorf("%s has no steps", name)
	}
	folder = filepath.Join(c.Out, "playlists", name)
	// the defaults come from every stored case; only this flow's cases are written
	if _, _, err := c.renderOnce(c.caseDir(), folder, strings.Join(ids, ",")); err != nil {
		return "", "", "", err
	}
	casesFile = filepath.Join(folder, "cases.json")
	out, _ := json.Marshal(map[string]any{"cases": cases})
	if err := os.WriteFile(casesFile, out, 0o644); err != nil {
		return "", "", "", err
	}
	return folder, casesFile, flowFile, nil
}

// playlistCmd: hydra playlist check|run --project KEY --name FLOW [--allow-writes]
func (c Config) playlistCmd(args []string) error {
	if len(args) == 0 || (args[0] != "check" && args[0] != "run") {
		return fmt.Errorf("usage: hydra playlist check|run --project KEY --name FLOW [--allow-writes]")
	}
	fs := flag.NewFlagSet("playlist", flag.ContinueOnError)
	name := fs.String("name", "", "the flow's name: hydra/playlists/<name>.json in the app's repo")
	allow := fs.Bool("allow-writes", false, "let the flow's cases write to the app (run only)")
	fs.String("project", "", "") // the common flags are read before this; accepted here so they do not stop the parse
	fs.String("registry", "", "")
	fs.String("profile", "", "")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	folder, casesFile, flowFile, err := c.buildPlaylist(*name)
	if err != nil {
		return err
	}
	argv := []string{filepath.Join(c.ToolsDir, "playlist.py"), args[0], "--playlist", flowFile, "--cases", casesFile, "--out", folder}
	if args[0] == "run" {
		if c.PortalURL == "" {
			return fmt.Errorf("the app's URL is not set (baseUrl in the profile)")
		}
		argv = append(argv, "--base-url", c.PortalURL, "--record", filepath.Join(folder, "result.json"))
		if *allow {
			argv = append(argv, "--allow-writes")
		}
		if c.LoginFile != "" {
			argv = append(argv, "--login", c.LoginFile, "--login-path", c.LoginPath)
		}
	}
	cmd := exec.Command(c.Python, argv...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
