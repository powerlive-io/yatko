package cdn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssetURLFallsBackUntilManifestExists(t *testing.T) {
	manifestExists := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("method = %s, want HEAD", r.Method)
		}
		if manifestExists {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resolver := New(server.URL)
	resolver.httpClient = server.Client()
	githubURL := "https://github.com/powerlive-io/JLiverTool/releases/download/v1/app.zip"

	if got := resolver.AssetURL(context.Background(), "powerlive-io", "JLiverTool", "v1", "app.zip", githubURL); got != githubURL {
		t.Fatalf("fallback URL = %q, want %q", got, githubURL)
	}

	// Clear the one-minute negative cache to model the next availability check.
	resolver.mu.Lock()
	resolver.available = make(map[string]availabilityEntry)
	resolver.mu.Unlock()
	manifestExists = true

	want := server.URL + "/releases/powerlive-io/JLiverTool/v1/app.zip"
	if got := resolver.AssetURL(context.Background(), "powerlive-io", "JLiverTool", "v1", "app.zip", githubURL); got != want {
		t.Fatalf("CDN URL = %q, want %q", got, want)
	}
}

func TestAssetURLEscapesPathSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resolver := New(server.URL + "/")
	resolver.httpClient = server.Client()
	got := resolver.AssetURL(context.Background(), "owner", "repo", "release/test", "app setup.zip", "fallback")
	want := server.URL + "/releases/owner/repo/release%2Ftest/app%20setup.zip"
	if got != want {
		t.Fatalf("CDN URL = %q, want %q", got, want)
	}
}
