package namecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServices stands in for GitHub search and the npm registry.
func fakeServices(t *testing.T, repos []map[string]any, npmPackages []string, githubStatus int) (*Checker, *atomic.Int32) {
	t.Helper()
	var githubCalls atomic.Int32
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		githubCalls.Add(1)
		if r.URL.Path != "/search/repositories" || r.Header.Get("User-Agent") == "" {
			t.Errorf("unexpected GitHub request %s (UA %q)", r.URL, r.Header.Get("User-Agent"))
		}
		if githubStatus != 0 {
			w.WriteHeader(githubStatus)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"total_count": len(repos) + 3, "items": repos})
	}))
	t.Cleanup(gh.Close)
	npm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, p := range npmPackages {
			if r.URL.Path == "/"+p {
				w.Write([]byte(`{"name":"` + p + `"}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(npm.Close)
	return &Checker{GitHubAPI: gh.URL, NPMAPI: npm.URL, HTTPClient: gh.Client(), TTL: time.Minute}, &githubCalls
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Kimi no Name wa": "kimi-no-name-wa",
		" Accele_Read! ":  "accele-read",
		"--yomi--":        "yomi",
		"君の名は":            "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckTakenAndFree(t *testing.T) {
	c, calls := fakeServices(t, []map[string]any{
		{"name": "nuggets-ui", "full_name": "a/nuggets-ui", "html_url": "https://github.com/a/nuggets-ui", "stargazers_count": 50},
		{"name": "Nuggets", "full_name": "b/Nuggets", "html_url": "https://github.com/b/Nuggets", "stargazers_count": 3},
		{"name": "nuggets", "full_name": "c/nuggets", "html_url": "https://github.com/c/nuggets", "stargazers_count": 9},
	}, []string{"nuggets"}, 0)

	res := c.Check(context.Background(), "Nuggets")
	if res.Slug != "nuggets" || res.GitHub.Status != Taken || res.GitHub.URL != "https://github.com/c/nuggets" || !strings.Contains(res.GitHub.Detail, "1 more") {
		t.Errorf("github = %+v", res.GitHub)
	}
	if res.NPM.Status != Taken {
		t.Errorf("npm = %+v", res.NPM)
	}

	// Cached: no second GitHub search.
	c.Check(context.Background(), "nuggets")
	if calls.Load() != 1 {
		t.Errorf("GitHub calls = %d, want 1 (cached)", calls.Load())
	}

	res = c.Check(context.Background(), "acceleread")
	if res.NPM.Status != Free {
		t.Errorf("npm for unknown package = %+v", res.NPM)
	}
}

func TestCheckRateLimitedIsUnknownAndNotCached(t *testing.T) {
	c, calls := fakeServices(t, nil, nil, http.StatusForbidden)
	for i := 0; i < 2; i++ {
		res := c.Check(context.Background(), "yomi")
		if res.GitHub.Status != Unknown || !strings.Contains(res.GitHub.Detail, "limit") {
			t.Errorf("github = %+v", res.GitHub)
		}
		if res.NPM.Status != Free {
			t.Errorf("npm = %+v", res.NPM)
		}
	}
	if calls.Load() != 2 {
		t.Errorf("GitHub calls = %d, want 2 (unknown is not cached)", calls.Load())
	}
}

func TestCheckUnreachable(t *testing.T) {
	c := &Checker{GitHubAPI: "http://127.0.0.1:1", NPMAPI: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: time.Second}}
	res := c.Check(context.Background(), "yomi")
	if res.GitHub.Status != Unknown || res.NPM.Status != Unknown {
		t.Errorf("res = %+v", res)
	}
}
