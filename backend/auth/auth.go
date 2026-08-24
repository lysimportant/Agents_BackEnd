package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"collector-backend/config"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SessionStore 持久化 Cookie 会话所需的最小数据集合。
type SessionStore interface {
	// CreateSession 表示登录会话。
	CreateSession(id string, userID int, expiresAt time.Time) error
	// FindSession 表示登录会话。
	FindSession(id string) (models.Session, bool)
	// DeleteSession 表示登录会话。
	DeleteSession(id string)
}

// Service 负责创建、校验和清除安全的 HttpOnly 登录会话。
type Service struct {
	// store 表示数据存储。
	store SessionStore
	// cookieName 表示名称。
	cookieName string
	// ttl 表示有效期。
	ttl time.Duration
	// sameSite 表示变量 sameSite。
	sameSite http.SameSite
	// secure 表示变量 secure。
	secure bool
}

// NewService 根据应用配置创建认证服务。
func NewService(store SessionStore, cfg config.Config) *Service {
	return &Service{
		store:      store,
		cookieName: cfg.SessionCookieName,
		ttl:        time.Duration(cfg.SessionTTLHours) * time.Hour,
		sameSite:   cfg.CookieSameSite,
		secure:     cfg.CookieSecure,
	}
}

// Create 为 userID 持久化随机会话，并返回会话 ID 和过期时间。
func (s *Service) Create(userID int) (string, time.Time, error) {
	// sessionID、err 保存当前操作结果以及可能返回的错误状态。
	sessionID, err := newSessionID()
	if err != nil {
		return "", time.Time{}, err
	}
	// expiresAt 保存变量 expiresAt。
	expiresAt := time.Now().Add(s.ttl)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.store.CreateSession(sessionID, userID, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return sessionID, expiresAt, nil
}

// UserIDFromRequest 从请求的会话 Cookie 中解析已登录用户 ID。
func (s *Service) UserIDFromRequest(c *gin.Context) (int, bool) {
	// sessionID、err 保存当前操作结果以及可能返回的错误状态。
	sessionID, err := c.Cookie(s.cookieName)
	if err != nil || sessionID == "" {
		return 0, false
	}
	// session、exists 保存业务值及其是否存在或处理成功的标记。
	session, exists := s.store.FindSession(sessionID)
	if !exists {
		return 0, false
	}
	return session.UserID, true
}

// Delete 根据会话 ID 使持久化会话失效。
func (s *Service) Delete(sessionID string) {
	s.store.DeleteSession(sessionID)
}

// DeleteFromRequest 在当前请求存在会话 Cookie 时使该会话失效。
func (s *Service) DeleteFromRequest(c *gin.Context) {
	// sessionID、err 保存当前操作结果以及可能返回的错误状态。
	if sessionID, err := c.Cookie(s.cookieName); err == nil && sessionID != "" {
		s.Delete(sessionID)
	}
}

// HashPassword 返回适合写入数据库的 bcrypt 密码哈希。
func HashPassword(password string) (string, error) {
	// hash、err 保存当前操作结果以及可能返回的错误状态。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// MustHashPassword 生成密码哈希；仅在 bcrypt 自身失败时触发 panic。
func MustHashPassword(password string) string {
	// hash、err 保存当前操作结果以及可能返回的错误状态。
	hash, err := HashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

// ComparePassword 判断明文密码是否与已存储的 bcrypt 哈希匹配。
func ComparePassword(passwordHash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

// ToAuthUser 移除敏感字段后生成可通过 API 返回的登录用户信息。
func ToAuthUser(user models.User, actionPermissions []string) models.AuthUser {
	if actionPermissions == nil {
		actionPermissions = []string{}
	}
	return models.AuthUser{
		ID:                user.ID,
		Username:          user.Username,
		Name:              user.Name,
		RoleID:            user.RoleID,
		Role:              user.Role,
		RoleCode:          user.RoleCode,
		DepartmentID:      user.DepartmentID,
		Department:        user.Department,
		Status:            user.Status,
		Phone:             user.Phone,
		Email:             user.Email,
		Age:               user.Age,
		Description:       user.Description,
		AvatarURL:         user.AvatarURL,
		CanLogin:          user.CanLogin,
		ActionPermissions: actionPermissions,
	}
}

// SetSessionCookie 写入按配置生成的安全 HttpOnly 会话 Cookie。
func (s *Service) SetSessionCookie(c *gin.Context, sessionID string, expiresAt time.Time) {
	// maxAge 保存年龄。
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(s.sameSite)
	c.SetCookie(s.cookieName, sessionID, maxAge, "/", "", s.secure, true)
}

// ClearSessionCookie 使浏览器中的会话 Cookie 立即过期。
func (s *Service) ClearSessionCookie(c *gin.Context) {
	c.SetSameSite(s.sameSite)
	c.SetCookie(s.cookieName, "", -1, "/", "", s.secure, true)
}

// SetPortalR18Cookie 写入由后端域管理的 18R 可见性偏好，供公开接口与登录会话共同校验。
func (s *Service) SetPortalR18Cookie(c *gin.Context, enabled bool) {
	// value 保存后端识别的 18R 偏好值。
	value := "0"
	if enabled {
		value = "1"
	}
	c.SetSameSite(s.sameSite)
	c.SetCookie("portal-r18", value, int((365 * 24 * time.Hour).Seconds()), "/", "", s.secure, true)
}

// ClearPortalR18Cookie 清除当前后端域中的 18R 可见性偏好。
func (s *Service) ClearPortalR18Cookie(c *gin.Context) {
	c.SetSameSite(s.sameSite)
	c.SetCookie("portal-r18", "", -1, "/", "", s.secure, true)
}

// newSessionID 构造并返回对应业务实例。
func newSessionID() (string, error) {
	// bytes 保存字节数。
	bytes := make([]byte, 32)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
