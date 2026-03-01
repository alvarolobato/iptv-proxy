/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY. See the GNU General Public License for more details.
 */

package server

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// Replacements holds regex replacement rules for M3U channel names and group titles.
type Replacements struct {
	Global []Replacement `json:"global-replacements"`
	Names  []Replacement `json:"names-replacements"`
	Groups []Replacement `json:"groups-replacements"`
}

// Replacement is a single regex replace rule.
type Replacement struct {
	Replace string `json:"replace"`
	With    string `json:"with"`
}

func loadReplacements(filename string) Replacements {
	var replacements Replacements
	if filename == "" {
		return replacements
	}
	info, err := os.Stat(filename)
	if err != nil || info.IsDir() {
		return replacements
	}
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("[iptv-proxy] Could not open replacements file %s: %v", filename, err)
		return replacements
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&replacements); err != nil {
		log.Printf("[iptv-proxy] Could not parse replacements file %s: %v", filename, err)
		return replacements
	}
	log.Printf("[iptv-proxy] Loaded replacements from %s", filepath.Base(filename))
	return replacements
}

func applyReplacements(rules []Replacement, value string) string {
	for _, r := range rules {
		re, err := regexp.Compile(r.Replace)
		if err != nil {
			log.Printf("[iptv-proxy] Invalid replacement regex %q: %v", r.Replace, err)
			continue
		}
		value = re.ReplaceAllString(value, r.With)
	}
	return value
}
