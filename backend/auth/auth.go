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

func NewService(store SessionStore, cfg config.Config) *Service {
	return &Service{
		store:      store,
		cookieName: cfg.SessionCookieName,
		ttl:        time.Duration(cfg.SessionTTLHours) * time.Hour,
		sameSite:   cfg.CookieSameSite,
		secure:     cfg.CookieSecure,
	}
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

func (s *Service) DeleteFromRequest(c *gin.Context) {
	if sessionID, err := c.Cookie(s.cookieName); err == nil && sessionID != "" {
		s.Delete(sessionID)
	}
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
