package api

import (
	"net/http"
	"os"
	"strconv"

	"rewindfs/internal/models"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	r := gin.Default()

	LoadSnapshots()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_snapshots": len(Snapshots),
		})
	})

	r.GET("/snapshots", func(c *gin.Context) {
		c.JSON(http.StatusOK, Snapshots)
	})

	r.GET("/snapshot/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid snapshot id",
			})
			return
		}

		for _, snapshot := range Snapshots {
			if snapshot.ID == id {
				c.JSON(http.StatusOK, snapshot)
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "snapshot not found",
		})
	})

	r.POST("/snapshot", func(c *gin.Context) {

		var snapshot models.Snapshot

		if err := c.BindJSON(&snapshot); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		AddSnapshot(snapshot)

		c.JSON(http.StatusCreated, snapshot)
	})

	r.POST("/restore/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid snapshot id",
			})
			return
		}

		for _, snapshot := range Snapshots {

			if snapshot.ID == id {

				err := os.WriteFile(
					snapshot.File,
					[]byte(snapshot.Content),
					0644,
				)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"message":  "restored",
					"snapshot": id,
				})

				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "snapshot not found",
		})
	})

	r.DELETE("/snapshot/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid snapshot id",
			})
			return
		}

		for i, snapshot := range Snapshots {

			if snapshot.ID == id {

				Snapshots = append(
					Snapshots[:i],
					Snapshots[i+1:]...,
				)

				SaveSnapshots()

				c.JSON(http.StatusOK, gin.H{
					"message": "snapshot deleted",
				})

				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "snapshot not found",
		})
	})

	r.Run(":8080")
}
