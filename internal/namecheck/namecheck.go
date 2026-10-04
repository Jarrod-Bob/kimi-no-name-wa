// Package namecheck answers "is this name already taken?" for a suggested
// name, best-effort and without API keys: is there a GitHub repository with
// exactly this name, and is it an npm package? Checks run only when someone
// asks about one name, and results are cached briefly because GitHub allows
// only ten unauthenticated searches a minute.
package namecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Status values for one source.
const (
	Taken   = "taken"
	Free    = "free"
	Unknown = "unknown" // the check couldn't run (offline, rate-limited, ...)
)

// SourceResult is one source's answer.
type SourceResult struct {
	Status string `json:"status"`
	// URL links to the existing project when Status is taken, else to a
	// search the person can look through themselves.
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

// Result is the answer for one name.
type Result struct {
	Name   string       `json:"name"`
	Slug   string       `json:"slug"`
	GitHub SourceResult `json:"github"`
	NPM    SourceResult `json:"npm"`
}

// Checker runs checks. Its base URLs are fields so tests can point it at
// httptest fakes; nothing in tests may reach the real services.
type Checker struct {
	GitHubAPI  string // default https://api.github.com
	NPMAPI     string // default https://registry.npmjs.org
	HTTPClient *http.Client
	TTL        time.Duration

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	result Result
	at     time.Time
}

func New() *Checker {
	return &Checker{
		GitHubAPI:  "https://api.github.com",
		NPMAPI:     "https://registry.npmjs.org",
		HTTPClient: &http.Client{Timeout: 8 * time.Second},
		TTL:        10 * time.Minute,
	}
}

var slugStrip = regexp.MustCompile(`[^a-z0-9._-]+`)
var slugDashes = regexp.MustCompile(`-{2,}`)

// Slug turns a name into the form a repository or npm package would use:
// lower case, spaces as dashes, nothing npm wouldn't accept.
func Slug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.NewReplacer(" ", "-", "_", "-").Replace(s)
	s = slugStrip.ReplaceAllString(s, "")
	s = slugDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-.")
}

// Check looks name up on GitHub and npm in parallel. It never fails: a
// source that can't answer reports Unknown with a reason.
func (c *Checker) Check(ctx context.Context, name string) Result {
	slug := Slug(name)
	res := Result{Name: name, Slug: slug}
	if slug == "" {
		unknown := SourceResult{Status: Unknown, Detail: "This name has no letters or digits a repository or package could use."}
		res.GitHub, res.NPM = unknown, unknown
		return res
	}

	c.mu.Lock()
	if hit, ok := c.cache[slug]; ok && time.Since(hit.at) < c.TTL {
		c.mu.Unlock()
		hit.result.Name = name
		return hit.result
	}
	c.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); res.GitHub = c.github(ctx, slug) }()
	go func() { defer wg.Done(); res.NPM = c.npm(ctx, slug) }()
	wg.Wait()

	// Only cache definite answers, so a rate-limited check can be retried.
	if res.GitHub.Status != Unknown && res.NPM.Status != Unknown {
		c.mu.Lock()
		if c.cache == nil {
			c.cache = map[string]cached{}
		}
		c.cache[slug] = cached{result: res, at: time.Now()}
		c.mu.Unlock()
	}
	return res
}

func (c *Checker) github(ctx context.Context, slug string) SourceResult {
	searchURL := "https://github.com/search?type=repositories&q=" + url.QueryEscape(slug+" in:name")
	endpoint := c.GitHubAPI + "/search/repositories?per_page=20&q=" + url.QueryEscape(slug+" in:name")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return SourceResult{Status: Unknown, URL: searchURL, Detail: err.Error()}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "kimi-no-name-wa")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return SourceResult{Status: Unknown, URL: searchURL, Detail: "Couldn't reach GitHub."}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return SourceResult{Status: Unknown, URL: searchURL, Detail: "GitHub's search limit (10 a minute without a key) was hit; try again in a minute."}
	}
	if resp.StatusCode != http.StatusOK {
		return SourceResult{Status: Unknown, URL: searchURL, Detail: fmt.Sprintf("GitHub answered %d.", resp.StatusCode)}
	}
	var out struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			HTMLURL  string `json:"html_url"`
			Stars    int    `json:"stargazers_count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return SourceResult{Status: Unknown, URL: searchURL, Detail: "GitHub's answer wasn't readable."}
	}
	exact := 0
	best := -1
	for i, item := range out.Items {
		if strings.EqualFold(item.Name, slug) {
			exact++
			if best < 0 || item.Stars > out.Items[best].Stars {
				best = i
			}
		}
	}
	if exact == 0 {
		detail := "No repository is called exactly this."
		if out.TotalCount > 0 {
			detail = fmt.Sprintf("No repository is called exactly this (%d have it in their name).", out.TotalCount)
		}
		return SourceResult{Status: Free, URL: searchURL, Detail: detail}
	}
	top := out.Items[best]
	detail := fmt.Sprintf("%s (%d★)", top.FullName, top.Stars)
	if exact > 1 {
		detail += fmt.Sprintf(" and %d more", exact-1)
	}
	return SourceResult{Status: Taken, URL: top.HTMLURL, Detail: detail}
}

func (c *Checker) npm(ctx context.Context, slug string) SourceResult {
	pkgURL := "https://www.npmjs.com/package/" + slug
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.NPMAPI+"/"+url.PathEscape(slug), nil)
	if err != nil {
		return SourceResult{Status: Unknown, URL: pkgURL, Detail: err.Error()}
	}
	// The abbreviated document is a fraction of the full one's size.
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return SourceResult{Status: Unknown, URL: pkgURL, Detail: "Couldn't reach the npm registry."}
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return SourceResult{Status: Taken, URL: pkgURL, Detail: "npm package " + slug + " exists."}
	case http.StatusNotFound:
		return SourceResult{Status: Free, URL: "https://www.npmjs.com/search?q=" + url.QueryEscape(slug), Detail: "No npm package has this name."}
	default:
		return SourceResult{Status: Unknown, URL: pkgURL, Detail: fmt.Sprintf("npm answered %d.", resp.StatusCode)}
	}
}
