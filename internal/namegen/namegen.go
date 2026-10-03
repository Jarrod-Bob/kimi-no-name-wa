// Package namegen turns a description of a project into name ideas: it builds
// the prompt and JSON schema for the model, then validates what comes back,
// dropping malformed or duplicate entries rather than failing the batch.
package namegen

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tone is one flavour a caller can ask for. The ID is what the API takes.
type Tone struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Guidance string `json:"guidance"`
}

// Tones are every tone the generator understands, in display order.
var Tones = []Tone{
	{"witty", "Witty", "clever and quick; a knowing wink, smart rather than silly"},
	{"punny", "Punny", "built on a pun or double meaning; the groan is the point"},
	{"sleek", "Cool / sleek", "short, confident and brandable; sounds like a product you'd trust"},
	{"comical", "Comical", "absurd and goofy; makes people laugh out loud"},
	{"melancholic", "Melancholic", "wistful and bittersweet; dusk, rain, endings, quiet longing"},
	{"joking", "Joking", "tongue-in-cheek and self-deprecating; refuses to take the project seriously"},
	{"mythic", "Mythic / epic", "grand and legendary; gods, quests, constellations, ancient weapons"},
	{"japanese", "Japanese-flavoured", "draws on Japanese words, anime, games, folklore or culture, written in romaji"},
}

// DefaultTones are used when a request names none.
var DefaultTones = []string{"witty", "punny", "sleek"}

// Techniques classify how a name was made. The model must pick one of these;
// anything else it invents is filed under "other".
var Techniques = []string{
	"portmanteau",
	"pun",
	"cultural-reference",
	"metaphor",
	"acronym",
	"foreign-word",
	"alliteration",
	"compound",
	"respelling",
	"other",
}

// Limits on a request, so one call can't ask the model for a novel.
const (
	DefaultCount      = 8
	MaxCount          = 20
	maxDescription    = 2000
	maxListItems      = 20
	maxListItemLength = 80
	maxAvoid          = 500
	maxNameLength     = 40
	maxExplanation    = 300
)

// Seed is a name the caller liked: "more like this" asks for names in its
// spirit.
type Seed struct {
	Name        string `json:"name"`
	Technique   string `json:"technique,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

// Request is the input to a generation, shared by the web UI and
// POST /api/v1/names.
type Request struct {
	Description string   `json:"description"`
	Inspiration []string `json:"inspiration,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Tones       []string `json:"tones,omitempty"`
	Count       int      `json:"count,omitempty"`
	// Avoid lists names already shown, so a re-roll doesn't repeat them.
	Avoid []string `json:"avoid,omitempty"`
	Like  *Seed    `json:"like,omitempty"`
}

// Name is one suggestion.
type Name struct {
	Name        string `json:"name"`
	Technique   string `json:"technique"`
	Explanation string `json:"explanation"`
	Tone        string `json:"tone"`
}

// ValidationError is a request the caller must fix; its message says how.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// Normalize trims and checks req, filling in defaults. It returns a
// *ValidationError for anything the caller must change.
func (req Request) Normalize() (Request, error) {
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		return req, invalid("Describe what you're naming: description is required.")
	}
	if utf8.RuneCountInString(req.Description) > maxDescription {
		return req, invalid("Keep the description under %d characters.", maxDescription)
	}

	var err error
	if req.Inspiration, err = cleanList("inspiration", req.Inspiration, maxListItems); err != nil {
		return req, err
	}
	if req.Keywords, err = cleanList("keywords", req.Keywords, maxListItems); err != nil {
		return req, err
	}
	if req.Avoid, err = cleanList("avoid", req.Avoid, maxAvoid); err != nil {
		return req, err
	}

	tones := make([]string, 0, len(req.Tones))
	for _, raw := range req.Tones {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		if _, ok := toneByID(id); !ok {
			return req, invalid("Unknown tone %q. Pick from: %s.", raw, strings.Join(ToneIDs(), ", "))
		}
		if !contains(tones, id) {
			tones = append(tones, id)
		}
	}
	if len(tones) == 0 {
		tones = append(tones, DefaultTones...)
	}
	req.Tones = tones

	if req.Count == 0 {
		req.Count = DefaultCount
	}
	if req.Count < 1 || req.Count > MaxCount {
		return req, invalid("count must be between 1 and %d.", MaxCount)
	}

	if req.Like != nil {
		seed := Seed{
			Name:        oneLine(req.Like.Name),
			Technique:   oneLine(req.Like.Technique),
			Explanation: oneLine(req.Like.Explanation),
		}
		if seed.Name == "" {
			return req, invalid("like.name is required when asking for more like a name.")
		}
		if utf8.RuneCountInString(seed.Name) > maxNameLength || utf8.RuneCountInString(seed.Explanation) > maxExplanation {
			return req, invalid("like is too long to be a name from this app.")
		}
		req.Like = &seed
	}
	return req, nil
}

func cleanList(field string, items []string, limit int) ([]string, error) {
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item = oneLine(item)
		if item == "" {
			continue
		}
		if utf8.RuneCountInString(item) > maxListItemLength {
			return nil, invalid("Each %s entry must be under %d characters.", field, maxListItemLength)
		}
		key := strings.ToLower(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	if len(out) > limit {
		return nil, invalid("Give at most %d %s entries.", limit, field)
	}
	return out, nil
}

// ToneIDs lists the tone IDs in display order.
func ToneIDs() []string {
	ids := make([]string, len(Tones))
	for i, t := range Tones {
		ids[i] = t.ID
	}
	return ids
}

func toneByID(id string) (Tone, bool) {
	for _, t := range Tones {
		if t.ID == id {
			return t, true
		}
	}
	return Tone{}, false
}

// Key is a name's identity for duplicate checks: case, spacing and
// punctuation don't make "Accele-Read" a different name from "acceleread".
func Key(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// oneLine trims s and collapses every run of whitespace, newlines included,
// to a single space.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
