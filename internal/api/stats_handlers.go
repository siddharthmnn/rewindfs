package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterStatsRoutes(r *gin.Engine) {

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/stats", func(c *gin.Context) {

		filesMap := make(map[string]bool)
		hashMap := make(map[string]bool)

		for _, snapshot := range Snapshots {
			filesMap[snapshot.File] = true
			hashMap[snapshot.Hash] = true
		}

		c.JSON(http.StatusOK, gin.H{
			"total_snapshots": len(Snapshots),
			"tracked_files":   len(filesMap),
			"unique_hashes":   len(hashMap),
		})
	})

	r.GET("/snapshots", func(c *gin.Context) {
		c.JSON(http.StatusOK, Snapshots)
	})

	r.GET("/files", func(c *gin.Context) {

		filesMap := make(map[string]bool)

		for _, snapshot := range Snapshots {
			filesMap[snapshot.File] = true
		}

		files := []string{}

		for file := range filesMap {
			files = append(files, file)
		}

		c.JSON(http.StatusOK, gin.H{
			"files": files,
		})
	})
}
