package engine

// Status is the outcome a block reports to the interpreter. The interpreter
// uses it to pick which outgoing edge to follow next, so adding new control
// flow later (retries, conditionals, switches) means teaching the interpreter
// one more status value, not rewriting the walker.
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"

	// StatusBlocked is emitted by the interpreter itself (never by a block)
	// when a block's precondition rules fail. The block does not run.
	StatusBlocked Status = "blocked"
)
