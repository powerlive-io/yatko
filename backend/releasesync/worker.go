package releasesync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultInterval     = 10 * time.Minute
	defaultReleaseLimit = 5
	maxAPIResponseSize  = 5 << 20
)

type Store interface {
	ObjectExists(key string) (bool, error)
	Put(key string, body io.Reader, contentType, cacheControl string) error
	Delete(key string) error
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

type Release struct {
	ID      int64   `json:"id"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Status struct {
	Enabled     bool   `json:"enabled"`
	Running     bool   `json:"running"`
	LastSuccess string `json:"last_success,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

// Worker periodically mirrors missing GitHub release assets into object
// storage. A manifest is written last and acts as the atomic completion marker.
type Worker struct {
	store        Store
	httpClient   *http.Client
	apiBaseURL   string
	githubToken  string
	repositories []string
	interval     time.Duration
	releaseLimit int
	running      atomic.Bool
	lastSuccess  atomic.Value
	lastError    atomic.Value
}

func New(store Store, repositories []string, interval time.Duration, releaseLimit int, githubToken string) *Worker {
	return &Worker{
		store:        store,
		httpClient:   &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second}},
		apiBaseURL:   "https://api.github.com",
		githubToken:  githubToken,
		repositories: repositories,
		interval:     interval,
		releaseLimit: releaseLimit,
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.poll(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Worker) poll(ctx context.Context) {
	if !w.running.CompareAndSwap(false, true) {
		return
	}
	defer w.running.Store(false)

	var pollError string
	for _, repository := range w.repositories {
		if err := w.syncRepository(ctx, repository); err != nil {
			message := fmt.Sprintf("%s: %v", repository, err)
			w.lastError.Store(message)
			pollError = message
			log.Printf("release sync failed: %s", message)
			continue
		}
	}
	if pollError == "" {
		w.lastError.Store("")
		w.lastSuccess.Store(time.Now().UTC().Format(time.RFC3339))
	}
}

func (w *Worker) Status() Status {
	status := Status{Enabled: w != nil}
	if w == nil {
		return status
	}
	status.Running = w.running.Load()
	if value := w.lastSuccess.Load(); value != nil {
		status.LastSuccess, _ = value.(string)
	}
	if value := w.lastError.Load(); value != nil {
		status.LastError, _ = value.(string)
	}
	return status
}

func (w *Worker) syncRepository(ctx context.Context, repository string) error {
	releases, err := w.getReleases(ctx, repository)
	if err != nil {
		return err
	}
	for _, release := range releases {
		if err := w.syncRelease(ctx, repository, release); err != nil {
			return fmt.Errorf("%s: %w", release.TagName, err)
		}
	}
	return nil
}

func (w *Worker) syncRelease(ctx context.Context, repository string, release Release) error {
	prefix := "releases/" + repository + "/" + release.TagName
	manifestKey := prefix + "/_manifest.json"
	complete, err := w.store.ObjectExists(manifestKey)
	if err != nil {
		return fmt.Errorf("checking manifest: %w", err)
	}
	if complete {
		return nil
	}

	for _, asset := range release.Assets {
		if asset.Name == "" || strings.ContainsAny(asset.Name, `/\`) {
			return fmt.Errorf("unsafe asset name %q", asset.Name)
		}
		key := prefix + "/" + asset.Name
		exists, err := w.store.ObjectExists(key)
		if err != nil {
			return fmt.Errorf("checking %s: %w", asset.Name, err)
		}
		if exists {
			continue
		}
		if err := w.copyAsset(ctx, key, asset); err != nil {
			return err
		}
	}

	manifest := struct {
		Repository string  `json:"repository"`
		Tag        string  `json:"tag"`
		ReleaseID  int64   `json:"releaseId"`
		SyncedAt   string  `json:"syncedAt"`
		Assets     []Asset `json:"assets"`
	}{
		Repository: repository,
		Tag:        release.TagName,
		ReleaseID:  release.ID,
		SyncedAt:   time.Now().UTC().Format(time.RFC3339),
		Assets:     release.Assets,
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	if err := w.store.Put(manifestKey, bytes.NewReader(data), "application/json; charset=utf-8", "no-cache, no-store, must-revalidate"); err != nil {
		return fmt.Errorf("uploading manifest: %w", err)
	}
	log.Printf("release sync complete: %s@%s (%d assets)", repository, release.TagName, len(release.Assets))
	return nil
}

func (w *Worker) copyAsset(ctx context.Context, key string, asset Asset) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("creating download request for %s: %w", asset.Name, err)
	}
	req.Header.Set("User-Agent", "PowerLive-release-sync/1.0")
	if w.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+w.githubToken)
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("downloading %s: GitHub returned %s", asset.Name, resp.Status)
	}

	contentType := asset.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	counter := &countingReader{reader: resp.Body}
	if err := w.store.Put(key, counter, contentType, "public, max-age=31536000, immutable"); err != nil {
		return fmt.Errorf("uploading %s: %w", asset.Name, err)
	}
	if counter.count != asset.Size {
		_ = w.store.Delete(key)
		return fmt.Errorf("copying %s: got %d bytes, expected %d", asset.Name, counter.count, asset.Size)
	}
	return nil
}

func (w *Worker) getReleases(ctx context.Context, repository string) ([]Release, error) {
	endpoint := w.apiBaseURL + "/repos/" + escapeRepository(repository) +
		"/releases?per_page=" + strconv.Itoa(w.releaseLimit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "PowerLive-release-sync/1.0")
	if w.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+w.githubToken)
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned %s", resp.Status)
	}
	var releases []Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxAPIResponseSize)).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func escapeRepository(repository string) string {
	parts := strings.SplitN(repository, "/", 2)
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1])
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count += int64(n)
	return n, err
}
