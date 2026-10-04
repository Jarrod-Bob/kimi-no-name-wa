package namegen

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// Parse reads the model's reply and returns the valid, distinct names in it,
// in order. Entries that are malformed, use a tone that wasn't asked for, or
// repeat an earlier entry or a name in avoid are dropped individually;
// dropped counts them. An unreadable reply yields no names, not an error: the
// caller decides whether an empty batch is worth a retry.
func Parse(content string, tones []string, avoid []string) (names []Name, dropped int) {
	entries := entriesOf(content)
	seen := map[string]bool{}
	for _, key := range avoid {
		seen[Key(key)] = true
	}
	for _, raw := range entries {
		var entry struct {
			Name        string `json:"name"`
			Technique   string `json:"technique"`
			Explanation string `json:"explanation"`
			Tone        string `json:"tone"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			dropped++
			continue
		}
		name, ok := cleanName(entry.Name)
		explanation := oneLine(entry.Explanation)
		tone, toneOK := matchTone(entry.Tone, tones)
		if !ok || !toneOK || explanation == "" || utf8.RuneCountInString(explanation) > maxExplanation {
			dropped++
			continue
		}
		key := Key(name)
		if seen[key] {
			dropped++
			continue
		}
		seen[key] = true
		names = append(names, Name{
			Name:        name,
			Technique:   matchTechnique(entry.Technique),
			Explanation: explanation,
			Tone:        tone,
		})
	}
	return names, dropped
}

// entriesOf finds the array of name objects in content: {"names":[...]} as
// the schema asks, or a bare array from a model that ignored it, possibly
// wrapped in a Markdown code fence or chatter.
func entriesOf(content string) []json.RawMessage {
	content = strings.TrimSpace(content)
	var wrapped struct {
		Names []json.RawMessage `json:"names"`
	}
	if err := json.Unmarshal([]byte(content), &wrapped); err == nil && wrapped.Names != nil {
		return wrapped.Names
	}
	var bare []json.RawMessage
	if err := json.Unmarshal([]byte(content), &bare); err == nil {
		return bare
	}
	// Last resort: the outermost {...} or [...] in surrounding text.
	if start, end := strings.Index(content, "{"), strings.LastIndex(content, "}"); start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end+1]), &wrapped); err == nil && wrapped.Names != nil {
			return wrapped.Names
		}
	}
	if start, end := strings.Index(content, "["), strings.LastIndex(content, "]"); start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end+1]), &bare); err == nil {
			return bare
		}
	}
	return nil
}

// cleanName trims quotes and whitespace a model sometimes wraps names in and
// checks what's left looks like a name.
func cleanName(raw string) (string, bool) {
	name := oneLine(raw)
	name = strings.Trim(name, "\"'`“”‘’*")
	name = strings.TrimSpace(name)
	if name == "" || Key(name) == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", false
	}
	return name, true
}

// matchTone maps the model's tone label onto a requested tone ID. A model
// that mislabels the tone when only one was asked for gets the benefit of the
// doubt; otherwise an unrequested tone is malformed.
func matchTone(raw string, tones []string) (string, bool) {
	want := strings.ToLower(strings.TrimSpace(raw))
	for _, id := range tones {
		tone, _ := toneByID(id)
		if want == id || want == strings.ToLower(tone.Label) {
			return id, true
		}
	}
	if len(tones) == 1 {
		return tones[0], true
	}
	return "", false
}

var techniqueAliases = map[string]string{
	"cultural":         "cultural-reference",
	"culture":          "cultural-reference",
	"pop-culture":      "cultural-reference",
	"reference":        "cultural-reference",
	"loanword":         "foreign-word",
	"foreign":          "foreign-word",
	"foreign-language": "foreign-word",
	"blend":            "portmanteau",
	"compound-word":    "compound",
	"misspelling":      "respelling",
	"wordplay":         "pun",
	"double-meaning":   "pun",
}

func matchTechnique(raw string) string {
	t := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(raw, "_", " ")), "-"))
	if contains(Techniques, t) {
		return t
	}
	if alias, ok := techniqueAliases[t]; ok {
		return alias
	}
	return "other"
}
