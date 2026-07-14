package middleware

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

const userContextKey = "currentUser"

type UserStore interface {
	FindUserByID(id int) (models.User, bool)
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if !allowed[origin] {
				if c.Request.Method == http.MethodOptions {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !allowed[origin] {
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
		if !found || !user.CanLogin {
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
