package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Removing a project: its registration (projects.json), its profile, and optionally its Hydra-<App> repo.
// Dry run unless apply is true. The repo goes only when it has no uncommitted work.

type Removal struct {
	Project string   `json:"project"`
	Planned []string `json:"planned"` // what would be removed (dry run) or was removed
	Applied bool     `json:"applied"`
}

// Remove plans the removal and, when apply is true, carries it out. Order: repo, profile, then the registration,
// so a failure part-way leaves the project still registered and retryable.
func (r *Registry) Remove(project string, purgeRepo, apply bool) (Removal, error) {
	e, ok := r.Projects[project]
	if !ok {
		return Removal{}, fmt.Errorf("project %s is not in the registry", project)
	}
	rm := Removal{Project: project, Applied: apply}
	repo, profile := r.RepoPath(e.App), ""
	if e.Profile != "" { // an entry without a profile must never resolve to the registry's own folder
		profile = r.abs(e.Profile)
	}
	if purgeRepo && fileExists(repo) {
		if st, err := git(repo, "status", "--porcelain"); err != nil || st != "" {
			return rm, fmt.Errorf("%s has uncommitted changes or is not a git repo: commit or discard them, then purge again", repo)
		}
		rm.Planned = append(rm.Planned, "repo "+repo)
	}
	if profile != "" && fileExists(profile) {
		rm.Planned = append(rm.Planned, "profile "+profile)
	}
	rm.Planned = append(rm.Planned, "registration "+project+" in "+r.path)
	if !apply {
		return rm, nil
	}
	if purgeRepo && fileExists(repo) {
		if err := os.RemoveAll(repo); err != nil {
			return rm, err
		}
	}
	if profile != "" && fileExists(profile) {
		if err := os.Remove(profile); err != nil {
			return rm, err
		}
	}
	return rm, unregister(r.path, project)
}

// unregister edits projects.json as raw JSON so keys the Registry struct does not know (like "_doc") survive.
func unregister(path, project string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(raw["projects"], &projects); err != nil {
		return fmt.Errorf("%s: projects: %w", path, err)
	}
	delete(projects, project)
	raw["projects"], _ = json.Marshal(projects)
	out, _ := json.MarshalIndent(raw, "", "  ")
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
