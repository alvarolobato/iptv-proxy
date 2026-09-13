/*
 * iptv-proxy2 is a fork of Iptv-Proxy by Pierre-Emmanuel Jacquier.
 * Original project: https://github.com/pierre-emmanuel-jacquier/iptv-proxy
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * New additions and modifications in this fork:
 * Copyright (C) 2026  Alvaro Lobato (github.com/alvarolobato/iptv-proxy)
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

import "testing"

// TestIncludedChannels verifies ?included=1 keeps only the final list: excluded rows are dropped, included rows
// are kept in order whether or not they have a stream URL.
func TestIncludedChannels(t *testing.T) {
	rows := []channelRowProcessed{
		{Name: "a", StreamURL: "http://proxy/play/u/p/1.ts"},
		{Name: "b", Excluded: true, StreamURL: "http://proxy/play/u/p/2.ts"},
		{Name: "c"},
	}
	got := includedChannels(rows)
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "c" {
		t.Fatalf("includedChannels() = %+v, want rows a and c", got)
	}
	for _, r := range got {
		if r.Excluded {
			t.Errorf("includedChannels() kept excluded row %q", r.Name)
		}
	}
}
