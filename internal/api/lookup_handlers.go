package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterLookupRoutes(r *gin.Engine) {

	r.GET("/snapshots/file/:name", func(c *gin.Context) {

		name := c.Param("name")
		results := SnapshotsByFile[name]

		c.JSON(http.StatusOK, results)

	})

	r.GET("/snapshot-count/:file", func(c *gin.Context) {

		filename := c.Param("file")

		count := len(SnapshotsByFile[filename])

		c.JSON(http.StatusOK, gin.H{
			"file":  filename,
			"count": count,
		})
	})
	r.GET("/snapshot/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid snapshot id",
			})
			return
		}

		snapshot, exists := SnapshotByID[id]

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "snapshot not found",
			})
			return
		}

		c.JSON(http.StatusOK, snapshot)

	})
}
