package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/argval/yatko/cache"
	"github.com/argval/yatko/cdn"
	"github.com/argval/yatko/github"
	"github.com/argval/yatko/handlers"
	"github.com/argval/yatko/middleware"
	"github.com/argval/yatko/ratelimit"
	"github.com/argval/yatko/releasesync"
	"github.com/gin-gonic/gin"
)

// defaultRateLimitRPM is the per-IP request budget per minute, overridable
// via RATE_LIMIT_RPM. Uses process-local limiting without Redis; prod should
// set a Redis URL so limits are shared across instances.
const defaultRateLimitRPM = 120

const defaultAllowedRepos = "powerlive-io/bili-shadowreplay,powerlive-io/JLiverTool,powerlive-io/bili-messenger"

func main() {
	ghClient := github.NewClient()
	redisCache := cache.New()
	limiter := ratelimit.New()
	releaseCDN := cdn.New(os.Getenv("RELEASE_CDN_URL"))

	redirectHandler := handlers.NewRedirectHandler(ghClient, redisCache, releaseCDN)
	pageHandler := handlers.NewPageHandler(redirectHandler, ghClient, redisCache)
	linkHandler := handlers.NewLinkHandler(redirectHandler)
	releasesHandler := handlers.NewReleasesHandler(ghClient, redisCache)

	r := gin.Default()

	// Only honor forwarded client IPs from explicitly configured reverse
	// proxies. With no setting, Gin uses the socket address rather than trusting
	// a client-supplied X-Forwarded-For header.
	var trustedProxies []string
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			trustedProxies = append(trustedProxies, proxy)
		}
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}

	// The frontend and backend share one public origin through the deployment
	// proxy, so browser CORS access is not needed.

	rateLimitRPM := defaultRateLimitRPM
	if v := os.Getenv("RATE_LIMIT_RPM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimitRPM = n
		} else {
			fmt.Fprintf(os.Stderr, "warning: invalid RATE_LIMIT_RPM=%q, using default %d\n", v, defaultRateLimitRPM)
		}
	}

	allowedRepos := os.Getenv("ALLOWED_REPOS")
	if allowedRepos == "" {
		allowedRepos = defaultAllowedRepos
	}
	repositories := make([]string, 0)
	for _, repository := range strings.Split(allowedRepos, ",") {
		if repository = strings.TrimSpace(repository); repository != "" {
			if strings.Count(repository, "/") != 1 {
				log.Fatalf("invalid repository in ALLOWED_REPOS: %q", repository)
			}
			repositories = append(repositories, repository)
		}
	}
	if len(repositories) == 0 {
		log.Fatal("ALLOWED_REPOS must contain at least one owner/repo")
	}
	releaseSync, err := releasesync.NewFromEnv(repositories)
	if err != nil {
		log.Fatalf("invalid release sync configuration: %v", err)
	}

	// Rate-limited routes only - /health is exempt so hosting-platform
	// probes (which hit it frequently from an internal IP) are never throttled.
	limited := r.Group("/")
	limited.Use(middleware.RateLimit(limiter, rateLimitRPM, time.Minute))
	limited.Use(middleware.AllowedRepositories(middleware.RepositoryAllowlist(allowedRepos)))
	limited.GET("/dl/:owner/:repo", redirectHandler.Handle)
	limited.GET("/dl/:owner/:repo/:version", redirectHandler.HandleVersioned)
	limited.GET("/code/:owner/:repo", handlers.HandleCode)
	limited.GET("/api/release/:owner/:repo", pageHandler.Handle)
	limited.GET("/api/link/:owner/:repo", linkHandler.Handle)
	limited.GET("/api/link/:owner/:repo/:version", linkHandler.HandleVersioned)
	limited.GET("/api/release/:owner/:repo/:version", pageHandler.HandleVersioned)
	limited.GET("/api/readme/:owner/:repo", pageHandler.HandleREADME)
	limited.GET("/api/releases/:owner/:repo", releasesHandler.Handle)

	r.GET("/health", func(c *gin.Context) {
		redisStatus := "missing"
		if redisCache.RedisConfigured() {
			redisStatus = "configured"
		}
		// Always 200 so platform probes stay healthy; body carries detail.
		c.JSON(200, gin.H{
			"status":       "ok",
			"redis":        redisStatus,
			"rate_limit":   limiter.Backend(),
			"github":       ghClient.Budget(),
			"github_token": ghClient.HasToken(),
			"release_sync": releaseSync.Status(),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if releaseSync != nil {
		go releaseSync.Run(ctx)
	}

	server := &http.Server{Addr: ":" + port, Handler: r}
	go func() {
		log.Printf("PowerLive backend starting on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
}
