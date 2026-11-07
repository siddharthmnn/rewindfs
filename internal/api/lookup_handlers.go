package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterLookupRoutes(r *gin.Engine) {
	r.GET("/snapshots/file/:name", func(c *gin.Context) {
		c.JSON(http.StatusOK, snapshotService.ByFile(c.Param("name")))
	})

	r.GET("/snapshot-count/:file", func(c *gin.Context) {
		filename := c.Param("file")
		c.JSON(http.StatusOK, gin.H{
			"file":  filename,
			"count": len(snapshotService.ByFile(filename)),
		})
	})

	r.GET("/snapshot/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}

		item, exists := snapshotService.Get(id)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}
		c.JSON(http.StatusOK, item)
	})
}
