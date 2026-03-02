package server

import (
	"regexp"
	"testing"
)

func TestApplyReplacements(t *testing.T) {
	rules := []CompiledReplacement{
		{Re: regexp.MustCompile(`\s+`), With: "-"},
		{Re: regexp.MustCompile(`^x`), With: "y"},
	}
	got := applyReplacements(rules, "x  a  b")
	if got != "y-a-b" {
		t.Errorf("applyReplacements() = %q, want %q", got, "y-a-b")
	}
}

func TestApplyReplacements_Empty(t *testing.T) {
	got := applyReplacements(nil, "hello")
	if got != "hello" {
		t.Errorf("applyReplacements(nil, ...) = %q, want hello", got)
	}
}
