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

        	filesMap := make(map[string]bool)

        	for _, snapshot := range Snapshots {
                	filesMap[snapshot.File] = true
        	}

        	c.JSON(http.StatusOK, gin.H{
                	"total_snapshots": len(Snapshots),
                	"tracked_files":  len(filesMap),
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

        	var files []string

        	for file := range filesMap {
                	files = append(files, file)
        	}
	
        	c.JSON(http.StatusOK, gin.H{
                	"files": files,
        	})
	})
	r.GET("/snapshots/file/:name", func(c *gin.Context) {

  	      name := c.Param("name")

        	var results []models.Snapshot

        	for _, snapshot := range Snapshots {

                	if snapshot.File == name {
                        	results = append(results, snapshot)
                	}
        	}

        	c.JSON(http.StatusOK, results)
	})
	r.GET("/snapshot-count/:file", func(c *gin.Context) {

        	filename := c.Param("file")

        	count := 0

        	for _, snapshot := range Snapshots {
                	if snapshot.File == filename {
                        	count++
                	}
        	}

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
		r.GET("/diff/:id1/:id2", func(c *gin.Context) {

        	id1, err := strconv.Atoi(c.Param("id1"))
        	if err != nil {
                	c.JSON(http.StatusBadRequest, gin.H{
                        	"error": "invalid snapshot id",
                	})
                	return
        	}

        	id2, err := strconv.Atoi(c.Param("id2"))
        	if err != nil {
                	c.JSON(http.StatusBadRequest, gin.H{
                	        "error": "invalid snapshot id",
                	})
                	return
        	}

        	var snap1 models.Snapshot
        	var snap2 models.Snapshot

        	found1 := false
        	found2 := false

        	for _, snapshot := range Snapshots {

                	if snapshot.ID == id1 {
                        	snap1 = snapshot
                        	found1 = true
                	}

                	if snapshot.ID == id2 {
                	        snap2 = snapshot
                	        found2 = true
                	}
        	}

        	if !found1 || !found2 {
                	c.JSON(http.StatusNotFound, gin.H{
                        	"error": "snapshot not found",
                	})
        	        return
        	}

        	c.JSON(http.StatusOK, gin.H{
                	"snapshot1": id1,
                	"snapshot2": id2,
                	"same":      snap1.Content == snap2.Content,
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
	r.POST("/recover/:file", func(c *gin.Context) {

  	      filename := c.Param("file")

        	for i := len(Snapshots) - 1; i >= 0; i-- {

                	if Snapshots[i].File == filename {

                        	err := os.WriteFile(
                                	filename,
                                	[]byte(Snapshots[i].Content),
                                	0644,
                        	)

                        	if err != nil {
                                	c.JSON(http.StatusInternalServerError, gin.H{
                                        	"error": err.Error(),
                                	})
                                	return
                        	}

                        	c.JSON(http.StatusOK, gin.H{
                                	"message": "file recovered",
                        	})

                        	return
                	}
        	}

        	c.JSON(http.StatusNotFound, gin.H{
                	"error": "no snapshot found",
        	})
		})

		r.GET("/latest/:file", func(c *gin.Context) {

        	filename := c.Param("file")

        	for i := len(Snapshots) - 1; i >= 0; i-- {

                	if Snapshots[i].File == filename {

                        	c.JSON(http.StatusOK, Snapshots[i])
                        	return
                	}
        	}

        	c.JSON(http.StatusNotFound, gin.H{
        	        "error": "file not found",
        	})
	})
	r.GET("/snapshot-oldest/:file", func(c *gin.Context) {

	        filename := c.Param("file")

        	for _, snapshot := range Snapshots {

                	if snapshot.File == filename {

                        	c.JSON(http.StatusOK, snapshot)
                        	return
                	}
        	}

        	c.JSON(http.StatusNotFound, gin.H{
                	"error": "file not found",
        	})
	})
	r.Run(":8080")
}
