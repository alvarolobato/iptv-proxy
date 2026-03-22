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
	"testing"
	"time"
)

func TestNewResponseCache_Disabled(t *testing.T) {
	c := newResponseCache(0, 10)
	if c == nil {
		t.Fatal("newResponseCache(0) returned nil")
	}
	_, ok := c.Get("x")
	if ok {
		t.Error("Get on disabled cache should miss")
	}
}

func TestResponseCache_GetSet(t *testing.T) {
	c := newResponseCache(10*time.Second, 10)
	if c == nil {
		t.Fatal("newResponseCache returned nil")
	}
	_, ok := c.Get("k")
	if ok {
		t.Error("Get before Set should miss")
	}
	c.Set("k", []byte("payload"), "text/plain")
	ent, ok := c.Get("k")
	if !ok {
		t.Fatal("Get after Set should hit")
	}
	if string(ent.payload) != "payload" || ent.contentType != "text/plain" {
		t.Errorf("entry = %q, %q", ent.payload, ent.contentType)
	}
}
