package utils

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return 0, false
	}
	return id, true
}
