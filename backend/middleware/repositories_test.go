package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowedRepositories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AllowedRepositories(RepositoryAllowlist("powerlive-io/bili-shadowreplay, powerlive-io/JLiverTool, powerlive-io/bili-messenger")))
	router.GET("/api/release/:owner/:repo", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/api/release/powerlive-io/JLiverTool", want: http.StatusNoContent},
		{path: "/api/release/POWERLIVE-IO/jlivertool", want: http.StatusNoContent},
		{path: "/api/release/powerlive-io/bili-shadowreplay", want: http.StatusNoContent},
		{path: "/api/release/powerlive-io/bili-messenger", want: http.StatusNoContent},
		{path: "/api/release/powerlive-io/JoiAsk", want: http.StatusNotFound},
		{path: "/api/release/Xinrea/JLiverTool", want: http.StatusNotFound},
		{path: "/api/release/other/project", want: http.StatusNotFound},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("%s: status = %d, want %d", test.path, response.Code, test.want)
		}
	}
}
