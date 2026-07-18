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

const userContextKey = "currentUser"

type UserStore interface {
	FindUserByID(id int) (models.User, bool)
	ListUserMenus(userID int) ([]models.Menu, string)
	ListUserActionPermissions(userID int) ([]string, string)
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	allowAnyOrigin := false
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
		origin := strings.TrimSpace(c.GetHeader("Origin"))
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

func RequireAuth(userStore UserStore, sessionService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := sessionService.UserIDFromRequest(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			c.Abort()
			return
		}

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

func CurrentUser(c *gin.Context) (models.User, bool) {
	value, exists := c.Get(userContextKey)
	if !exists {
		return models.User{}, false
	}
	user, ok := value.(models.User)
	return user, ok
}

func RequireMenu(userStore UserStore, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		menus, message := userStore.ListUserMenus(user.ID)
		if message != "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": message})
			return
		}
		for _, menu := range menus {
			if menu.Code == code && menu.Status == "启用" {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问该功能"})
	}
}

func RequireAction(userStore UserStore, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := CurrentUser(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
			return
		}
		if !permissions.IsKnown(code) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "动作权限配置错误"})
			return
		}
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

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
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
