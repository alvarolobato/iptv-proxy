package server

import (
	"regexp"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
)

// UserAccessFilter holds compiled regex patterns for per-user content access control.
// Allow lists restrict to matches only; block lists remove matches.
// Empty lists impose no restriction (default = full access).
type UserAccessFilter struct {
	GroupAllow   []*regexp.Regexp
	GroupBlock   []*regexp.Regexp
	ChannelAllow []*regexp.Regexp
	ChannelBlock []*regexp.Regexp
}

// NewUserAccessFilter compiles the user's access rule patterns into a filter.
// Returns nil if the user has no access rules configured (fast path).
func NewUserAccessFilter(u *config.User) *UserAccessFilter {
	if u == nil {
		return nil
	}
	if len(u.GroupAllowList) == 0 && len(u.GroupBlockList) == 0 &&
		len(u.ChannelAllowList) == 0 && len(u.ChannelBlockList) == 0 {
		return nil
	}
	return &UserAccessFilter{
		GroupAllow:   compileRegexList(u.GroupAllowList, "user_group_allow"),
		GroupBlock:   compileRegexList(u.GroupBlockList, "user_group_block"),
		ChannelAllow: compileRegexList(u.ChannelAllowList, "user_channel_allow"),
		ChannelBlock: compileRegexList(u.ChannelBlockList, "user_channel_block"),
	}
}

// MatchTrack returns true if the track (identified by group and channel name) is allowed
// for the user. Logic: allow lists first (whitelist), then block lists (blacklist).
func (f *UserAccessFilter) MatchTrack(group, channelName string) bool {
	if f == nil {
		return true
	}
	return matchInclusionExclusion(group, channelName, f.GroupAllow, f.GroupBlock, f.ChannelAllow, f.ChannelBlock)
}

// HasRules returns true if the filter has any compiled rules.
func (f *UserAccessFilter) HasRules() bool {
	if f == nil {
		return false
	}
	return len(f.GroupAllow)+len(f.GroupBlock)+len(f.ChannelAllow)+len(f.ChannelBlock) > 0
}

// ValidatePatterns checks that all patterns in the given lists are valid regexes.
// Returns a map of field name -> list of invalid patterns with error messages.
func ValidatePatterns(groupAllow, groupBlock, channelAllow, channelBlock []string) map[string][]string {
	errors := make(map[string][]string)
	check := func(field string, patterns []string) {
		for _, p := range patterns {
			if p == "" {
				continue
			}
			if _, err := regexp.Compile(p); err != nil {
				errors[field] = append(errors[field], p+": "+err.Error())
			}
		}
	}
	check("group_allow_list", groupAllow)
	check("group_block_list", groupBlock)
	check("channel_allow_list", channelAllow)
	check("channel_block_list", channelBlock)
	if len(errors) == 0 {
		return nil
	}
	return errors
}
