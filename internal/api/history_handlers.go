package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHistoryRoutes(r *gin.Engine) {
	latest := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		item, exists := snapshotService.Latest(file)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, item)
	}

	oldest := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		item, exists := snapshotService.Oldest(file)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, item)
	}

	history := func(full bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			file, err := requestedFile(c)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
				return
			}
			items := snapshotService.ByFile(file)
			if len(items) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				return
			}
			if full {
				c.JSON(http.StatusOK, items)
				return
			}

			result := make([]gin.H, 0, len(items))
			for _, item := range items {
				result = append(result, gin.H{
					"id":         item.ID,
					"created_at": item.CreatedAt,
				})
			}
			c.JSON(http.StatusOK, result)
		}
	}

	latestID := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		item, exists := snapshotService.Latest(file)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": item.File, "latest_id": item.ID})
	}

	firstID := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		item, exists := snapshotService.Oldest(file)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": item.File, "first_id": item.ID})
	}

	count := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"file":  file,
			"count": len(snapshotService.ByFile(file)),
		})
	}

	exists := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"exists": snapshotService.Exists(file)})
	}

	r.GET("/latest", latest)
	r.GET("/latest/:file", latest)
	r.GET("/snapshot-oldest", oldest)
	r.GET("/snapshot-oldest/:file", oldest)
	r.GET("/history", history(false))
	r.GET("/history/:file", history(false))
	r.GET("/history-full", history(true))
	r.GET("/history-full/:file", history(true))
	r.GET("/snapshot-latest-id", latestID)
	r.GET("/snapshot-latest-id/:file", latestID)
	r.GET("/snapshot-first-id", firstID)
	r.GET("/snapshot-first-id/:file", firstID)
	r.GET("/snapshot-count", count)
	r.GET("/snapshot-count/:file", count)
	r.GET("/files/exists", exists)
	r.GET("/files/:name/exists", exists)
}
