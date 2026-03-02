# Replacements file

The replacements file lets you rewrite channel names and group titles in the M3U playlist using regex rules. This is useful to clean up provider names, fix encoding, or standardise labels.

- **File name:** `replacements.json`
- **Location:** Inside the folder set by `--json-folder` (e.g. `/data`). So the full path is `<json-folder>/replacements.json`.

---

## File structure

```json
{
  "global-replacements": [ ... ],
  "names-replacements": [ ... ],
  "groups-replacements": [ ... ]
}
```

Each key is an array of rules. Each rule has:

| Field | Description |
|-------|-------------|
| `replace` | A **regex** pattern (Go regexp). Matches are replaced. |
| `with` | Replacement string. Use `$1`, `$2`, etc. for capture groups. |

Rules in each array are applied in order. Invalid regexes are logged and skipped.

---

## What each section does

### global-replacements

Applied to **both** channel names and group titles (and other relevant text) before the other two are applied. Use this for changes you want everywhere (e.g. normalise spaces, fix a character).

**Example:** Replace any run of spaces with a single space:

```json
"global-replacements": [
  { "replace": "\\s+", "with": " " }
]
```

### names-replacements

Applied only to **channel names** (the track name and the value of the `tvg-name` tag when used for display). Use this to rename channels, add prefixes, or fix provider-specific naming.

**Example:** Add a prefix to channel names that start with "ES ":

```json
"names-replacements": [
  { "replace": "^ES ", "with": "ES: " }
]
```

### groups-replacements

Applied only to **group-title** values. Use this to rename or normalise group names (e.g. "Movies", "Sports").

**Example:** Rewrite "M+" to "MOVISTAR+" in group titles:

```json
"groups-replacements": [
  { "replace": "M\\+", "with": "MOVISTAR+" }
]
```

---

## Full example

See [replacements-example.json](replacements-example.json) in this folder. Minimal example:

```json
{
  "global-replacements": [
    { "replace": "\\s+", "with": " " }
  ],
  "names-replacements": [
    { "replace": "^ES ", "with": "ES: " },
    { "replace": "  ", "with": " " }
  ],
  "groups-replacements": [
    { "replace": "M\\+", "with": "MOVISTAR+" }
  ]
}
```

---

## Using with Docker and `/data`

1. Create a folder on the host, e.g. `./data`.
2. Put `replacements.json` inside it.
3. Run with a volume and `--json-folder /data`:

   ```bash
   docker run -d -p 8080:8080 \
     -v $(pwd)/data:/data \
     -e M3U_URL="..." \
     -e JSON_FOLDER="/data" \
     alobato/iptv-proxy2
   ```

Replacements are loaded at startup. Changing the file requires restarting the container.
