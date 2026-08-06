package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// VisitorAnalyticsStore 定义对应业务的数据结构与调用契约。
type VisitorAnalyticsStore interface {
	// ListVisitorAnalytics 表示列表访问者分析数据。
	ListVisitorAnalytics(filter models.VisitorAnalyticsFilter) (models.VisitorAnalyticsResponse, error)
}

// VisitorAnalyticsHandler 定义对应业务的数据结构与调用契约。
type VisitorAnalyticsHandler struct {
	// store 表示数据存储。
	store VisitorAnalyticsStore
}

// NewVisitorAnalyticsHandler 构造并返回对应业务实例。
func NewVisitorAnalyticsHandler(store VisitorAnalyticsStore) *VisitorAnalyticsHandler {
	return &VisitorAnalyticsHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *VisitorAnalyticsHandler) List(c *gin.Context) {
	// filter、err 保存当前操作结果以及可能返回的错误状态。
	filter, err := parseVisitorAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := h.store.ListVisitorAnalytics(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载访问分析失败"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// parseVisitorAnalyticsFilter 解析对应业务数据。
func parseVisitorAnalyticsFilter(c *gin.Context) (models.VisitorAnalyticsFilter, error) {
	// rangeValue 保存值。
	rangeValue := strings.TrimSpace(strings.ToLower(c.DefaultQuery("range", "7d")))
	// duration 保存耗时。
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
	// page 保存页码。
	page := queryPositiveInt(c, "page", 1, 1, 100000)
	// pageSize 保存页码大小。
	pageSize := queryPositiveInt(c, "pageSize", 10, 10, 100)
	// filter 保存筛选条件。
	filter := models.VisitorAnalyticsFilter{
		From:     time.Now().UTC().Add(-duration),
		To:       time.Now().UTC().Add(time.Second),
		Range:    rangeValue,
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	}
	// statusRaw 保存状态。
	if statusRaw := strings.TrimSpace(c.Query("statusCode")); statusRaw != "" {
		// status、err 保存当前操作结果以及可能返回的错误状态。
		status, err := strconv.Atoi(statusRaw)
		if err != nil || status < 100 || status > 599 {
			return models.VisitorAnalyticsFilter{}, &visitorAnalyticsStatusError{}
		}
		filter.StatusCode = &status
	}
	return filter, nil
}

// queryPositiveInt 查询并返回对应业务列表。
func queryPositiveInt(c *gin.Context, key string, fallback, minimum, maximum int) int {
	// value、err 保存当前操作结果以及可能返回的错误状态。
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || value < minimum || value > maximum {
		return fallback
	}
	return value
}

// visitorAnalyticsRangeError 定义对应业务的数据结构与调用契约。
type visitorAnalyticsRangeError struct{ value string }

// Error 实现对应业务逻辑。
func (e *visitorAnalyticsRangeError) Error() string { return "时间范围只能是 24h、7d 或 30d" }

// visitorAnalyticsStatusError 定义对应业务的数据结构与调用契约。
type visitorAnalyticsStatusError struct{}

// Error 实现对应业务逻辑。
func (*visitorAnalyticsStatusError) Error() string {
	return "状态码必须是 100 到 599 之间的数字"
}
