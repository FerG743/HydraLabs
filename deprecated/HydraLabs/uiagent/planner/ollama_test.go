package planner

import "testing"

func TestParseAction(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // expected kind
	}{
		{"clean click", `{"kind":"click","x":120,"y":48,"reason":"login button"}`, "click"},
		{"with prose", "Sure! Here is the action:\n{\"kind\":\"type\",\"text\":\"alice\"}\nHope that helps.", "type"},
		{"done", `{"kind":"done","reason":"goal reached"}`, "done"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, err := ParseAction(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(a.Kind) != c.want {
				t.Fatalf("kind = %q, want %q", a.Kind, c.want)
			}
		})
	}

	if _, err := ParseAction(`not json at all`); err == nil {
		t.Fatal("expected error for non-JSON input")
	}
	if _, err := ParseAction(`{"kind":"teleport"}`); err == nil {
		t.Fatal("expected error for unknown action kind")
	}
}

func TestParseClickCoords(t *testing.T) {
	a, err := ParseAction(`{"kind":"click","x":300,"y":210}`)
	if err != nil {
		t.Fatal(err)
	}
	if a.X != 300 || a.Y != 210 {
		t.Fatalf("coords = (%d,%d), want (300,210)", a.X, a.Y)
	}
}
