package handlers

import (
	"net/http"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type DataPointStore interface {
	ListDataPoints() []models.DataPoint
	CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint
}

type DataPointHandler struct {
	store DataPointStore
}

func NewDataPointHandler(store DataPointStore) *DataPointHandler {
	return &DataPointHandler{store: store}
}

func (h *DataPointHandler) List(c *gin.Context) {
	dataPoints := h.store.ListDataPoints()
	if dataPoints == nil {
		dataPoints = []models.DataPoint{}
	}
	c.JSON(http.StatusOK, dataPoints)
}

func (h *DataPointHandler) Create(c *gin.Context) {
	var request models.CreateDataPointRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.store.CreateDataPoint(request))
}
