package api

import (
	"net/http"

	"rewindfs/internal/models"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/snapshots", func(c *gin.Context) {

		snapshots := []models.Snapshot{
			{
				ID:   1,
				File: "notes.txt",
			},
			{
				ID:   2,
				File: "todo.txt",
			},
		}

		c.JSON(http.StatusOK, snapshots)
	})

	r.Run(":8080")
}
