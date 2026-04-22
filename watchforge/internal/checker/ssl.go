package checker

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

const (
	sslWarnDays     = 14
	sslCriticalDays = 3
)

// SSL priority hints used by the scheduler.
const (
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityCritical = "critical"
)

type SSLChecker struct {
	Host string
}

func (c *SSLChecker) Check() Result {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		c.Host+":443",
		&tls.Config{ServerName: c.Host},
	)
	if err != nil {
		return Result{OK: false, Reason: err.Error(), Priority: PriorityHigh, CheckedAt: time.Now()}
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return Result{OK: false, Reason: "no certificates found", Priority: PriorityHigh, CheckedAt: time.Now()}
	}

	expiry := certs[0].NotAfter
	days := int(time.Until(expiry).Hours() / 24)

	if days <= 0 {
		return Result{OK: false, Reason: "certificate has expired", Priority: PriorityCritical, CheckedAt: time.Now()}
	}
	if days <= sslCriticalDays {
		return Result{
			OK:       false,
			Reason:   fmt.Sprintf("certificate expires in %d days", days),
			Priority: PriorityCritical, CheckedAt: time.Now(),
		}
	}
	if days <= sslWarnDays {
		return Result{
			OK:       false,
			Reason:   fmt.Sprintf("certificate expires in %d days", days),
			Priority: PriorityMedium, CheckedAt: time.Now(),
		}
	}

	return Result{OK: true, CheckedAt: time.Now()}
}
