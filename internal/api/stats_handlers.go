package api

import (
	"net/http"

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

	r.GET("/files", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"files": snapshotService.Files(),
		})
	})
}
