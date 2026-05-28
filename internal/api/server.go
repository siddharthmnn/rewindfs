package api

import (
	"net/http"

	"rewindfs/internal/models"

	"github.com/gin-gonic/gin"
)

var snapshots []models.Snapshot

func StartServer() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/snapshots", func(c *gin.Context) {
		c.JSON(http.StatusOK, snapshots)
	})

	r.POST("/snapshot", func(c *gin.Context) {

		var snapshot models.Snapshot

		if err := c.BindJSON(&snapshot); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		snapshots = append(snapshots, snapshot)

		c.JSON(http.StatusCreated, snapshot)
	})

	r.Run(":8080")
}
