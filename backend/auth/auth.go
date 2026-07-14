package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"collector-backend/config"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	FindUserByID(id int) (models.User, bool)
	FindUserByUsername(username string) (models.User, bool)
}

type SessionStore interface {
	CreateSession(id string, userID int, expiresAt time.Time) error
	FindSession(id string) (models.Session, bool)
	DeleteSession(id string)
}

type Service struct {
	store      SessionStore
	cookieName string
	ttl        time.Duration
	sameSite   http.SameSite
	secure     bool
}

type Handler struct {
	store    UserStore
	sessions *Service
}

func NewService(store SessionStore, cfg config.Config) *Service {
	return &Service{
		store:      store,
		cookieName: cfg.SessionCookieName,
		ttl:        time.Duration(cfg.SessionTTLHours) * time.Hour,
		sameSite:   cfg.CookieSameSite,
		secure:     cfg.CookieSecure,
	}
}

func NewHandler(store UserStore, sessions *Service) *Handler {
	return &Handler{store: store, sessions: sessions}
}

func (h *Handler) Login(c *gin.Context) {
	var request models.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	username := strings.TrimSpace(request.Username)
	if username == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	user, found := h.store.FindUserByUsername(username)
	if !found || !ComparePassword(user.PasswordHash, request.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}
	if !user.CanLogin {
		c.JSON(http.StatusForbidden, gin.H{"error": "该账号已禁用登录"})
		return
	}

	sessionID, expiresAt, err := h.sessions.Create(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	h.sessions.SetSessionCookie(c, sessionID, expiresAt)
	c.JSON(http.StatusOK, gin.H{"user": ToAuthUser(user)})
}

func (h *Handler) GetSession(c *gin.Context) {
	userID, ok := h.sessions.UserIDFromRequest(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	user, found := h.store.FindUserByID(userID)
	if !found || !user.CanLogin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": ToAuthUser(user)})
}

func (h *Handler) Logout(c *gin.Context) {
	if cookie, err := c.Cookie(h.sessions.cookieName); err == nil && cookie != "" {
		h.sessions.Delete(cookie)
	}
	h.sessions.ClearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

func (s *Service) Create(userID int) (string, time.Time, error) {
	sessionID, err := newSessionID()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(s.ttl)
	if err := s.store.CreateSession(sessionID, userID, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return sessionID, expiresAt, nil
}

func (s *Service) UserIDFromRequest(c *gin.Context) (int, bool) {
	sessionID, err := c.Cookie(s.cookieName)
	if err != nil || sessionID == "" {
		return 0, false
	}
	session, exists := s.store.FindSession(sessionID)
	if !exists {
		return 0, false
	}
	return session.UserID, true
}

func (s *Service) Delete(sessionID string) {
	s.store.DeleteSession(sessionID)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func MustHashPassword(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

func ComparePassword(passwordHash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func ToAuthUser(user models.User) models.AuthUser {
	return models.AuthUser{
		ID:         user.ID,
		Username:   user.Username,
		Name:       user.Name,
		Role:       user.Role,
		Department: user.Department,
		CanLogin:   user.CanLogin,
	}
}

func (s *Service) SetSessionCookie(c *gin.Context, sessionID string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(s.sameSite)
	c.SetCookie(s.cookieName, sessionID, maxAge, "/", "", s.secure, true)
}

func (s *Service) ClearSessionCookie(c *gin.Context) {
	c.SetSameSite(s.sameSite)
	c.SetCookie(s.cookieName, "", -1, "/", "", s.secure, true)
}

func newSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
