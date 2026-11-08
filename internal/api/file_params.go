package api

import (
	"rewindfs/internal/workspace"

	"github.com/gin-gonic/gin"
)

func requestedFile(c *gin.Context) (string, error) {
	value := c.Query("file")
	if value == "" {
		value = c.Param("file")
	}
	if value == "" {
		value = c.Param("name")
	}
	if workspaceRoot == nil {
		return "", workspace.ErrInvalidPath
	}

	key, err := workspaceRoot.NormalizeKey(value)
	if err != nil {
		return "", err
	}
	if _, err := workspaceRoot.Resolve(key); err != nil {
		return "", err
	}
	return key, nil
}
