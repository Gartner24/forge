package model

import (
	"testing"
)

// Kano 2 (Core): priority is the central routing key — must be 100% correct.

func TestPriority_Level(t *testing.T) {
	cases := []struct {
		p     Priority
		level int
	}{
		{PriorityLow, 1},
		{PriorityMedium, 2},
		{PriorityHigh, 3},
		{PriorityCritical, 4},
		{Priority("unknown"), 0},
		{Priority(""), 0},
	}
	for _, c := range cases {
		if got := c.p.Level(); got != c.level {
			t.Errorf("Priority(%q).Level() = %d, want %d", c.p, got, c.level)
		}
	}
}

func TestPriority_Valid(t *testing.T) {
	valid := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}
	for _, p := range valid {
		if !p.Valid() {
			t.Errorf("Priority(%q).Valid() = false, want true", p)
		}
	}
	invalid := []Priority{Priority(""), Priority("CRITICAL"), Priority("Low"), Priority("urgent")}
	for _, p := range invalid {
		if p.Valid() {
			t.Errorf("Priority(%q).Valid() = true, want false", p)
		}
	}
}

func TestParsePriority(t *testing.T) {
	ok := []struct {
		input string
		want  Priority
	}{
		{"low", PriorityLow},
		{"medium", PriorityMedium},
		{"high", PriorityHigh},
		{"critical", PriorityCritical},
	}
	for _, c := range ok {
		got, err := ParsePriority(c.input)
		if err != nil {
			t.Errorf("ParsePriority(%q) unexpected error: %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("ParsePriority(%q) = %q, want %q", c.input, got, c.want)
		}
	}

	bad := []string{"", "LOW", "HIGH", "Medium", "urgent", "0", "4"}
	for _, s := range bad {
		if _, err := ParsePriority(s); err == nil {
			t.Errorf("ParsePriority(%q) expected error, got nil", s)
		}
	}
}

func TestPriority_Ordering(t *testing.T) {
	if PriorityLow.Level() >= PriorityMedium.Level() {
		t.Error("low must be less than medium")
	}
	if PriorityMedium.Level() >= PriorityHigh.Level() {
		t.Error("medium must be less than high")
	}
	if PriorityHigh.Level() >= PriorityCritical.Level() {
		t.Error("high must be less than critical")
	}
}

func TestChannelType_Valid(t *testing.T) {
	valid := []ChannelType{ChannelTypeGotify, ChannelTypeEmail, ChannelTypeWebhook}
	for _, ct := range valid {
		if !ct.Valid() {
			t.Errorf("ChannelType(%q).Valid() = false, want true", ct)
		}
	}
	invalid := []ChannelType{ChannelType(""), ChannelType("slack"), ChannelType("push")}
	for _, ct := range invalid {
		if ct.Valid() {
			t.Errorf("ChannelType(%q).Valid() = true, want false", ct)
		}
	}
}
