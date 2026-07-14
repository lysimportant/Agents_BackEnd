package data_point_handlers

import (
	"net/http"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type Store interface {
	ListDataPoints() []models.DataPoint
	CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *gin.Context) {
	dataPoints := h.store.ListDataPoints()
	if dataPoints == nil {
		dataPoints = []models.DataPoint{}
	}
	c.JSON(http.StatusOK, dataPoints)
}

func (h *Handler) Create(c *gin.Context) {
	var request models.CreateDataPointRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.store.CreateDataPoint(request))
}
