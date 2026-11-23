package main

import (
	"encoding/json"
	"os"
	"sort"
)

// SweepMessage is one option of a swept dropdown and the fields the form shows with it ("role:name").
type SweepMessage struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

// SweepView is a page's swept dropdown: its options, each with the fields that option shows.
type SweepView struct {
	Page     string         `json:"page"`
	Control  string         `json:"control"`
	Messages []SweepMessage `json:"messages"`
}

type sweepRaw struct {
	Control  string               `json:"control"`
	Baseline []string             `json:"baseline"`
	Options  map[string]sweepDiff `json:"options"`
}

type sweepDiff struct {
	Adds    []string `json:"adds"`
	Removes []string `json:"removes"`
	Skipped string   `json:"skipped"`
	Error   string   `json:"error"`
}

// sweepsFromFile reads the sweep results the crawl stored: per page, the fields each option of the swept dropdown shows.
func sweepsFromFile(path string) []SweepView {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Pages []struct {
			URL   string    `json:"url"`
			Sweep *sweepRaw `json:"sweep"`
		} `json:"pages"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	var out []SweepView
	for _, p := range doc.Pages {
		if p.Sweep == nil {
			continue
		}
		view := SweepView{Page: pagePath(p.URL), Control: p.Sweep.Control}
		names := make([]string, 0, len(p.Sweep.Options))
		for n := range p.Sweep.Options {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			d := p.Sweep.Options[n]
			if d.Skipped != "" || d.Error != "" {
				continue
			}
			fields := map[string]bool{}
			for _, f := range p.Sweep.Baseline {
				fields[f] = true
			}
			for _, f := range d.Adds {
				fields[f] = true
			}
			for _, f := range d.Removes {
				delete(fields, f)
			}
			msg := SweepMessage{Name: n}
			for f := range fields {
				msg.Fields = append(msg.Fields, f)
			}
			sort.Strings(msg.Fields)
			view.Messages = append(view.Messages, msg)
		}
		out = append(out, view)
	}
	return out
}
