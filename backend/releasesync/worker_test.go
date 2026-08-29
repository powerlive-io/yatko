package releasesync

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *memoryStore) ObjectExists(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok, nil
}

func (s *memoryStore) Put(key string, body io.Reader, _, _ string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.objects[key] = data
	s.mu.Unlock()
	return nil
}

func (s *memoryStore) Delete(key string) error {
	s.mu.Lock()
	delete(s.objects, key)
	s.mu.Unlock()
	return nil
}

func TestSyncReleaseWritesManifestLastAndIsIdempotent(t *testing.T) {
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("release"))
	}))
	defer assetServer.Close()

	store := &memoryStore{objects: make(map[string][]byte)}
	worker := New(store, nil, time.Minute, 5, "")
	release := Release{
		ID:      1,
		TagName: "v1.0.0",
		Assets: []Asset{{
			Name:               "app.zip",
			BrowserDownloadURL: assetServer.URL,
			Size:               int64(len("release")),
		}},
	}
	if err := worker.syncRelease(context.Background(), "Xinrea/JLiverTool", release); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.objects["releases/Xinrea/JLiverTool/v1.0.0/app.zip"], []byte("release")) {
		t.Fatal("asset was not mirrored")
	}
	if _, ok := store.objects["releases/Xinrea/JLiverTool/v1.0.0/_manifest.json"]; !ok {
		t.Fatal("completion manifest was not written")
	}

	assetServer.Close()
	if err := worker.syncRelease(context.Background(), "Xinrea/JLiverTool", release); err != nil {
		t.Fatalf("completed release should not download again: %v", err)
	}
}

func TestCopyAssetDeletesSizeMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("short"))
	}))
	defer server.Close()

	store := &memoryStore{objects: make(map[string][]byte)}
	worker := New(store, nil, time.Minute, 5, "")
	err := worker.copyAsset(context.Background(), "asset", Asset{
		Name:               "asset",
		BrowserDownloadURL: server.URL,
		Size:               100,
	})
	if err == nil {
		t.Fatal("expected size mismatch")
	}
	if _, ok := store.objects["asset"]; ok {
		t.Fatal("invalid object was not deleted")
	}
}

func TestSyncRepositoryPollsRecentReleases(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Xinrea/JoiAsk/releases":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":2,"tag_name":"v2","assets":[{"name":"app.zip","browser_download_url":"`+server.URL+`/asset","size":3}]}]`)
		case "/asset":
			_, _ = io.WriteString(w, "zip")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{objects: make(map[string][]byte)}
	worker := New(store, nil, time.Minute, 5, "")
	worker.apiBaseURL = server.URL
	if err := worker.syncRepository(context.Background(), "Xinrea/JoiAsk"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.objects["releases/Xinrea/JoiAsk/v2/_manifest.json"]; !ok {
		t.Fatal("polled release was not synchronized")
	}
}
