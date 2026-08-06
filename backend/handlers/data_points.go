package handlers

import (
	"net/http"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// DataPointStore 定义对应业务的数据结构与调用契约。
type DataPointStore interface {
	// ListDataPoints 表示列表业务数据。
	ListDataPoints() []models.DataPoint
	// CreateDataPoint 表示业务数据。
	CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint
}

// DataPointHandler 定义对应业务的数据结构与调用契约。
type DataPointHandler struct {
	// store 表示数据存储。
	store DataPointStore
}

// NewDataPointHandler 构造并返回对应业务实例。
func NewDataPointHandler(store DataPointStore) *DataPointHandler {
	return &DataPointHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *DataPointHandler) List(c *gin.Context) {
	// dataPoints 保存业务数据。
	dataPoints := h.store.ListDataPoints()
	if dataPoints == nil {
		dataPoints = []models.DataPoint{}
	}
	c.JSON(http.StatusOK, dataPoints)
}

// Create 创建或追加对应业务记录。
func (h *DataPointHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.CreateDataPointRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.store.CreateDataPoint(request))
}
