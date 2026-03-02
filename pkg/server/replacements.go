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

// Replacements holds compiled regex replacement rules for M3U channel names and group titles.
type Replacements struct {
	Global []CompiledReplacement
	Names  []CompiledReplacement
	Groups []CompiledReplacement
}

// Replacement is a single regex replace rule (JSON shape).
type Replacement struct {
	Replace string `json:"replace"`
	With    string `json:"with"`
}

// CompiledReplacement is a compiled regex and its replacement string.
type CompiledReplacement struct {
	Re   *regexp.Regexp
	With string
}

type replacementsJSON struct {
	Global []Replacement `json:"global-replacements"`
	Names  []Replacement `json:"names-replacements"`
	Groups []Replacement `json:"groups-replacements"`
}

func compileReplacementSlice(rules []Replacement) []CompiledReplacement {
	out := make([]CompiledReplacement, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Replace)
		if err != nil {
			log.Printf("[iptv-proxy] Invalid replacement regex %q: %v", r.Replace, err)
			continue
		}
		out = append(out, CompiledReplacement{Re: re, With: r.With})
	}
	return out
}

// EnsureStubReplacements creates an empty replacements.json with all sections if the file does not exist.
// dir is the JSON folder (e.g. from --json-folder). No-op if dir is empty.
func EnsureStubReplacements(dir string) {
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "replacements.json")
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[iptv-proxy] Could not create JSON folder %s: %v", dir, err)
		return
	}
	stub := replacementsJSON{
		Global: []Replacement{},
		Names:  []Replacement{},
		Groups: []Replacement{},
	}
	data, err := json.MarshalIndent(stub, "", "  ")
	if err != nil {
		log.Printf("[iptv-proxy] Could not marshal stub replacements: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[iptv-proxy] Could not write stub %s: %v", path, err)
		return
	}
	log.Printf("[iptv-proxy] Created stub %s", path)
}

func loadReplacements(filename string) Replacements {
	var out Replacements
	if filename == "" {
		return out
	}
	info, err := os.Stat(filename)
	if err != nil || info.IsDir() {
		return out
	}
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("[iptv-proxy] Could not open replacements file %s: %v", filename, err)
		return out
	}
	defer f.Close()
	var raw replacementsJSON
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		log.Printf("[iptv-proxy] Could not parse replacements file %s: %v", filename, err)
		return out
	}
	out.Global = compileReplacementSlice(raw.Global)
	out.Names = compileReplacementSlice(raw.Names)
	out.Groups = compileReplacementSlice(raw.Groups)
	log.Printf("[iptv-proxy] Loaded replacements from %s", filepath.Base(filename))
	return out
}

func applyReplacements(rules []CompiledReplacement, value string) string {
	for _, r := range rules {
		value = r.Re.ReplaceAllString(value, r.With)
	}
	return value
}
