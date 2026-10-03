package namegen

import (
	"encoding/json"
	"fmt"
	"strings"
)

// systemPrompt sets the model up as a namer and shows it the bar, using the
// captain's own examples.
const systemPrompt = `You are a naming savant who coins memorable, witty names for software projects, side projects and things people make.

You work like a person who finds names in the world around them. Your raw material is what the project does, the words the person gives you, and anything they mention from their surroundings: things on their desk, a game they play, the weather, a song, their pet.

Techniques you use:
- portmanteau: fuse two words ("acceleread" = ACCELErate + READ, for a tool that ingests documents faster)
- pun: a double meaning or sound-alike ("kimi-no-name-wa" riffs on the anime film Kimi no Na wa, "Your Name", for an app that names things)
- cultural-reference: films, anime, games, books, myths, memes, history
- metaphor: name the thing after what it is like ("nuggets" for an app storing little nuggets of information)
- acronym: a real word whose letters stand for something relevant
- foreign-word: a fitting word from another language, romanised
- alliteration, compound, respelling: sound play, two words joined, or a familiar word spelt differently
- other: anything else that works

Rules:
- A name is short (one to three words, at most 40 characters), easy to say, and works as a repository or package name.
- Every name must connect to the project; the explanation must show the connection in one line, naming the parts it is built from (like "ACCELErate + READ").
- Vary the techniques across the list. Don't lean on one trick.
- Don't pad with generic tech words (hub, ify, ly, AI, pro) unless they make the joke.
- Never repeat a name you were told to avoid, nor a trivial variant of one.
- Answer only with JSON matching the schema.`

// buildUserPrompt describes this request to the model.
func buildUserPrompt(req Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Suggest %d names.\n\n", req.Count)
	fmt.Fprintf(&b, "What it is:\n%s\n", req.Description)

	if len(req.Inspiration) > 0 {
		fmt.Fprintf(&b, "\nSurroundings and inspiration to draw on: %s\n", strings.Join(req.Inspiration, "; "))
	}
	if len(req.Keywords) > 0 {
		fmt.Fprintf(&b, "\nKeywords to work in where they fit: %s\n", strings.Join(req.Keywords, "; "))
	}

	b.WriteString("\nTones (spread the names across these, and label each with the tone it has):\n")
	for _, id := range req.Tones {
		tone, _ := toneByID(id)
		fmt.Fprintf(&b, "- %s: %s\n", tone.ID, tone.Guidance)
	}

	if req.Like != nil {
		fmt.Fprintf(&b, "\nThe person loved the name %q", req.Like.Name)
		if req.Like.Technique != "" {
			fmt.Fprintf(&b, " (%s)", req.Like.Technique)
		}
		if req.Like.Explanation != "" {
			fmt.Fprintf(&b, ": %s", req.Like.Explanation)
		}
		b.WriteString(". Give names in the same spirit: same kind of wordplay and feel, different words.\n")
	}

	if len(req.Avoid) > 0 {
		// The newest names matter most; an enormous list would crowd out the
		// actual brief. Older ones are still filtered out after generation.
		avoid := req.Avoid
		if len(avoid) > 150 {
			avoid = avoid[len(avoid)-150:]
		}
		fmt.Fprintf(&b, "\nAlready suggested, so don't repeat these or close variants: %s\n", strings.Join(avoid, ", "))
	}
	return b.String()
}

// schema is the JSON schema Ollama constrains the reply to.
func schema(tones []string) json.RawMessage {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"names": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":        map[string]any{"type": "string"},
						"technique":   map[string]any{"type": "string", "enum": Techniques},
						"explanation": map[string]any{"type": "string"},
						"tone":        map[string]any{"type": "string", "enum": tones},
					},
					"required": []string{"name", "technique", "explanation", "tone"},
				},
			},
		},
		"required": []string{"names"},
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		panic(err) // a static shape of maps and strings always encodes
	}
	return encoded
}
