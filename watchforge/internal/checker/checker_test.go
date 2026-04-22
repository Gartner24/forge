package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- HTTPChecker ---

func TestHTTPChecker_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello world")
	}))
	defer srv.Close()

	c := &HTTPChecker{URL: srv.URL, ExpectedStatus: 200, Timeout: 5 * time.Second}
	res := c.Check()
	if !res.OK {
		t.Errorf("expected OK, got reason: %s", res.Reason)
	}
}

func TestHTTPChecker_WrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := &HTTPChecker{URL: srv.URL, ExpectedStatus: 200, Timeout: 5 * time.Second}
	res := c.Check()
	if res.OK {
		t.Error("expected failure on wrong status")
	}
	if res.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestHTTPChecker_Contains_Match(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "status: ok, ready=true")
	}))
	defer srv.Close()

	c := &HTTPChecker{URL: srv.URL, ExpectedStatus: 200, Contains: "ready=true", Timeout: 5 * time.Second}
	res := c.Check()
	if !res.OK {
		t.Errorf("expected OK with matching contains, got: %s", res.Reason)
	}
}

func TestHTTPChecker_Contains_Miss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "not ready")
	}))
	defer srv.Close()

	c := &HTTPChecker{URL: srv.URL, ExpectedStatus: 200, Contains: "ready=true", Timeout: 5 * time.Second}
	res := c.Check()
	if res.OK {
		t.Error("expected failure when contains string missing")
	}
}

func TestHTTPChecker_Unreachable(t *testing.T) {
	c := &HTTPChecker{URL: "http://127.0.0.1:19999", ExpectedStatus: 200, Timeout: 500 * time.Millisecond}
	res := c.Check()
	if res.OK {
		t.Error("expected failure for unreachable host")
	}
}

// --- TCPChecker ---

func TestTCPChecker_OK(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	c := &TCPChecker{Address: ln.Addr().String(), Timeout: 5 * time.Second}
	res := c.Check()
	if !res.OK {
		t.Errorf("expected OK: %s", res.Reason)
	}
}

func TestTCPChecker_Unreachable(t *testing.T) {
	c := &TCPChecker{Address: "127.0.0.1:19998", Timeout: 500 * time.Millisecond}
	res := c.Check()
	if res.OK {
		t.Error("expected failure for unreachable address")
	}
}

// --- HeartbeatChecker ---

func TestHeartbeatChecker_NilPing(t *testing.T) {
	c := &HeartbeatChecker{IntervalSec: 60, GraceSec: 300}
	res := c.Check()
	if res.OK {
		t.Error("expected failure when no ping received")
	}
}

func TestHeartbeatChecker_RecentPing(t *testing.T) {
	now := time.Now()
	c := &HeartbeatChecker{IntervalSec: 60, GraceSec: 300, LastPing: &now}
	res := c.Check()
	if !res.OK {
		t.Errorf("expected OK with recent ping: %s", res.Reason)
	}
}

func TestHeartbeatChecker_ExpiredPing(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	c := &HeartbeatChecker{IntervalSec: 60, GraceSec: 300, LastPing: &old}
	res := c.Check()
	if res.OK {
		t.Error("expected failure when ping is too old")
	}
}

func TestHeartbeatChecker_WithinGrace(t *testing.T) {
	// IntervalSec=60, GraceSec=300 means deadline=360s.
	// Ping 200 seconds ago = within grace.
	recent := time.Now().Add(-200 * time.Second)
	c := &HeartbeatChecker{IntervalSec: 60, GraceSec: 300, LastPing: &recent}
	res := c.Check()
	if !res.OK {
		t.Errorf("expected OK within grace period: %s", res.Reason)
	}
}

func TestHeartbeatChecker_JustExpired(t *testing.T) {
	// Ping 361 seconds ago, deadline is 360s.
	expired := time.Now().Add(-361 * time.Second)
	c := &HeartbeatChecker{IntervalSec: 60, GraceSec: 300, LastPing: &expired}
	res := c.Check()
	if res.OK {
		t.Error("expected failure just past deadline")
	}
}
