/*
 * iptv-proxy2 is a fork of Iptv-Proxy by Pierre-Emmanuel Jacquier.
 * Original project: https://github.com/pierre-emmanuel-jacquier/iptv-proxy
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * New additions and modifications in this fork:
 * Copyright (C) 2024  Alvaro Lobato (github.com/alvarolobato/iptv-proxy)
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
