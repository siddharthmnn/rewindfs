package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHistoryRoutes(r *gin.Engine) {
	r.GET("/latest/:file", func(c *gin.Context) {
		item, exists := snapshotService.Latest(c.Param("file"))
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, item)
	})

	r.GET("/snapshot-oldest/:file", func(c *gin.Context) {
		item, exists := snapshotService.Oldest(c.Param("file"))
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, item)
	})

	r.GET("/history/:file", func(c *gin.Context) {
		history := snapshotService.ByFile(c.Param("file"))
		if len(history) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}

		result := make([]gin.H, 0, len(history))
		for _, item := range history {
			result = append(result, gin.H{
				"id":         item.ID,
				"created_at": item.CreatedAt,
			})
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/history-full/:file", func(c *gin.Context) {
		history := snapshotService.ByFile(c.Param("file"))
		if len(history) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, history)
	})

	r.GET("/files/:name/exists", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"exists": snapshotService.Exists(c.Param("name")),
		})
	})

	r.GET("/snapshot-latest-id/:file", func(c *gin.Context) {
		item, exists := snapshotService.Latest(c.Param("file"))
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"file":      item.File,
			"latest_id": item.ID,
		})
	})

	r.GET("/snapshot-first-id/:file", func(c *gin.Context) {
		item, exists := snapshotService.Oldest(c.Param("file"))
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"file":     item.File,
			"first_id": item.ID,
		})
	})
}
