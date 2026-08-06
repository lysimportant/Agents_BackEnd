package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// VisitorAccessStore 定义对应业务的数据结构与调用契约。
type VisitorAccessStore interface {
	// RecordVisitorAccess 表示记录访问者。
	RecordVisitorAccess(models.VisitorAccessRecord) error
	// PruneVisitorAccessBefore 表示访问者。
	PruneVisitorAccessBefore(time.Time) error
}

// VisitorAccessLogger 记录请求元数据，但不读取 Cookie、查询字符串或请求体。
// IP 地理信息由可信反向代理提供；代理未提供时相应字段保持为空。
func VisitorAccessLogger(store VisitorAccessStore, retentionDays int) gin.HandlerFunc {
	// lastCleanup 保存变量 lastCleanup。
	var lastCleanup atomic.Int64
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		// started 保存启动状态。
		started := time.Now()
		c.Next()

		// remoteIP 保存变量 remoteIP。
		remoteIP := requestRemoteIP(c.Request.RemoteAddr)
		// forwardedIP 保存变量 forwardedIP。
		forwardedIP := firstForwardedIP(c.GetHeader("X-Forwarded-For"))
		// user、authenticated 保存用户、变量 authenticated。
		user, authenticated := CurrentUser(c)
		// browser、operatingSystem、device 保存浏览器、变量 operatingSystem、设备。
		browser, operatingSystem, device := classifyUserAgent(c.GetHeader("User-Agent"))
		// record 保存记录。
		record := models.VisitorAccessRecord{
			IP:             limitText(remoteIP, 64),
			ForwardedIP:    limitText(forwardedIP, 64),
			Country:        firstHeader(c, "CF-IPCountry", "X-Country", "X-Geo-Country", "X-Vercel-IP-Country"),
			Region:         firstHeader(c, "X-Region", "X-Geo-Region"),
			City:           firstHeader(c, "X-City", "X-Geo-City"),
			ISP:            firstHeader(c, "X-ISP", "X-Geo-ISP"),
			Host:           limitText(c.Request.Host, 255),
			Method:         limitText(c.Request.Method, 16),
			Path:           limitText(c.Request.URL.Path, 1024),
			StatusCode:     c.Writer.Status(),
			DurationMS:     time.Since(started).Milliseconds(),
			Bytes:          int64(maxInt(c.Writer.Size(), 0)),
			UserAgent:      limitText(c.GetHeader("User-Agent"), 1024),
			Browser:        browser,
			OS:             operatingSystem,
			Device:         device,
			Referer:        limitText(c.GetHeader("Referer"), 1024),
			AcceptLanguage: limitText(c.GetHeader("Accept-Language"), 255),
			Authenticated:  authenticated,
			CreatedAt:      time.Now().UTC(),
		}
		if authenticated {
			record.UserID = &user.ID
			record.UserName = limitText(user.Name, 255)
		}
		_ = store.RecordVisitorAccess(record)

		if retentionDays > 0 {
			// now 保存当前时间。
			now := time.Now().Unix()
			// last 保存变量 last。
			last := lastCleanup.Load()
			if now-last >= int64(time.Hour/time.Second) && lastCleanup.CompareAndSwap(last, now) {
				_ = store.PruneVisitorAccessBefore(time.Now().UTC().AddDate(0, 0, -retentionDays))
			}
		}
	}
}

// requestRemoteIP 实现对应业务逻辑。
func requestRemoteIP(remoteAddr string) string {
	// host、err 保存当前操作结果以及可能返回的错误状态。
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

// firstForwardedIP 实现对应业务逻辑。
func firstForwardedIP(value string) string {
	// candidate 表示当前循环中的索引、键或业务元素。
	for _, candidate := range strings.Split(value, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// firstHeader 实现对应业务逻辑。
func firstHeader(c *gin.Context, names ...string) string {
	// name 表示当前循环中的索引、键或业务元素。
	for _, name := range names {
		// value 保存值。
		if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
			return limitText(value, 128)
		}
	}
	return ""
}

// limitText 实现对应业务逻辑。
func limitText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

// maxInt 实现对应业务逻辑。
func maxInt(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

// classifyUserAgent 实现对应业务逻辑。
func classifyUserAgent(value string) (browser, operatingSystem, device string) {
	// ua 保存变量 ua。
	ua := strings.ToLower(value)
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browser = "Opera"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	case strings.Contains(ua, "micromessenger"):
		browser = "微信"
	default:
		browser = "其它"
	}
	switch {
	case strings.Contains(ua, "windows"):
		operatingSystem = "Windows"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios"):
		operatingSystem = "iOS"
	case strings.Contains(ua, "android"):
		operatingSystem = "Android"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		operatingSystem = "macOS"
	case strings.Contains(ua, "linux"):
		operatingSystem = "Linux"
	default:
		operatingSystem = "其它"
	}
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		device = "平板"
	} else if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		device = "手机"
	} else {
		device = "桌面端"
	}
	return browser, operatingSystem, device
}
