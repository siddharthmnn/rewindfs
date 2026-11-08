package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterLookupRoutes(r *gin.Engine) {
	listFile := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		c.JSON(http.StatusOK, snapshotService.ByFile(file))
	}

	r.GET("/snapshots/file", listFile)
	r.GET("/snapshots/file/:name", listFile)

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
