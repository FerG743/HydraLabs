package agent

// Rect is a pixel bounding box; Center is the natural click point.
type Rect struct{ X, Y, W, H int }

func (r Rect) Center() (int, int) { return r.X + r.W/2, r.Y + r.H/2 }

// Element is one interactive thing on screen. With a DOM or accessibility tree
// it arrives with a Ref; with vision + set-of-marks the Box is filled in and the
// Ref is the mark number. The brain targets a Ref OR a coordinate — never a CSS
// selector — which is what keeps it driver-agnostic.
type Element struct {
	Ref   string
	Role  string
	Name  string
	Value string
	Box   Rect // pixel bounds (zero value when unknown)
}

// Observation is everything perceived in one step. For the vision path it
// carries the raw Screenshot (what the VLM sees) plus dimensions; Elements is
// optional, populated only when a detector / accessibility / set-of-marks pass
// has run.
type Observation struct {
	URL        string
	Title      string
	Width      int
	Height     int
	Screenshot []byte    // PNG bytes (nil for purely structured surfaces)
	Elements   []Element // optional detected elements
}

type ActionKind string

const (
	ActClick  ActionKind = "click"  // click Ref's element, or pixel X,Y when Ref == ""
	ActType   ActionKind = "type"   // type Text
	ActKey    ActionKind = "key"    // press Keys, e.g. "enter" or "cmd+s"
	ActScroll ActionKind = "scroll"
	ActDone   ActionKind = "done"
)

type Action struct {
	Kind   ActionKind
	Ref    string // element handle (set-of-marks / accessibility)
	X, Y   int    // pixel target when Ref is empty
	Text   string // text to type
	Keys   string // key combo for ActKey
	Reason string
}

type Goal struct {
	Task string
}

type Step struct {
	N      int
	Action Action
	Obs    Observation
}

type Transcript struct {
	Steps []Step
	Done  bool
}
