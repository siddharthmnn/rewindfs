package api

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"rewindfs/internal/config"
	snapshotarchive "rewindfs/internal/archive"
)

var maxRequestBodyBytes int64 = config.DefaultRequestBodyLimit

var localHosts = map[string]struct{}{
	"127.0.0.1:8080": {},
	"localhost:8080": {},
}

func localHostOnly(c *gin.Context) {
	host := strings.ToLower(c.Request.Host)
	if _, ok := localHosts[host]; !ok {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "request host is not allowed"})
		return
	}
	c.Next()
}

func sameOrigin(c *gin.Context) {
	origin := c.GetHeader("Origin")
	if origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "http" ||
			!strings.EqualFold(parsed.Host, c.Request.Host) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-origin request is not allowed"})
			return
		}
	} else if strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-origin request is not allowed"})
		return
	}
	c.Next()
}

func limitRequestBody(c *gin.Context) {
	if c.Request.Body != nil {
		limit := maxRequestBodyBytes
		if c.Request.URL.Path == "/import" {
			limit = snapshotarchive.MaxArchiveBytes
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	}
	c.Next()
}
