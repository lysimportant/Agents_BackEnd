package middleware

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// userContextKey 保存模块使用的固定配置或共享状态。
const userContextKey = "currentUser"

// UserStore 定义对应业务的数据结构与调用契约。
type UserStore interface {
	// FindUserByID 表示用户标识。
	FindUserByID(id int) (models.User, bool)
	// ListUserMenus 表示列表用户。
	ListUserMenus(userID int) ([]models.Menu, string)
	// ListUserActionPermissions 表示列表用户。
	ListUserActionPermissions(userID int) ([]string, string)
}

// CORS 实现对应业务逻辑。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	// allowed 保存允许范围。
	allowed := make(map[string]bool, len(allowedOrigins))
	// allowAnyOrigin 保存请求来源。
	allowAnyOrigin := false
	// origin 表示当前循环中的索引、键或业务元素。
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAnyOrigin = true
			continue
		}
		if origin != "" {
			allowed[origin] = true
		}
	}

	return func(c *gin.Context) {
		// origin 保存请求来源。
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		// originAllowed 保存请求来源允许范围。
		originAllowed := origin != "" && (allowAnyOrigin || allowed[origin])
		if origin != "" {
			if !originAllowed {
				if c.Request.Method == http.MethodOptions {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Socket-Visitor-Token")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !originAllowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequireAuth 校验对应业务条件。
func RequireAuth(userStore UserStore, sessionService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// userID、ok 保存业务值及其是否存在或处理成功的标记。
		userID, ok := sessionService.UserIDFromRequest(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			c.Abort()
			return
		}

		// user、found 保存业务值及其是否存在或处理成功的标记。
		user, found := userStore.FindUserByID(userID)
		if !found || !user.LoginAllowed() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			c.Abort()
			return
		}

		c.Set(userContextKey, user)
		c.Next()
	}
}

// CurrentUser 实现对应业务逻辑。
func CurrentUser(c *gin.Context) (models.User, bool) {
	// value、exists 保存业务值及其是否存在或处理成功的标记。
	value, exists := c.Get(userContextKey)
	if !exists {
		return models.User{}, false
	}
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := value.(models.User)
	return user, ok
}

// RequireMenu 校验对应业务条件。
func RequireMenu(userStore UserStore, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// user、exists 保存业务值及其是否存在或处理成功的标记。
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		// menus、message 保存菜单、消息。
		menus, message := userStore.ListUserMenus(user.ID)
		if message != "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": message})
			return
		}
		// menu 表示当前循环中的索引、键或业务元素。
		for _, menu := range menus {
			if menu.Code == code && menu.Status == "启用" {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问该功能"})
	}
}

// RequireAction 校验对应业务条件。
func RequireAction(userStore UserStore, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// user、exists 保存业务值及其是否存在或处理成功的标记。
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		if !permissions.IsKnown(code) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "动作权限配置错误"})
			return
		}
		// codes、message 保存编码、消息。
		codes, message := userStore.ListUserActionPermissions(user.ID)
		if message != "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": message})
			return
		}
		if permissions.Contains(codes, code) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权执行该操作"})
	}
}

// RequireAdmin 校验对应业务条件。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// user、exists 保存业务值及其是否存在或处理成功的标记。
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		if !utils.IsAdmin(user) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "仅超级管理员或系统管理员可执行此操作"})
			return
		}
		c.Next()
	}
}

// RequireSuperAdmin 仅允许稳定角色编码为 super-admin 的用户继续访问。
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// user、exists 表示当前会话用户及其是否已由认证中间件写入上下文。
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		if !utils.IsSuperAdmin(user) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "仅超级管理员可使用服务器终端"})
			return
		}
		c.Next()
	}
}
