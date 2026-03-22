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

package config

import (
	"testing"
)

func TestProxyConfig_DebugAndCacheFields(t *testing.T) {
	conf := &ProxyConfig{
		DebugLoggingEnabled: true,
		CacheFolder:         "/var/cache/iptv",
	}
	if !conf.DebugLoggingEnabled {
		t.Error("DebugLoggingEnabled should be true")
	}
	if conf.CacheFolder != "/var/cache/iptv" {
		t.Errorf("CacheFolder = %q", conf.CacheFolder)
	}
}

func TestCredentialString_String(t *testing.T) {
	c := CredentialString("user@pass")
	if got := c.String(); got != "user@pass" {
		t.Errorf("String() = %q, want %q", got, "user@pass")
	}
}

func TestCredentialString_PathEscape(t *testing.T) {
	tests := []struct {
		name string
		c    CredentialString
		want string
	}{
		{"simple", CredentialString("user"), "user"},
		{"with slash", CredentialString("a/b"), "a%2Fb"},
		{"empty", CredentialString(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.PathEscape(); got != tt.want {
				t.Errorf("PathEscape() = %q, want %q", got, tt.want)
			}
		})
	}
}
