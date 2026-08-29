package releasesync

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSStore struct {
	bucket *oss.Bucket
}

// NewFromEnv returns a disabled worker when OSS is not configured.
func NewFromEnv(repositories []string) (*Worker, error) {
	endpoint := strings.TrimSpace(os.Getenv("OSS_ENDPOINT"))
	bucketName := strings.TrimSpace(os.Getenv("OSS_BUCKET"))
	accessKeyID := firstEnv("OSS_ACCESS_KEY_ID", "ALIBABA_CLOUD_ACCESS_KEY_ID")
	accessKeySecret := firstEnv("OSS_ACCESS_KEY_SECRET", "ALIBABA_CLOUD_ACCESS_KEY_SECRET")
	if endpoint == "" && bucketName == "" && accessKeyID == "" && accessKeySecret == "" {
		return nil, nil
	}
	if endpoint == "" || bucketName == "" || accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("OSS_ENDPOINT, OSS_BUCKET, OSS_ACCESS_KEY_ID, and OSS_ACCESS_KEY_SECRET must all be set")
	}

	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}

	interval := defaultInterval
	if value := os.Getenv("RELEASE_SYNC_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed < time.Minute {
			return nil, fmt.Errorf("invalid RELEASE_SYNC_INTERVAL %q (minimum 1m)", value)
		}
		interval = parsed
	}
	limit := defaultReleaseLimit
	if value := os.Getenv("RELEASE_SYNC_LIMIT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return nil, fmt.Errorf("invalid RELEASE_SYNC_LIMIT %q (expected 1-100)", value)
		}
		limit = parsed
	}
	return New(&OSSStore{bucket: bucket}, repositories, interval, limit, os.Getenv("GITHUB_TOKEN")), nil
}

func (s *OSSStore) ObjectExists(key string) (bool, error) {
	return s.bucket.IsObjectExist(key)
}

func (s *OSSStore) Put(key string, body io.Reader, contentType, cacheControl string) error {
	return s.bucket.PutObject(
		key,
		body,
		oss.ContentType(contentType),
		oss.CacheControl(cacheControl),
	)
}

func (s *OSSStore) Delete(key string) error {
	return s.bucket.DeleteObject(key)
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
