package namegen

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
)

func TestNormalizeDefaultsAndCleans(t *testing.T) {
	req, err := Request{
		Description: "  a tool to ingest documents quicker ",
		Inspiration: []string{" maplestory ", "", "MapleStory", "rainy\nwindow"},
		Tones:       []string{"Witty", "witty", " japanese "},
	}.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if req.Description != "a tool to ingest documents quicker" {
		t.Errorf("description = %q", req.Description)
	}
	if strings.Join(req.Inspiration, "|") != "maplestory|rainy window" {
		t.Errorf("inspiration = %q", req.Inspiration)
	}
	if strings.Join(req.Tones, ",") != "witty,japanese" {
		t.Errorf("tones = %q", req.Tones)
	}
	if req.Count != DefaultCount {
		t.Errorf("count = %d", req.Count)
	}

	req, _ = Request{Description: "x"}.Normalize()
	if strings.Join(req.Tones, ",") != strings.Join(DefaultTones, ",") {
		t.Errorf("default tones = %q", req.Tones)
	}
}

func TestNormalizeRejects(t *testing.T) {
	cases := map[string]Request{
		"empty description": {Description: "   "},
		"unknown tone":      {Description: "x", Tones: []string{"spicy"}},
		"count too big":     {Description: "x", Count: MaxCount + 1},
		"negative count":    {Description: "x", Count: -1},
		"like without name": {Description: "x", Like: &Seed{Technique: "pun"}},
		"long keyword":      {Description: "x", Keywords: []string{strings.Repeat("k", 81)}},
	}
	for name, req := range cases {
		_, err := req.Normalize()
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("%s: err = %v, want ValidationError", name, err)
		}
	}
}

func TestParseDropsMalformedAndDuplicates(t *testing.T) {
	content := `{"names":[
		{"name":"acceleread","technique":"portmanteau","explanation":"ACCELErate + READ","tone":"witty"},
		{"name":"Accele-Read","technique":"portmanteau","explanation":"same again","tone":"witty"},
		{"name":"","technique":"pun","explanation":"empty name","tone":"witty"},
		{"name":"nuggets","technique":"metaphor","explanation":"","tone":"witty"},
		{"name":"skimurai","technique":"pun","explanation":"skim + samurai","tone":"spicy"},
		"not an object",
		{"name":42},
		{"name":"\"Yomi\"","technique":"Foreign Word","explanation":"Japanese for reading","tone":"Japanese-flavoured"},
		{"name":"old-favourite","technique":"pun","explanation":"shown before","tone":"witty"},
		{"name":"Pagecrusher","technique":"mashup","explanation":"page + crusher","tone":"witty"}
	]}`
	names, dropped := Parse(content, []string{"witty", "japanese"}, []string{"Old Favourite"})

	got := make([]string, len(names))
	for i, n := range names {
		got[i] = n.Name + "/" + n.Technique + "/" + n.Tone
	}
	want := "acceleread/portmanteau/witty|Yomi/foreign-word/japanese|Pagecrusher/other/witty"
	if strings.Join(got, "|") != want {
		t.Errorf("names = %s\nwant    %s", strings.Join(got, "|"), want)
	}
	if dropped != 7 {
		t.Errorf("dropped = %d, want 7", dropped)
	}
}

func TestParseToleratesFencesAndBareArrays(t *testing.T) {
	entry := `{"name":"nuggets","technique":"metaphor","explanation":"little nuggets of information","tone":"witty"}`
	for _, content := range []string{
		"```json\n{\"names\":[" + entry + "]}\n```",
		"[" + entry + "]",
		"Here you go: [" + entry + "] enjoy!",
	} {
		names, _ := Parse(content, []string{"witty"}, nil)
		if len(names) != 1 || names[0].Name != "nuggets" {
			t.Errorf("Parse(%q) = %+v", content, names)
		}
	}
	if names, _ := Parse("not json at all", []string{"witty"}, nil); len(names) != 0 {
		t.Errorf("garbage parsed to %+v", names)
	}
}

func TestParseSingleToneForgivesLabel(t *testing.T) {
	names, _ := Parse(`{"names":[{"name":"Dusk","technique":"metaphor","explanation":"the end of the day","tone":"sad"}]}`, []string{"melancholic"}, nil)
	if len(names) != 1 || names[0].Tone != "melancholic" {
		t.Errorf("names = %+v", names)
	}
}

func TestKey(t *testing.T) {
	if Key("Kimi-no Name_wa!") != "kiminonamewa" || Key("君の名は") != "君の名は" {
		t.Errorf("Key misbehaves: %q %q", Key("Kimi-no Name_wa!"), Key("君の名は"))
	}
}

type fakeChat struct {
	replies []string
	err     error
	calls   []ollama.ChatRequest
}

func (f *fakeChat) Chat(_ context.Context, req ollama.ChatRequest) (string, error) {
	f.calls = append(f.calls, req)
	if f.err != nil {
		return "", f.err
	}
	reply := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	return reply, nil
}

func TestGenerateBuildsPromptAndTrims(t *testing.T) {
	var entries []string
	for _, n := range []string{"one", "two", "three", "four"} {
		entries = append(entries, `{"name":"`+n+`","technique":"pun","explanation":"because","tone":"punny"}`)
	}
	chat := &fakeChat{replies: []string{`{"names":[` + strings.Join(entries, ",") + `]}`}}
	req, _ := Request{
		Description: "an app that names things",
		Inspiration: []string{"anime posters"},
		Keywords:    []string{"name"},
		Tones:       []string{"punny"},
		Count:       3,
		Avoid:       []string{"kimi-no-name-wa"},
		Like:        &Seed{Name: "acceleread", Technique: "portmanteau", Explanation: "ACCELErate + READ"},
	}.Normalize()

	names, err := Generate(context.Background(), chat, "gemma4:31b", req)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(names) != 3 || names[2].Name != "three" {
		t.Errorf("names = %+v, want first three", names)
	}

	call := chat.calls[0]
	if call.Model != "gemma4:31b" || !strings.Contains(call.System, "acceleread") {
		t.Errorf("model/system wrong: %q", call.Model)
	}
	for _, want := range []string{"Suggest 5 names", "an app that names things", "anime posters", "name", "punny:", `"acceleread"`, "kimi-no-name-wa"} {
		if !strings.Contains(call.User, want) {
			t.Errorf("user prompt missing %q:\n%s", want, call.User)
		}
	}
	var s struct {
		Properties struct {
			Names struct {
				Items struct {
					Properties struct {
						Tone struct {
							Enum []string `json:"enum"`
						} `json:"tone"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"names"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(call.Format, &s); err != nil || strings.Join(s.Properties.Names.Items.Properties.Tone.Enum, ",") != "punny" {
		t.Errorf("schema tone enum = %v (%v)", s.Properties.Names.Items.Properties.Tone.Enum, err)
	}
}

func TestGenerateRetriesOnceThenGivesUp(t *testing.T) {
	good := `{"names":[{"name":"nuggets","technique":"metaphor","explanation":"little nuggets","tone":"witty"}]}`
	req, _ := Request{Description: "ideas", Tones: []string{"witty"}}.Normalize()

	chat := &fakeChat{replies: []string{"garbage", good}}
	names, err := Generate(context.Background(), chat, "m", req)
	if err != nil || len(names) != 1 || len(chat.calls) != 2 {
		t.Fatalf("retry: names %+v, err %v, calls %d", names, err, len(chat.calls))
	}

	chat = &fakeChat{replies: []string{"garbage"}}
	if _, err := Generate(context.Background(), chat, "m", req); !errors.Is(err, ErrNoNames) || len(chat.calls) != 2 {
		t.Fatalf("give up: err %v, calls %d", err, len(chat.calls))
	}

	down := &ollama.UnreachableError{URL: "http://x", Err: errors.New("refused")}
	chat = &fakeChat{err: down}
	if _, err := Generate(context.Background(), chat, "m", req); !errors.Is(err, down) || len(chat.calls) != 1 {
		t.Fatalf("model error should pass straight through: %v, calls %d", err, len(chat.calls))
	}
}

func TestGenerateTopsUpAShortBatch(t *testing.T) {
	entry := func(n string) string {
		return `{"name":"` + n + `","technique":"pun","explanation":"because","tone":"witty"}`
	}
	req, _ := Request{Description: "ideas", Tones: []string{"witty"}, Count: 6, Avoid: []string{"old"}}.Normalize()

	chat := &fakeChat{replies: []string{
		`{"names":[` + entry("old") + `,` + entry("one") + `]}`,
		`{"names":[` + entry("one") + `,` + entry("two") + `,` + entry("three") + `]}`,
	}}
	names, err := Generate(context.Background(), chat, "m", req)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(namesOf(names), ","); got != "one,two,three" {
		t.Errorf("names = %s, want one,two,three", got)
	}
	if len(chat.calls) != 2 || !strings.Contains(chat.calls[1].User, "old, one") {
		t.Errorf("second ask should avoid the first reply too:\n%s", chat.calls[len(chat.calls)-1].User)
	}

	// A failing top-up keeps what the first reply gave.
	chat = &fakeChat{replies: []string{`{"names":[` + entry("solo") + `]}`}}
	calls := 0
	failing := chatFunc(func(ctx context.Context, r ollama.ChatRequest) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("boom")
		}
		return chat.Chat(ctx, r)
	})
	names, err = Generate(context.Background(), failing, "m", req)
	if err != nil || len(names) != 1 {
		t.Errorf("failed top-up: %v, %v", names, err)
	}
}

type chatFunc func(context.Context, ollama.ChatRequest) (string, error)

func (f chatFunc) Chat(ctx context.Context, r ollama.ChatRequest) (string, error) { return f(ctx, r) }
