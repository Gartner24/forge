package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/registry"
)

// Kano 2 (Core): routing is the central function. Priority filtering and per-channel
// failure isolation are correctness-critical.

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Write a minimal config so paths resolve to tmp instead of /opt/data.
	forgeDir := filepath.Join(tmp, ".forge")
	if err := os.MkdirAll(forgeDir, 0755); err != nil {
		t.Fatalf("creating forge dir: %v", err)
	}
	cfg := fmt.Sprintf("[forge]\ndata_dir = %q\ninstall_dir = %q\n", tmp+"/data", tmp)
	if err := os.WriteFile(filepath.Join(forgeDir, "config.toml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	r, err := New()
	if err != nil {
		t.Fatalf("router.New() error: %v", err)
	}
	return r
}

func addWebhookChannel(t *testing.T, reg *registry.Registry, name string, pmin model.Priority, url string) model.Channel {
	t.Helper()
	ch, err := reg.Add(model.Channel{
		Type:        model.ChannelTypeWebhook,
		Name:        name,
		Enabled:     true,
		PriorityMin: pmin,
		Config:      model.ChannelConfig{WebhookURL: url},
	})
	if err != nil {
		t.Fatalf("Add channel %q: %v", name, err)
	}
	return ch
}

func TestRouter_InvalidPriority_Error(t *testing.T) {
	r := newTestRouter(t)
	_, err := r.Send(model.Message{
		Title:    "test",
		Priority: model.Priority("bogus"),
		Source:   "test",
	})
	if err == nil {
		t.Error("Send() with invalid priority should return error")
	}
}

func TestRouter_NoChannels_DeliverNoOp(t *testing.T) {
	r := newTestRouter(t)
	delivered, err := r.Send(model.Message{
		Title:    "test",
		Priority: model.PriorityLow,
		Source:   "test",
	})
	if err != nil {
		t.Errorf("Send() with no channels returned error: %v", err)
	}
	if len(delivered) != 0 {
		t.Errorf("Send() with no channels delivered to %v, want []", delivered)
	}
}

func TestRouter_PriorityFilter_BelowMin_NotDelivered(t *testing.T) {
	// A webhook that only wants high+ — sending medium should not reach it.
	r := newTestRouter(t)

	// We need to add a channel that would actually succeed on delivery.
	// Use a test HTTP server that records calls.
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer ts.Close()

	addWebhookChannel(t, r.reg, "high-only", model.PriorityHigh, ts.URL)

	delivered, err := r.Send(model.Message{
		Title:    "deploy started",
		Priority: model.PriorityMedium,
		Source:   "smeltforge",
	})
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}
	if called {
		t.Error("webhook was called for message below priority_min")
	}
	if len(delivered) != 0 {
		t.Errorf("delivered = %v, want []", delivered)
	}
}

func TestRouter_PriorityFilter_AtMin_Delivered(t *testing.T) {
	r := newTestRouter(t)

	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer ts.Close()

	addWebhookChannel(t, r.reg, "high-channel", model.PriorityHigh, ts.URL)

	delivered, err := r.Send(model.Message{
		Title:    "deploy failed",
		Priority: model.PriorityHigh, // exactly at min
		Source:   "smeltforge",
	})
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}
	if !called {
		t.Error("webhook was not called for message at priority_min")
	}
	if len(delivered) != 1 {
		t.Errorf("delivered = %v, want [high-channel]", delivered)
	}
}

func TestRouter_PriorityFilter_AboveMin_Delivered(t *testing.T) {
	r := newTestRouter(t)

	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer ts.Close()

	addWebhookChannel(t, r.reg, "medium-channel", model.PriorityMedium, ts.URL)

	delivered, _ := r.Send(model.Message{
		Title:    "ssl expiring",
		Priority: model.PriorityCritical, // above min
		Source:   "watchforge",
	})
	if !called {
		t.Error("webhook not called when message is above priority_min")
	}
	if len(delivered) != 1 {
		t.Errorf("delivered = %v, want [medium-channel]", delivered)
	}
}

func TestRouter_DisabledChannel_Skipped(t *testing.T) {
	r := newTestRouter(t)

	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer ts.Close()

	ch := addWebhookChannel(t, r.reg, "disabled-ch", model.PriorityLow, ts.URL)
	r.reg.SetEnabled(ch.ID, false)

	delivered, _ := r.Send(model.Message{
		Title:    "test",
		Priority: model.PriorityHigh,
		Source:   "test",
	})
	if called {
		t.Error("disabled channel should not be called")
	}
	if len(delivered) != 0 {
		t.Errorf("delivered = %v, want []", delivered)
	}
}

func TestRouter_FailingChannel_DoesNotBlockOthers(t *testing.T) {
	// If channel A fails delivery, channel B should still receive the message.
	r := newTestRouter(t)

	bCalled := false
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bCalled = true
		w.WriteHeader(200)
	}))
	defer goodServer.Close()

	// Channel A: bad URL — will fail.
	addWebhookChannel(t, r.reg, "failing-ch", model.PriorityLow, "http://127.0.0.1:1") // port 1 always refused
	// Channel B: good URL — should still be reached.
	addWebhookChannel(t, r.reg, "good-ch", model.PriorityLow, goodServer.URL)

	delivered, err := r.Send(model.Message{
		Title:    "monitor down",
		Priority: model.PriorityHigh,
		Source:   "watchforge",
	})
	if err != nil {
		t.Errorf("Send() should not return error even if a channel fails: %v", err)
	}
	if !bCalled {
		t.Error("good channel was not called even though failing channel errored")
	}
	if len(delivered) != 1 || delivered[0] != "good-ch" {
		t.Errorf("delivered = %v, want [good-ch]", delivered)
	}
}

func TestRouter_Deduplication_SameEventType(t *testing.T) {
	r := newTestRouter(t)

	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		w.WriteHeader(200)
	}))
	defer ts.Close()
	addWebhookChannel(t, r.reg, "wh", model.PriorityLow, ts.URL)

	msg := model.Message{
		Title:     "api down",
		Priority:  model.PriorityHigh,
		Source:    "watchforge",
		EventType: "monitor:api-health",
	}

	// First delivery: should go through.
	r.Send(msg)
	if calls != 1 {
		t.Errorf("first Send() calls = %d, want 1", calls)
	}

	// Second delivery (same source+event_type): should be deduplicated.
	r.Send(msg)
	if calls != 1 {
		t.Errorf("after dedup, calls = %d, want still 1", calls)
	}
}

func TestRouter_Deduplication_DifferentEventType_NotDeduplicated(t *testing.T) {
	r := newTestRouter(t)

	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		w.WriteHeader(200)
	}))
	defer ts.Close()
	addWebhookChannel(t, r.reg, "wh", model.PriorityLow, ts.URL)

	r.Send(model.Message{Title: "api down", Priority: model.PriorityHigh, Source: "watchforge", EventType: "monitor:api"})
	r.Send(model.Message{Title: "db down", Priority: model.PriorityHigh, Source: "watchforge", EventType: "monitor:db"})

	if calls != 2 {
		t.Errorf("two different event_types: calls = %d, want 2", calls)
	}
}

func TestRouter_NoEventType_NeverDeduplicated(t *testing.T) {
	r := newTestRouter(t)

	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		w.WriteHeader(200)
	}))
	defer ts.Close()
	addWebhookChannel(t, r.reg, "wh", model.PriorityLow, ts.URL)

	msg := model.Message{
		Title:    "deploy done",
		Priority: model.PriorityMedium,
		Source:   "smeltforge",
		// no event_type
	}
	r.Send(msg)
	r.Send(msg)
	r.Send(msg)

	if calls != 3 {
		t.Errorf("no event_type: calls = %d, want 3 (no dedup)", calls)
	}
}

func TestRouter_MultipleChannels_AllReceive(t *testing.T) {
	r := newTestRouter(t)

	var received []string
	makeServer := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			received = append(received, name)
			w.WriteHeader(200)
		}))
	}

	ts1 := makeServer("ch1")
	defer ts1.Close()
	ts2 := makeServer("ch2")
	defer ts2.Close()
	ts3 := makeServer("ch3")
	defer ts3.Close()

	addWebhookChannel(t, r.reg, "ch1", model.PriorityLow, ts1.URL)
	addWebhookChannel(t, r.reg, "ch2", model.PriorityLow, ts2.URL)
	addWebhookChannel(t, r.reg, "ch3", model.PriorityMedium, ts3.URL)

	delivered, _ := r.Send(model.Message{
		Title:    "critical event",
		Priority: model.PriorityCritical,
		Source:   "test",
	})
	if len(delivered) != 3 {
		t.Errorf("delivered = %v, want 3 channels", delivered)
	}
	if len(received) != 3 {
		t.Errorf("received calls = %v, want 3", received)
	}
}

func TestRouter_SendToChannel_SpecificChannel(t *testing.T) {
	r := newTestRouter(t)

	ch1Called, ch2Called := false, false
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ch1Called = true
		w.WriteHeader(200)
	}))
	defer ts1.Close()
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ch2Called = true
		w.WriteHeader(200)
	}))
	defer ts2.Close()

	ch1 := addWebhookChannel(t, r.reg, "ch1", model.PriorityLow, ts1.URL)
	addWebhookChannel(t, r.reg, "ch2", model.PriorityLow, ts2.URL)

	delivered, err := r.SendToChannel(ch1.ID, model.Message{
		Title:    "targeted",
		Priority: model.PriorityHigh,
		Source:   "test",
	})
	if err != nil {
		t.Errorf("SendToChannel() error: %v", err)
	}
	if !ch1Called {
		t.Error("ch1 was not called")
	}
	if ch2Called {
		t.Error("ch2 was called — should not be when targeting ch1 only")
	}
	if len(delivered) != 1 || delivered[0] != "ch1" {
		t.Errorf("delivered = %v, want [ch1]", delivered)
	}
}

func TestRouter_SendToChannel_DisabledReturnsError(t *testing.T) {
	r := newTestRouter(t)
	ch := addWebhookChannel(t, r.reg, "dis-ch", model.PriorityLow, "http://127.0.0.1:1")
	r.reg.SetEnabled(ch.ID, false)

	_, err := r.SendToChannel(ch.ID, model.Message{
		Title:    "test",
		Priority: model.PriorityLow,
		Source:   "test",
	})
	if err == nil {
		t.Error("SendToChannel() on disabled channel should return error")
	}
}

func TestRouter_DeliveryLog_Written(t *testing.T) {
	r := newTestRouter(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()
	addWebhookChannel(t, r.reg, "logged-ch", model.PriorityLow, ts.URL)

	r.Send(model.Message{Title: "logged event", Priority: model.PriorityMedium, Source: "test"})

	records, err := r.log.Read(time.Time{})
	if err != nil {
		t.Fatalf("log.Read() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("delivery log has %d records, want 1", len(records))
	}
	if records[0].Title != "logged event" {
		t.Errorf("log record title = %q, want %q", records[0].Title, "logged event")
	}
	if records[0].Status != "ok" {
		t.Errorf("log record status = %q, want %q", records[0].Status, "ok")
	}
}

func TestRouter_DeliveryLog_FailureRecorded(t *testing.T) {
	r := newTestRouter(t)
	// Add a channel that will fail (unreachable URL).
	addWebhookChannel(t, r.reg, "fail-ch", model.PriorityLow, "http://127.0.0.1:1")

	r.Send(model.Message{Title: "failed event", Priority: model.PriorityHigh, Source: "test"})

	records, _ := r.log.Read(time.Time{})
	if len(records) != 1 {
		t.Fatalf("delivery log has %d records after failure, want 1", len(records))
	}
	if records[0].Status != "error" {
		t.Errorf("log record status = %q, want %q", records[0].Status, "error")
	}
	if records[0].Error == "" {
		t.Error("log record error field is empty for failed delivery")
	}
}
