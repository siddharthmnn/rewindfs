package api

import (
        "net/http"

	"rewindfs/internal/models" 

       "github.com/gin-gonic/gin"
)
func RegisterHistoryRoutes(r *gin.Engine) {

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

                var history []models.Snapshot

                for _, snapshot := range Snapshots {

                        if snapshot.File == filename {
                                history = append(history, snapshot)
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

        r.GET("/files/:name/exists", func(c *gin.Context) {

                filename := c.Param("name")

                for _, snapshot := range Snapshots {

                        if snapshot.File == filename {

                                c.JSON(http.StatusOK, gin.H{
                                        "exists": true,
                                })
                                return
                        }
                }

                c.JSON(http.StatusOK, gin.H{
                        "exists": false,
                })
        })

        r.GET("/snapshot-latest-id/:file", func(c *gin.Context) {

                filename := c.Param("file")

                latestID := -1

                for _, snapshot := range Snapshots {

                        if snapshot.File == filename {
                                latestID = snapshot.ID
                        }
                }

                if latestID == -1 {
                        c.JSON(http.StatusNotFound, gin.H{
                                "error": "file not found",
                        })
                        return
                }

                c.JSON(http.StatusOK, gin.H{
                        "file":      filename,
                        "latest_id": latestID,
                })
        })

        r.GET("/snapshot-first-id/:file", func(c *gin.Context) {

                filename := c.Param("file")

                for _, snapshot := range Snapshots {

                        if snapshot.File == filename {

                                c.JSON(http.StatusOK, gin.H{
                                        "file":     filename,
                                        "first_id": snapshot.ID,
                                })
                                return
                        }
                }

                c.JSON(http.StatusNotFound, gin.H{
                        "error": "file not found",
                })
        })
}
