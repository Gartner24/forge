package checker

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPChecker struct {
	URL            string
	ExpectedStatus int
	Contains       string
	Timeout        time.Duration
}

func (c *HTTPChecker) Check() Result {
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(c.URL)
	if err != nil {
		return Result{OK: false, Reason: err.Error(), CheckedAt: time.Now()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != c.ExpectedStatus {
		return Result{
			OK:        false,
			Reason:    fmt.Sprintf("expected status %d, got %d", c.ExpectedStatus, resp.StatusCode),
			CheckedAt: time.Now(),
		}
	}

	if c.Contains != "" {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return Result{OK: false, Reason: fmt.Sprintf("reading body: %v", err), CheckedAt: time.Now()}
		}
		if !strings.Contains(string(body), c.Contains) {
			return Result{
				OK:        false,
				Reason:    fmt.Sprintf("body does not contain %q", c.Contains),
				CheckedAt: time.Now(),
			}
		}
	}

	return Result{OK: true, CheckedAt: time.Now()}
}
