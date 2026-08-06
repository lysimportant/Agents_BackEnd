package utils

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseID 解析对应业务数据。
func ParseID(c *gin.Context) (int, bool) {
	// id、err 保存当前操作结果以及可能返回的错误状态。
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return 0, false
	}
	return id, true
}
