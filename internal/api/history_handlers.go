package api

import (
        "net/http"


       "github.com/gin-gonic/gin"
)
func RegisterHistoryRoutes(r *gin.Engine) {

        r.GET("/latest/:file", func(c *gin.Context) {

                filename := c.Param("file")

		snapshots := SnapshotsByFile[filename]

		if len(snapshots) == 0 {
        		c.JSON(http.StatusNotFound, gin.H{
                		"error": "file not found",
        		})
        		return
		}

		c.JSON(http.StatusOK, snapshots[len(snapshots)-1])
		return


                c.JSON(http.StatusNotFound, gin.H{
                        "error": "file not found",
                })
        })

r.GET("/snapshot-oldest/:file", func(c *gin.Context) {

                filename := c.Param("file")

		snapshots := SnapshotsByFile[filename]

		if len(snapshots) == 0 {
        		c.JSON(http.StatusNotFound, gin.H{
                		"error": "file not found",
        		})
        		return
		}

		c.JSON(http.StatusOK, snapshots[0])
		return


                c.JSON(http.StatusNotFound, gin.H{
                        "error": "file not found",
                })
        })
        r.GET("/history/:file", func(c *gin.Context) {

                filename := c.Param("file")

                var history []gin.H

                for _, snapshot := range Snapshots {

                        if snapshot.File == filename {

                                history = append(history, gin.H{
                                        "id":         snapshot.ID,
                                        "created_at": snapshot.CreatedAt,
                                })
                        }
                }

                if len(history) == 0 {
                        c.JSON(http.StatusNotFound, gin.H{
                                "error": "file not found",
                        })
                        return
                }

                c.JSON(http.StatusOK, history)
        })

        r.GET("/history-full/:file", func(c *gin.Context) {

                filename := c.Param("file")

		history := SnapshotsByFile[filename]

		if len(history) == 0 {

                        c.JSON(http.StatusNotFound, gin.H{
                                "error": "file not found",
                        })
                        return
                }

                c.JSON(http.StatusOK, history)
        })

        r.GET("/files/:name/exists", func(c *gin.Context) {

                filename := c.Param("name")


		_, exists := SnapshotsByFile[filename]

		c.JSON(http.StatusOK, gin.H{
        		"exists": exists,
		})


        })

        r.GET("/snapshot-latest-id/:file", func(c *gin.Context) {

                filename := c.Param("file")

		snapshots := SnapshotsByFile[filename]

		if len(snapshots) == 0 {
        		c.JSON(http.StatusNotFound, gin.H{
                		"error": "file not found",
        		})
        		return
		}

		latestID := snapshots[len(snapshots)-1].ID

                c.JSON(http.StatusOK, gin.H{
                        "file":      filename,
                        "latest_id": latestID,
                })
        })

        r.GET("/snapshot-first-id/:file", func(c *gin.Context) {

                filename := c.Param("file")

		snapshots := SnapshotsByFile[filename]

		if len(snapshots) == 0 {
        		c.JSON(http.StatusNotFound, gin.H{
                		"error": "file not found",
        		})
        		return
		}

		c.JSON(http.StatusOK, gin.H{
        		"file":     filename,
        		"first_id": snapshots[0].ID,
		})
		return


        })
}
