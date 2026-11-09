package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"rewindfs/internal/diff"
	"rewindfs/internal/models"
	"rewindfs/internal/snapshot"
	"rewindfs/internal/workspace"
)

const serverAddress = "127.0.0.1:8080"

func StartServer() {
	if err := Initialize(); err != nil {
		panic(err)
	}
	defer Close()

	if err := StartServerContext(context.Background()); err != nil {
		panic(err)
	}
}

// StartServerContext serves requests until the context is cancelled. The
// caller closes the watcher before the database so pending snapshots are kept.
func StartServerContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("server context must not be nil")
	}
	if err := Initialize(); err != nil {
		return err
	}

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	serverErrors := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	case err := <-serverErrors:
		return err
	}
}

func newRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), localHostOnly, sameOrigin, limitRequestBody)
	r.Static("/frontend", "./frontend")

	r.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	RegisterStatsRoutes(r)
	RegisterHistoryRoutes(r)
	RegisterLookupRoutes(r)

	r.GET("/diff/:id1/:id2", func(c *gin.Context) {
		id1, err := strconv.Atoi(c.Param("id1"))
		if err != nil || id1 <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}

		id2, err := strconv.Atoi(c.Param("id2"))
		if err != nil || id2 <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}

		snap1, found1 := snapshotService.Get(id1)
		snap2, found2 := snapshotService.Get(id2)
		if !found1 || !found2 {
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"snapshot1": id1,
			"snapshot2": id2,
			"same":      snap1.Content == snap2.Content,
			"diff":      diff.Compare(snap1.Content, snap2.Content),
			"file1":     snap1.File,
			"file2":     snap2.File,
			"content1":  snap1.Content,
			"content2":  snap2.Content,
			"length1":   len(snap1.Content),
			"length2":   len(snap2.Content),
		})
	})

	r.POST("/snapshot", func(c *gin.Context) {
		var request models.Snapshot
		if err := c.ShouldBindJSON(&request); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body is too large"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot payload"})
			return
		}

		created, err := AddSnapshot(&request)
		if err != nil {
			if errors.Is(err, snapshot.ErrInvalidFile) || errors.Is(err, workspace.ErrInvalidPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist snapshot"})
			return
		}

		status := http.StatusCreated
		if !created {
			status = http.StatusOK
		}
		c.JSON(status, request)
	})

	r.POST("/restore/:id", func(c *gin.Context) {
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

		if err := workspaceRoot.WriteFile(item.File, []byte(item.Content)); err != nil {
			if errors.Is(err, workspace.ErrInvalidPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot path is not safe to restore"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore snapshot"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "restored", "snapshot": id})
	})

	r.DELETE("/snapshot/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}

		if err := snapshotService.Delete(id); err != nil {
			if errors.Is(err, snapshot.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete snapshot"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "snapshot deleted"})
	})

	recoverFile := func(c *gin.Context) {
		file, err := requestedFile(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}

		history := snapshotService.ByFile(file)
		if len(history) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "no snapshot found"})
			return
		}

		latest := history[len(history)-1]
		if err := workspaceRoot.WriteFile(latest.File, []byte(latest.Content)); err != nil {
			if errors.Is(err, workspace.ErrInvalidPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot path is not safe to recover"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to recover file"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "file recovered", "snapshot": latest.ID})
	}

	r.POST("/recover", recoverFile)
	r.POST("/recover/:file", recoverFile)

	return r
}
