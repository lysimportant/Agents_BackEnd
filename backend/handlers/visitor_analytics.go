package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type VisitorAnalyticsStore interface {
	ListVisitorAnalytics(filter models.VisitorAnalyticsFilter) (models.VisitorAnalyticsResponse, error)
}

type VisitorAnalyticsHandler struct {
	store VisitorAnalyticsStore
}

func NewVisitorAnalyticsHandler(store VisitorAnalyticsStore) *VisitorAnalyticsHandler {
	return &VisitorAnalyticsHandler{store: store}
}

func (h *VisitorAnalyticsHandler) List(c *gin.Context) {
	filter, err := parseVisitorAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.store.ListVisitorAnalytics(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载访问分析失败"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func parseVisitorAnalyticsFilter(c *gin.Context) (models.VisitorAnalyticsFilter, error) {
	rangeValue := strings.TrimSpace(strings.ToLower(c.DefaultQuery("range", "7d")))
	var duration time.Duration
	switch rangeValue {
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		return models.VisitorAnalyticsFilter{}, &visitorAnalyticsRangeError{value: rangeValue}
	}
	page := queryPositiveInt(c, "page", 1, 1, 100000)
	pageSize := queryPositiveInt(c, "pageSize", 10, 10, 100)
	filter := models.VisitorAnalyticsFilter{
		From:     time.Now().UTC().Add(-duration),
		To:       time.Now().UTC().Add(time.Second),
		Range:    rangeValue,
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	}
	if statusRaw := strings.TrimSpace(c.Query("statusCode")); statusRaw != "" {
		status, err := strconv.Atoi(statusRaw)
		if err != nil || status < 100 || status > 599 {
			return models.VisitorAnalyticsFilter{}, &visitorAnalyticsStatusError{}
		}
		filter.StatusCode = &status
	}
	return filter, nil
}

func queryPositiveInt(c *gin.Context, key string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || value < minimum || value > maximum {
		return fallback
	}
	return value
}

type visitorAnalyticsRangeError struct{ value string }

func (e *visitorAnalyticsRangeError) Error() string { return "时间范围只能是 24h、7d 或 30d" }

type visitorAnalyticsStatusError struct{}

func (*visitorAnalyticsStatusError) Error() string {
	return "状态码必须是 100 到 599 之间的数字"
}
