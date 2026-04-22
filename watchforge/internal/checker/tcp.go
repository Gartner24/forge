package checker

import (
	"net"
	"time"
)

type TCPChecker struct {
	Address string
	Timeout time.Duration
}

func (c *TCPChecker) Check() Result {
	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return Result{OK: false, Reason: err.Error(), CheckedAt: time.Now()}
	}
	conn.Close()
	return Result{OK: true, CheckedAt: time.Now()}
}
