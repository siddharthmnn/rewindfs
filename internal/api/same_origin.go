package api

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func sameOrigin(c *gin.Context) {
	origin := c.GetHeader("Origin")
	if origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "http" ||
			!strings.EqualFold(parsed.Host, c.Request.Host) {
			c.AbortWithStatusJSON(403, gin.H{"error": "cross-origin request is not allowed"})
			return
		}
	} else if strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
		c.AbortWithStatusJSON(403, gin.H{"error": "cross-origin request is not allowed"})
		return
	}

	c.Next()
}
