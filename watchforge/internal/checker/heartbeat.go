package checker

import (
	"fmt"
	"time"
)

type HeartbeatChecker struct {
	IntervalSec int
	GraceSec    int
	LastPing    *time.Time
}

func (c *HeartbeatChecker) Check() Result {
	if c.LastPing == nil {
		return Result{OK: false, Reason: "no ping received yet", CheckedAt: time.Now()}
	}
	deadline := time.Duration(c.IntervalSec+c.GraceSec) * time.Second
	if time.Since(*c.LastPing) > deadline {
		return Result{
			OK:        false,
			Reason:    fmt.Sprintf("no ping in %ds (last ping %s)", c.IntervalSec+c.GraceSec, c.LastPing.Format(time.RFC3339)),
			CheckedAt: time.Now(),
		}
	}
	return Result{OK: true, CheckedAt: time.Now()}
}
