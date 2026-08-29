package cdn

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const availabilityTTL = time.Minute

type availabilityEntry struct {
	available bool
	expiresAt time.Time
}

// Resolver switches release assets to a CDN only after the release manifest
// exists there. Until then, callers keep using the original GitHub URL.
type Resolver struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
	available  map[string]availabilityEntry
}

func New(baseURL string) *Resolver {
	return &Resolver{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		available: make(map[string]availabilityEntry),
	}
}

// AssetURL returns the CDN URL when a completed release manifest is present,
// otherwise it returns githubURL so downloads remain available during sync.
func (r *Resolver) AssetURL(ctx context.Context, owner, repo, tag, name, githubURL string) string {
	if r == nil || r.baseURL == "" || !r.releaseAvailable(ctx, owner, repo, tag) {
		return githubURL
	}
	return r.releaseURL(owner, repo, tag, name)
}

func (r *Resolver) releaseAvailable(ctx context.Context, owner, repo, tag string) bool {
	manifestURL := r.releaseURL(owner, repo, tag, "_manifest.json")
	now := time.Now()

	r.mu.Lock()
	if entry, ok := r.available[manifestURL]; ok && now.Before(entry.expiresAt) {
		r.mu.Unlock()
		return entry.available
	}
	r.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	available := false
	if err == nil {
		if resp, requestErr := r.httpClient.Do(req); requestErr == nil {
			available = resp.StatusCode >= 200 && resp.StatusCode < 300
			_ = resp.Body.Close()
		}
	}

	r.mu.Lock()
	r.available[manifestURL] = availabilityEntry{
		available: available,
		expiresAt: now.Add(availabilityTTL),
	}
	r.mu.Unlock()
	return available
}

func (r *Resolver) releaseURL(owner, repo, tag, name string) string {
	return r.baseURL + "/releases/" +
		url.PathEscape(owner) + "/" +
		url.PathEscape(repo) + "/" +
		url.PathEscape(tag) + "/" +
		url.PathEscape(name)
}
