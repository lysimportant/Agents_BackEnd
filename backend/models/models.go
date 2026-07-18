package models

import (
	"strings"
	"time"
)

type DataPoint struct {
	ID        int       `json:"id"`
	Source    string    `json:"source"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateDataPointRequest struct {
	Source string  `json:"source" binding:"required"`
	Metric string  `json:"metric" binding:"required"`
	Value  float64 `json:"value" binding:"required"`
	Unit   string  `json:"unit"`
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Name         string    `json:"name"`
	RoleID       *int      `json:"roleId"`
	Role         string    `json:"role"`
	RoleCode     string    `json:"roleCode"`
	DepartmentID *int      `json:"departmentId"`
	Department   string    `json:"department"`
	Status       string    `json:"status"`
	Shift        string    `json:"shift"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	Age          int       `json:"age"`
	Description  string    `json:"description"`
	AvatarURL    string    `json:"avatarUrl"`
	CanLogin     bool      `json:"canLogin"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type AuthUser struct {
	ID                int      `json:"id"`
	Username          string   `json:"username"`
	Name              string   `json:"name"`
	RoleID            *int     `json:"roleId"`
	Role              string   `json:"role"`
	RoleCode          string   `json:"roleCode"`
	DepartmentID      *int     `json:"departmentId"`
	Department        string   `json:"department"`
	Status            string   `json:"status"`
	Phone             string   `json:"phone"`
	Email             string   `json:"email"`
	Age               int      `json:"age"`
	Description       string   `json:"description"`
	AvatarURL         string   `json:"avatarUrl"`
	CanLogin          bool     `json:"canLogin"`
	ActionPermissions []string `json:"actionPermissions"`
}

func (u User) LoginAllowed() bool {
	return u.CanLogin && strings.TrimSpace(u.Status) != "停用"
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Session struct {
	UserID    int
	ExpiresAt time.Time
}

type Menu struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Path      string    `json:"path"`
	Icon      string    `json:"icon"`
	ParentID  *int      `json:"parentId,omitempty"`
	Sort      int       `json:"sort"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	RoleID       *int    `json:"roleId"`
	Role         string  `json:"role"`
	DepartmentID *int    `json:"departmentId"`
	Department   string  `json:"department"`
	Status       string  `json:"status"`
	Shift        string  `json:"shift"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Age          *int    `json:"age"`
	Description  *string `json:"description"`
	AvatarURL    *string `json:"avatarUrl"`
	CanLogin     *bool   `json:"canLogin"`
	Password     string  `json:"password"`
}

type UserProfileRequest struct {
	Name        *string `json:"name"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	Age         *int    `json:"age"`
	Description *string `json:"description"`
	AvatarURL   *string `json:"avatarUrl"`
}

type PasswordCodeRequest struct {
	Email string `json:"email"`
}

type ChangePasswordRequest struct {
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type UserMenusRequest struct {
	MenuIDs []int `json:"menuIds" binding:"required"`
}

type UserActionsRequest struct {
	ActionCodes []string `json:"actionCodes"`
}

type UserPermissionDetail struct {
	DepartmentMenuIDs    []int    `json:"departmentMenuIds"`
	RoleMenuIDs          []int    `json:"roleMenuIds"`
	UserMenuIDs          []int    `json:"userMenuIds"`
	EffectiveMenuIDs     []int    `json:"effectiveMenuIds"`
	RoleActionCodes      []string `json:"roleActionCodes"`
	UserActionCodes      []string `json:"userActionCodes"`
	EffectiveActionCodes []string `json:"effectiveActionCodes"`
}

type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Sort        int       `json:"sort"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type RoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      string `json:"status" binding:"required"`
}

type Department struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	ParentID  *int      `json:"parentId"`
	Leader    string    `json:"leader"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Sort      int       `json:"sort"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DepartmentRequest struct {
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	ParentID *int   `json:"parentId"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sort     int    `json:"sort"`
	Status   string `json:"status" binding:"required"`
}

type MenuRequest struct {
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	ParentID *int   `json:"parentId"`
	Sort     int    `json:"sort"`
	Status   string `json:"status" binding:"required"`
}

type Article struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Author    string    `json:"author"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	Views     int       `json:"views"`
	OwnerID   int       `json:"ownerId"`
	OwnerName string    `json:"ownerName"`
	IsPrivate bool      `json:"isPrivate"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ArticleRequest struct {
	Title     string `json:"title" binding:"required"`
	Category  string `json:"category" binding:"required"`
	Author    string `json:"author" binding:"required"`
	Status    string `json:"status" binding:"required"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
	Views     int    `json:"views"`
	IsPrivate bool   `json:"isPrivate"`
}

type ManagedFile struct {
	ID           int        `json:"id"`
	DisplayName  string     `json:"displayName"`
	OriginalName string     `json:"originalName"`
	Category     string     `json:"category"`
	Description  string     `json:"description"`
	ContentType  string     `json:"contentType"`
	Size         int64      `json:"size"`
	StorageName  string     `json:"storageName"`
	OwnerID      int        `json:"ownerId"`
	OwnerName    string     `json:"ownerName"`
	IsPrivate    bool       `json:"isPrivate"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

type FileMetadataRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"isPrivate"`
}

type FileContentRequest struct {
	Content string `json:"content" binding:"required"`
}

type SocketConversation struct {
	ID           string    `json:"id"`
	VisitorName  string    `json:"visitorName"`
	Status       string    `json:"status"`
	Online       bool      `json:"online"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	LastMessage  string    `json:"lastMessage"`
	MessageCount int       `json:"messageCount"`
}

type SocketMessage struct {
	ID                int       `json:"id"`
	ConversationID    string    `json:"conversationId"`
	SenderType        string    `json:"senderType"`
	SenderName        string    `json:"senderName"`
	MessageType       string    `json:"messageType"`
	Content           string    `json:"content"`
	AttachmentName    string    `json:"attachmentName"`
	AttachmentType    string    `json:"attachmentType"`
	AttachmentSize    int64     `json:"attachmentSize"`
	AttachmentStorage string    `json:"-"`
	CreatedAt         time.Time `json:"createdAt"`
}

type SocketMessageRequest struct {
	ConversationID string `json:"conversationId"`
	MessageType    string `json:"messageType" binding:"required"`
	Content        string `json:"content"`
}
