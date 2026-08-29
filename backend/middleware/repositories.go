package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RepositoryAllowlist parses a comma-separated owner/repo list. Repository
// names are case-insensitive on GitHub, so keys are normalized to lower case.
func RepositoryAllowlist(raw string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || strings.Count(value, "/") != 1 {
			continue
		}
		allowed[value] = struct{}{}
	}
	return allowed
}

// AllowedRepositories hides repository-scoped routes that are not part of this
// portal. Routes without owner/repo parameters (for example /health) pass
// through unchanged.
func AllowedRepositories(allowed map[string]struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		owner, repo := c.Param("owner"), c.Param("repo")
		if owner == "" && repo == "" {
			c.Next()
			return
		}
		_, ok := allowed[strings.ToLower(owner+"/"+repo)]
		if !ok {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "repository not found"})
			return
		}
		c.Next()
	}
}
