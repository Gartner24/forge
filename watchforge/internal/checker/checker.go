package checker

import "time"

// Result is the outcome of a single monitor check.
type Result struct {
	OK        bool
	Reason    string
	Priority  string // alert priority override; empty = use monitor-type default
	CheckedAt time.Time
}
