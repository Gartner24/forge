package registry

import (
	"testing"
	"time"

	"github.com/gartner24/forge/sparkforge/internal/model"
)

// Kano 2 (Core): channel CRUD is the foundation of all routing.

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return r
}

func TestRegistry_AddAndList(t *testing.T) {
	r := newTestRegistry(t)

	ch := model.Channel{
		Type:        model.ChannelTypeWebhook,
		Name:        "test-wh",
		Enabled:     true,
		PriorityMin: model.PriorityLow,
		Config:      model.ChannelConfig{WebhookURL: "https://hooks.example.com"},
	}

	added, err := r.Add(ch)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if added.ID == "" {
		t.Error("Add() returned empty ID")
	}
	if added.CreatedAt.IsZero() {
		t.Error("Add() returned zero CreatedAt")
	}

	channels, err := r.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("List() returned %d channels, want 1", len(channels))
	}
	if channels[0].Name != "test-wh" {
		t.Errorf("List()[0].Name = %q, want %q", channels[0].Name, "test-wh")
	}
}

func TestRegistry_DuplicateNameRejected(t *testing.T) {
	r := newTestRegistry(t)

	ch := model.Channel{Type: model.ChannelTypeWebhook, Name: "dupe", Enabled: true,
		PriorityMin: model.PriorityLow, Config: model.ChannelConfig{WebhookURL: "https://a.example.com"}}
	if _, err := r.Add(ch); err != nil {
		t.Fatalf("first Add() error: %v", err)
	}

	ch2 := model.Channel{Type: model.ChannelTypeWebhook, Name: "dupe", Enabled: true,
		PriorityMin: model.PriorityLow, Config: model.ChannelConfig{WebhookURL: "https://b.example.com"}}
	if _, err := r.Add(ch2); err == nil {
		t.Error("second Add() with same name should return error, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := newTestRegistry(t)

	added, _ := r.Add(model.Channel{
		Type: model.ChannelTypeEmail, Name: "email-ch", Enabled: true,
		PriorityMin: model.PriorityMedium,
	})

	got, err := r.Get(added.ID)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", added.ID, err)
	}
	if got.Name != "email-ch" {
		t.Errorf("Get().Name = %q, want %q", got.Name, "email-ch")
	}

	if _, err := r.Get("nonexistent-id"); err == nil {
		t.Error("Get() with nonexistent ID should return error")
	}
}

func TestRegistry_SetEnabled(t *testing.T) {
	r := newTestRegistry(t)
	added, _ := r.Add(model.Channel{
		Type: model.ChannelTypeWebhook, Name: "toggle-ch", Enabled: true,
		PriorityMin: model.PriorityLow, Config: model.ChannelConfig{WebhookURL: "https://x.example.com"},
	})

	if err := r.SetEnabled(added.ID, false); err != nil {
		t.Fatalf("SetEnabled(false) error: %v", err)
	}
	ch, _ := r.Get(added.ID)
	if ch.Enabled {
		t.Error("channel should be disabled after SetEnabled(false)")
	}

	if err := r.SetEnabled(added.ID, true); err != nil {
		t.Fatalf("SetEnabled(true) error: %v", err)
	}
	ch, _ = r.Get(added.ID)
	if !ch.Enabled {
		t.Error("channel should be enabled after SetEnabled(true)")
	}

	if err := r.SetEnabled("nonexistent", true); err == nil {
		t.Error("SetEnabled on nonexistent ID should return error")
	}
}

func TestRegistry_Delete(t *testing.T) {
	r := newTestRegistry(t)
	added, _ := r.Add(model.Channel{
		Type: model.ChannelTypeWebhook, Name: "del-ch", Enabled: true,
		PriorityMin: model.PriorityLow, Config: model.ChannelConfig{WebhookURL: "https://del.example.com"},
	})

	if err := r.Delete(added.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	channels, _ := r.List()
	if len(channels) != 0 {
		t.Errorf("after Delete(), List() returned %d channels, want 0", len(channels))
	}

	if err := r.Delete("nonexistent"); err == nil {
		t.Error("Delete() nonexistent ID should return error")
	}
}

func TestRegistry_Update(t *testing.T) {
	r := newTestRegistry(t)
	added, _ := r.Add(model.Channel{
		Type: model.ChannelTypeWebhook, Name: "upd-ch", Enabled: true,
		PriorityMin: model.PriorityLow, Config: model.ChannelConfig{WebhookURL: "https://old.example.com"},
	})

	added.PriorityMin = model.PriorityHigh
	added.Config.WebhookURL = "https://new.example.com"
	if err := r.Update(added); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, _ := r.Get(added.ID)
	if got.PriorityMin != model.PriorityHigh {
		t.Errorf("Update() PriorityMin = %q, want %q", got.PriorityMin, model.PriorityHigh)
	}
	if got.Config.WebhookURL != "https://new.example.com" {
		t.Errorf("Update() URL = %q, want %q", got.Config.WebhookURL, "https://new.example.com")
	}
}

func TestRegistry_EmptyListReturnsSlice(t *testing.T) {
	r := newTestRegistry(t)
	channels, err := r.List()
	if err != nil {
		t.Fatalf("List() on empty registry error: %v", err)
	}
	if channels == nil {
		t.Error("List() on empty registry returned nil, want empty slice")
	}
}

func TestRegistry_IDsAreUnique(t *testing.T) {
	r := newTestRegistry(t)
	ids := make(map[string]bool)
	for i := 0; i < 20; i++ {
		added, err := r.Add(model.Channel{
			Type: model.ChannelTypeWebhook, Name: randomName(i), Enabled: true,
			PriorityMin: model.PriorityLow,
			Config:      model.ChannelConfig{WebhookURL: "https://x.example.com"},
			CreatedAt:   time.Now(),
		})
		if err != nil {
			t.Fatalf("Add() error on iteration %d: %v", i, err)
		}
		if ids[added.ID] {
			t.Errorf("duplicate ID %q on iteration %d", added.ID, i)
		}
		ids[added.ID] = true
	}
}

func randomName(i int) string {
	return "ch-" + string(rune('a'+i))
}
