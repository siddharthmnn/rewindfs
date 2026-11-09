package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterStatsRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/stats", func(c *gin.Context) {
		stats := snapshotService.Stats()
		c.JSON(http.StatusOK, gin.H{
			"total_snapshots": stats.TotalSnapshots,
			"tracked_files":   stats.TrackedFiles,
			"unique_hashes":   stats.UniqueHashes,
		})
	})

	r.GET("/snapshots", func(c *gin.Context) {
		c.JSON(http.StatusOK, snapshotService.All())
	})

	r.GET("/snapshots/search", func(c *gin.Context) {
		limit := 25
		offset := 0
		if raw := c.Query("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 100 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 100"})
				return
			}
			limit = value
		}
		if raw := c.Query("offset"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be zero or greater"})
				return
			}
			offset = value
		}

		items, total := snapshotService.Search(c.Query("q"), offset, limit)
		c.JSON(http.StatusOK, gin.H{
			"items":  items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	})

	r.GET("/files", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"files": snapshotService.Files()})
	})
}
