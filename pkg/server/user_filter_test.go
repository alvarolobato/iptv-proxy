package server

import (
	"testing"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

func TestNewUserAccessFilter_NilUser(t *testing.T) {
	f := NewUserAccessFilter(nil)
	if f != nil {
		t.Error("expected nil filter for nil user")
	}
}

func TestNewUserAccessFilter_NoRules(t *testing.T) {
	u := &config.User{Username: "test", Enabled: true}
	f := NewUserAccessFilter(u)
	if f != nil {
		t.Error("expected nil filter for user with no access rules")
	}
}

func TestNewUserAccessFilter_HasRules(t *testing.T) {
	u := &config.User{
		Username:       "test",
		GroupAllowList: []string{"^Sports$"},
	}
	f := NewUserAccessFilter(u)
	if f == nil {
		t.Fatal("expected non-nil filter")
	}
	if !f.HasRules() {
		t.Error("expected HasRules() = true")
	}
}

func TestUserAccessFilter_MatchTrack_NilFilter(t *testing.T) {
	var f *UserAccessFilter
	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("nil filter should allow everything")
	}
}

func TestUserAccessFilter_GroupAllowOnly(t *testing.T) {
	u := &config.User{GroupAllowList: []string{"^Sports$", "^News$"}}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("Sports group should be allowed")
	}
	if !f.MatchTrack("News", "CNN") {
		t.Error("News group should be allowed")
	}
	if f.MatchTrack("Entertainment", "Comedy Central") {
		t.Error("Entertainment group should be blocked by allow list")
	}
}

func TestUserAccessFilter_GroupBlockOnly(t *testing.T) {
	u := &config.User{GroupBlockList: []string{"^Adult$"}}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("Sports should be allowed")
	}
	if f.MatchTrack("Adult", "Adult Channel") {
		t.Error("Adult group should be blocked")
	}
}

func TestUserAccessFilter_ChannelAllowOnly(t *testing.T) {
	u := &config.User{ChannelAllowList: []string{"^BBC.*", "^CNN.*"}}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("News", "BBC News") {
		t.Error("BBC News should be allowed")
	}
	if !f.MatchTrack("News", "CNN International") {
		t.Error("CNN should be allowed")
	}
	if f.MatchTrack("News", "Fox News") {
		t.Error("Fox News should be blocked by channel allow list")
	}
}

func TestUserAccessFilter_ChannelBlockOnly(t *testing.T) {
	u := &config.User{ChannelBlockList: []string{"^Premium.*"}}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("ESPN should be allowed")
	}
	if f.MatchTrack("Sports", "Premium Sports") {
		t.Error("Premium Sports should be blocked")
	}
}

func TestUserAccessFilter_CombinedGroupAndChannel(t *testing.T) {
	u := &config.User{
		GroupAllowList:   []string{"^Sports$"},
		ChannelBlockList: []string{"^Premium.*"},
	}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("ESPN in Sports should be allowed")
	}
	if f.MatchTrack("Sports", "Premium Sports") {
		t.Error("Premium Sports should be blocked even in allowed group")
	}
	if f.MatchTrack("News", "CNN") {
		t.Error("News group not in allow list, should be blocked")
	}
}

func TestUserAccessFilter_AllowThenBlock(t *testing.T) {
	u := &config.User{
		GroupAllowList: []string{"^Sports$", "^News$"},
		GroupBlockList: []string{"^News$"},
	}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("Sports should be allowed")
	}
	// News passes allow but is then blocked
	if f.MatchTrack("News", "CNN") {
		t.Error("News should be blocked (block overrides allow)")
	}
}

func TestUserAccessFilter_RegexPatterns(t *testing.T) {
	u := &config.User{
		GroupAllowList: []string{"(?i)sport"},
	}
	f := NewUserAccessFilter(u)

	if !f.MatchTrack("Sports", "ESPN") {
		t.Error("case-insensitive regex should match Sports")
	}
	if !f.MatchTrack("SPORTS HD", "ESPN") {
		t.Error("case-insensitive regex should match SPORTS HD")
	}
}

func TestValidatePatterns_AllValid(t *testing.T) {
	errs := ValidatePatterns([]string{"^Sports$"}, []string{"^Adult$"}, nil, nil)
	if errs != nil {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidatePatterns_InvalidRegex(t *testing.T) {
	errs := ValidatePatterns([]string{"^Sports$", "[invalid"}, nil, nil, nil)
	if errs == nil {
		t.Fatal("expected validation errors")
	}
	if len(errs["group_allow_list"]) != 1 {
		t.Errorf("expected 1 error for group_allow_list, got %v", errs)
	}
}

func TestValidatePatterns_EmptyPatternsIgnored(t *testing.T) {
	errs := ValidatePatterns([]string{""}, []string{""}, nil, nil)
	if errs != nil {
		t.Errorf("expected no errors for empty patterns, got %v", errs)
	}
}

func TestValidatePatterns_MultipleFieldErrors(t *testing.T) {
	errs := ValidatePatterns([]string{"[bad1"}, []string{"[bad2"}, []string{"[bad3"}, []string{"[bad4"})
	if errs == nil {
		t.Fatal("expected validation errors")
	}
	if len(errs) != 4 {
		t.Errorf("expected errors for 4 fields, got %d: %v", len(errs), errs)
	}
}
