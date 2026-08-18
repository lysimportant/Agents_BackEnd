package models

import (
	"strings"
	"time"
)

// DataPoint 表示从指定来源采集的一条工作台指标。
type DataPoint struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Source 表示来源。
	Source string `json:"source"`
	// Metric 表示指标。
	Metric string `json:"metric"`
	// Value 表示值。
	Value float64 `json:"value"`
	// Unit 表示计量单位。
	Unit string `json:"unit"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
}

// VisitorAccessRecord 保存单次 HTTP 请求涉及隐私的访问元数据。
type VisitorAccessRecord struct {
	// ID 表示标识。
	ID int `json:"id"`
	// IP 表示变量 IP。
	IP string `json:"ip"`
	// ForwardedIP 表示变量 ForwardedIP。
	ForwardedIP string `json:"forwardedIp,omitempty"`
	// Country 表示国家或地区。
	Country string `json:"country"`
	// Region 表示变量 Region。
	Region string `json:"region"`
	// City 表示变量 City。
	City string `json:"city"`
	// ISP 表示变量 ISP。
	ISP string `json:"isp"`
	// Host 表示主机地址。
	Host string `json:"host"`
	// Method 表示请求方法。
	Method string `json:"method"`
	// Path 表示路径。
	Path string `json:"path"`
	// StatusCode 表示状态编码。
	StatusCode int `json:"statusCode"`
	// DurationMS 表示耗时。
	DurationMS int64 `json:"durationMs"`
	// Bytes 表示字节数。
	Bytes int64 `json:"bytes"`
	// UserAgent 表示用户。
	UserAgent string `json:"userAgent"`
	// Browser 表示浏览器。
	Browser string `json:"browser"`
	// OS 表示变量 OS。
	OS string `json:"os"`
	// Device 表示设备。
	Device string `json:"device"`
	// Referer 表示变量 Referer。
	Referer string `json:"referer"`
	// AcceptLanguage 表示变量 AcceptLanguage。
	AcceptLanguage string `json:"acceptLanguage"`
	// UserID 表示用户标识。
	UserID *int `json:"userId,omitempty"`
	// UserName 表示用户名称。
	UserName string `json:"userName,omitempty"`
	// Authenticated 表示变量 Authenticated。
	Authenticated bool `json:"authenticated"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
}

// VisitorAnalyticsFilter 描述访问分析的分页和筛选条件。
type VisitorAnalyticsFilter struct {
	// From 表示起始时间。
	From time.Time
	// To 表示变量 To。
	To time.Time
	// Range 表示时间范围。
	Range string
	// Page 表示页码。
	Page int
	// PageSize 表示页码大小。
	PageSize int
	// Keyword 表示搜索关键词。
	Keyword string
	// StatusCode 表示状态编码。
	StatusCode *int
}

// VisitorAnalyticsDimension 表示分析图表使用的具名聚合维度。
type VisitorAnalyticsDimension struct {
	// Name 表示名称。
	Name string `json:"name"`
	// Value 表示值。
	Value int `json:"value"`
}

// VisitorAnalyticsPoint 表示访问趋势时间序列中的一个数据点。
type VisitorAnalyticsPoint struct {
	// Label 表示显示标签。
	Label string `json:"label"`
	// Value 表示值。
	Value int `json:"value"`
}

// VisitorAnalyticsSummary 包含访问聚合计数和图表序列。
type VisitorAnalyticsSummary struct {
	// TotalRequests 表示总数。
	TotalRequests int64 `json:"totalRequests"`
	// UniqueIPs 表示去重结果。
	UniqueIPs int64 `json:"uniqueIps"`
	// AuthenticatedRequests 表示请求参数。
	AuthenticatedRequests int64 `json:"authenticatedRequests"`
	// ErrorRequests 表示错误状态。
	ErrorRequests int64 `json:"errorRequests"`
	// AverageDurationMS 表示耗时。
	AverageDurationMS int64 `json:"averageDurationMs"`
	// Countries 表示国家或地区。
	Countries []VisitorAnalyticsDimension `json:"countries"`
	// Paths 表示路径。
	Paths []VisitorAnalyticsDimension `json:"paths"`
	// Timeline 表示变量 Timeline。
	Timeline []VisitorAnalyticsPoint `json:"timeline"`
}

// VisitorAnalyticsResponse 表示访问分析 API 的分页响应。
type VisitorAnalyticsResponse struct {
	// Records 表示记录。
	Records []VisitorAccessRecord `json:"records"`
	// Total 表示总数。
	Total int `json:"total"`
	// Page 表示页码。
	Page int `json:"page"`
	// PageSize 表示页码大小。
	PageSize int `json:"pageSize"`
	// Summary 表示摘要。
	Summary VisitorAnalyticsSummary `json:"summary"`
}

// CreateDataPointRequest 表示创建工作台指标时经过校验的请求体。
type CreateDataPointRequest struct {
	// Source 表示来源。
	Source string `json:"source" binding:"required"`
	// Metric 表示指标。
	Metric string `json:"metric" binding:"required"`
	// Value 表示值。
	Value float64 `json:"value" binding:"required"`
	// Unit 表示计量单位。
	Unit string `json:"unit"`
}

// User 表示持久化的管理员或操作人员账户视图。
type User struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Username 表示用户名。
	Username string `json:"username"`
	// Name 表示名称。
	Name string `json:"name"`
	// RoleID 表示角色标识。
	RoleID *int `json:"roleId"`
	// Role 表示角色。
	Role string `json:"role"`
	// RoleCode 表示角色编码。
	RoleCode string `json:"roleCode"`
	// DepartmentID 表示部门标识。
	DepartmentID *int `json:"departmentId"`
	// Department 表示部门。
	Department string `json:"department"`
	// Status 表示状态。
	Status string `json:"status"`
	// Shift 表示班次。
	Shift string `json:"shift"`
	// Phone 表示电话号码。
	Phone string `json:"phone"`
	// Email 表示邮箱地址。
	Email string `json:"email"`
	// Age 表示年龄。
	Age int `json:"age"`
	// Description 表示说明。
	Description string `json:"description"`
	// AvatarURL 表示头像地址。
	AvatarURL string `json:"avatarUrl"`
	// CanLogin 表示登录。
	CanLogin bool `json:"canLogin"`
	// PasswordHash 表示密码。
	PasswordHash string `json:"-"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// AuthUser 表示认证接口可安全返回的用户投影视图。
type AuthUser struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Username 表示用户名。
	Username string `json:"username"`
	// Name 表示名称。
	Name string `json:"name"`
	// RoleID 表示角色标识。
	RoleID *int `json:"roleId"`
	// Role 表示角色。
	Role string `json:"role"`
	// RoleCode 表示角色编码。
	RoleCode string `json:"roleCode"`
	// DepartmentID 表示部门标识。
	DepartmentID *int `json:"departmentId"`
	// Department 表示部门。
	Department string `json:"department"`
	// Status 表示状态。
	Status string `json:"status"`
	// Phone 表示电话号码。
	Phone string `json:"phone"`
	// Email 表示邮箱地址。
	Email string `json:"email"`
	// Age 表示年龄。
	Age int `json:"age"`
	// Description 表示说明。
	Description string `json:"description"`
	// AvatarURL 表示头像地址。
	AvatarURL string `json:"avatarUrl"`
	// CanLogin 表示登录。
	CanLogin bool `json:"canLogin"`
	// ActionPermissions 表示操作权限权限。
	ActionPermissions []string `json:"actionPermissions"`
}

// LoginAllowed 判断账户是否允许创建新的登录会话。
func (u User) LoginAllowed() bool {
	return u.CanLogin && strings.TrimSpace(u.Status) != "停用"
}

// LoginRequest 表示用户名和密码登录请求体。
type LoginRequest struct {
	// Username 表示用户名。
	Username string `json:"username" binding:"required"`
	// Password 表示密码。
	Password string `json:"password" binding:"required"`
}

// Session 标识已登录用户及其会话过期时间。
type Session struct {
	// UserID 表示用户标识。
	UserID int
	// ExpiresAt 表示变量 ExpiresAt。
	ExpiresAt time.Time
}

// Menu 表示用于导航和路由鉴权的菜单项。
type Menu struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Name 表示名称。
	Name string `json:"name"`
	// Code 表示编码。
	Code string `json:"code"`
	// Path 表示路径。
	Path string `json:"path"`
	// Icon 表示图标。
	Icon string `json:"icon"`
	// ParentID 表示标识。
	ParentID *int `json:"parentId,omitempty"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserRequest 表示管理员创建或更新用户时的请求体。
type UserRequest struct {
	// Username 表示用户名。
	Username string `json:"username" binding:"required"`
	// Name 表示名称。
	Name string `json:"name" binding:"required"`
	// RoleID 表示角色标识。
	RoleID *int `json:"roleId"`
	// Role 表示角色。
	Role string `json:"role"`
	// DepartmentID 表示部门标识。
	DepartmentID *int `json:"departmentId"`
	// Department 表示部门。
	Department string `json:"department"`
	// Status 表示状态。
	Status string `json:"status"`
	// Shift 表示班次。
	Shift string `json:"shift"`
	// Phone 表示电话号码。
	Phone string `json:"phone"`
	// Email 表示邮箱地址。
	Email string `json:"email"`
	// Age 表示年龄。
	Age *int `json:"age"`
	// Description 表示说明。
	Description *string `json:"description"`
	// AvatarURL 表示头像地址。
	AvatarURL *string `json:"avatarUrl"`
	// CanLogin 表示登录。
	CanLogin *bool `json:"canLogin"`
	// Password 表示密码。
	Password string `json:"password"`
}

// UserProfileRequest 包含已登录用户可自行修改的资料字段。
type UserProfileRequest struct {
	// Name 表示名称。
	Name *string `json:"name"`
	// Email 表示邮箱地址。
	Email *string `json:"email"`
	// Phone 表示电话号码。
	Phone *string `json:"phone"`
	// Age 表示年龄。
	Age *int `json:"age"`
	// Description 表示说明。
	Description *string `json:"description"`
	// AvatarURL 表示头像地址。
	AvatarURL *string `json:"avatarUrl"`
}

// PasswordCodeRequest 表示向指定邮箱申请密码验证码的请求体。
type PasswordCodeRequest struct {
	// Email 表示邮箱地址。
	Email string `json:"email"`
}

// ChangePasswordRequest 包含验证码和新密码。
type ChangePasswordRequest struct {
	// Code 表示编码。
	Code string `json:"code" binding:"required"`
	// NewPassword 表示密码。
	NewPassword string `json:"newPassword" binding:"required"`
}

// UserMenusRequest 用于替换用户的个人附加菜单权限。
type UserMenusRequest struct {
	// MenuIDs 表示菜单标识列表。
	MenuIDs []int `json:"menuIds" binding:"required"`
}

// UserActionsRequest 用于替换用户的个人附加动作权限。
type UserActionsRequest struct {
	// ActionCodes 表示操作权限编码。
	ActionCodes []string `json:"actionCodes"`
}

// UserPermissionDetail 展示权限来源及其最终有效并集。
type UserPermissionDetail struct {
	// DepartmentMenuIDs 表示部门菜单标识列表。
	DepartmentMenuIDs []int `json:"departmentMenuIds"`
	// RoleMenuIDs 表示角色菜单标识列表。
	RoleMenuIDs []int `json:"roleMenuIds"`
	// UserMenuIDs 表示用户菜单标识列表。
	UserMenuIDs []int `json:"userMenuIds"`
	// EffectiveMenuIDs 表示最终生效菜单标识列表。
	EffectiveMenuIDs []int `json:"effectiveMenuIds"`
	// RoleActionCodes 表示角色。
	RoleActionCodes []string `json:"roleActionCodes"`
	// UserActionCodes 表示用户。
	UserActionCodes []string `json:"userActionCodes"`
	// EffectiveActionCodes 表示最终生效。
	EffectiveActionCodes []string `json:"effectiveActionCodes"`
}

// Role 表示由不可变编码标识的稳定角色定义。
type Role struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Name 表示名称。
	Name string `json:"name"`
	// Code 表示编码。
	Code string `json:"code"`
	// Description 表示说明。
	Description string `json:"description"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// RoleRequest 表示创建或更新角色的请求体。
type RoleRequest struct {
	// Name 表示名称。
	Name string `json:"name" binding:"required"`
	// Code 表示编码。
	Code string `json:"code" binding:"required"`
	// Description 表示说明。
	Description string `json:"description"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status" binding:"required"`
}

// Department 表示部门树中的一个组织单元。
type Department struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Name 表示名称。
	Name string `json:"name"`
	// Code 表示编码。
	Code string `json:"code"`
	// ParentID 表示标识。
	ParentID *int `json:"parentId"`
	// Leader 表示负责人。
	Leader string `json:"leader"`
	// Phone 表示电话号码。
	Phone string `json:"phone"`
	// Email 表示邮箱地址。
	Email string `json:"email"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// DepartmentRequest 表示创建或更新部门的请求体。
type DepartmentRequest struct {
	// Name 表示名称。
	Name string `json:"name" binding:"required"`
	// Code 表示编码。
	Code string `json:"code" binding:"required"`
	// ParentID 表示标识。
	ParentID *int `json:"parentId"`
	// Leader 表示负责人。
	Leader string `json:"leader"`
	// Phone 表示电话号码。
	Phone string `json:"phone"`
	// Email 表示邮箱地址。
	Email string `json:"email"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status" binding:"required"`
}

// MenuRequest 表示创建或更新菜单的请求体。
type MenuRequest struct {
	// Name 表示名称。
	Name string `json:"name" binding:"required"`
	// Code 表示编码。
	Code string `json:"code" binding:"required"`
	// Path 表示路径。
	Path string `json:"path"`
	// Icon 表示图标。
	Icon string `json:"icon"`
	// ParentID 表示标识。
	ParentID *int `json:"parentId"`
	// Sort 表示排序。
	Sort int `json:"sort"`
	// Status 表示状态。
	Status string `json:"status" binding:"required"`
}

// Article 表示知识库文章及其所有权元数据。
type Article struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Title 表示标题。
	Title string `json:"title"`
	// Category 表示分类。
	Category string `json:"category"`
	// Author 表示作者。
	Author string `json:"author"`
	// Status 表示状态。
	Status string `json:"status"`
	// Summary 表示摘要。
	Summary string `json:"summary"`
	// Content 表示内容。
	Content string `json:"content"`
	// Views 表示查看次数。
	Views int `json:"views"`
	// OwnerID 表示所有者标识。
	OwnerID int `json:"ownerId"`
	// OwnerName 表示所有者名称。
	OwnerName string `json:"ownerName"`
	// IsPrivate 表示变量 IsPrivate。
	IsPrivate bool `json:"isPrivate"`
	// Is18R 是否开启 18R 限制。
	Is18R bool `json:"is18r"`
	// PortalPublishedAt 表示首次发布时间。
	PortalPublishedAt *time.Time `json:"portalPublishedAt,omitempty"`
	// ContentLocale 表示正文实际语言，用于 SEO 的 lang 与结构化数据。
	ContentLocale string `json:"contentLocale"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// ArticleRequest 表示创建或更新文章的请求体。
type ArticleRequest struct {
	// Title 表示标题。
	Title string `json:"title" binding:"required"`
	// Category 表示分类。
	Category string `json:"category" binding:"required"`
	// Author 表示作者。
	Author string `json:"author" binding:"required"`
	// Status 表示状态。
	Status string `json:"status" binding:"required"`
	// Summary 表示摘要。
	Summary string `json:"summary"`
	// Content 表示内容。
	Content string `json:"content"`
	// Views 表示查看次数。
	Views int `json:"views"`
	// IsPrivate 表示变量 IsPrivate。
	IsPrivate bool `json:"isPrivate"`
	// Is18R 是否开启 18R 限制。
	Is18R bool `json:"is18r"`
	// ContentLocale 表示正文实际语言，受支持语言白名单校验。
	ContentLocale string `json:"contentLocale"`
}

// ManagedFile 描述文件管理页面中的受管文件。
type ManagedFile struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Source 表示来源。
	Source string `json:"source,omitempty"`
	// DisplayName 表示名称。
	DisplayName string `json:"displayName"`
	// OriginalName 表示名称。
	OriginalName string `json:"originalName"`
	// Category 表示分类。
	Category string `json:"category"`
	// Description 表示说明。
	Description string `json:"description"`
	// ContentType 表示内容。
	ContentType string `json:"contentType"`
	// Size 表示大小。
	Size int64 `json:"size"`
	// StorageName 表示存储名称。
	StorageName string `json:"storageName"`
	// ContentSHA256 保存服务端内容去重哈希，不向客户端暴露。
	ContentSHA256 string `json:"-"`
	// OwnerID 表示所有者标识。
	OwnerID int `json:"ownerId"`
	// OwnerName 表示所有者名称。
	OwnerName string `json:"ownerName"`
	// IsPrivate 表示变量 IsPrivate。
	IsPrivate bool `json:"isPrivate"`
	// Is18R 是否开启 18R 限制，开启后仅登录且开启 18R 的用户可见。
	Is18R bool `json:"is18r"`
	// ImageWidth 表示公开图片原始宽度，首期用于瀑布流占位。
	ImageWidth int `json:"imageWidth,omitempty"`
	// ImageHeight 表示公开图片原始高度，首期用于瀑布流占位。
	ImageHeight int `json:"imageHeight,omitempty"`
	// ReadOnly 表示只读状态。
	ReadOnly bool `json:"readOnly"`
	// PreviewURL 表示预览地址。
	PreviewURL string `json:"previewUrl,omitempty"`
	// DownloadURL 表示地址。
	DownloadURL string `json:"downloadUrl,omitempty"`
	// StoragePath 表示存储路径。
	StoragePath string `json:"-"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
	// DeletedAt 表示删除状态。
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// FileMetadataRequest 用于更新展示元数据而不替换文件内容。
type FileMetadataRequest struct {
	// DisplayName 表示名称。
	DisplayName string `json:"displayName" binding:"required"`
	// Category 表示分类。
	Category string `json:"category"`
	// Description 表示说明。
	Description string `json:"description"`
	// IsPrivate 表示变量 IsPrivate。
	IsPrivate bool `json:"isPrivate"`
	// Is18R 是否开启 18R 限制。
	Is18R bool `json:"is18r"`
}

// FileContentRequest 用于替换可编辑文件的文本内容。
type FileContentRequest struct {
	// Content 表示内容。
	Content string `json:"content" binding:"required"`
}

// SocketConversation 表示客服聊天会话摘要。
type SocketConversation struct {
	// ID 表示标识。
	ID string `json:"id"`
	// VisitorName 表示访问者名称。
	VisitorName string `json:"visitorName"`
	// Title 表示标题。
	Title string `json:"title"`
	// Status 表示状态。
	Status string `json:"status"`
	// Online 表示在线状态。
	Online bool `json:"online"`
	// LastSeenAt 表示已处理集合。
	LastSeenAt time.Time `json:"lastSeenAt"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
	// LastMessage 表示消息。
	LastMessage string `json:"lastMessage"`
	// MessageCount 表示消息数量。
	MessageCount int `json:"messageCount"`
}

// SocketConversationTitleRequest 用于修改客服会话标题。
type SocketConversationTitleRequest struct {
	// Title 表示标题。
	Title string `json:"title" binding:"required"`
}

// SocketMessage 表示客服聊天消息投影视图。
type SocketMessage struct {
	// ID 表示标识。
	ID int `json:"id"`
	// ConversationID 表示会话标识。
	ConversationID string `json:"conversationId"`
	// SenderType 表示类型。
	SenderType string `json:"senderType"`
	// SenderName 表示名称。
	SenderName string `json:"senderName"`
	// MessageType 表示消息。
	MessageType string `json:"messageType"`
	// Content 表示内容。
	Content string `json:"content"`
	// AttachmentName 表示附件名称。
	AttachmentName string `json:"attachmentName"`
	// AttachmentType 表示附件。
	AttachmentType string `json:"attachmentType"`
	// AttachmentSize 表示附件大小。
	AttachmentSize int64 `json:"attachmentSize"`
	// AttachmentStorage 表示附件存储。
	AttachmentStorage string `json:"-"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
}

// SocketMessageRequest 表示发送客服聊天消息的请求体。
type SocketMessageRequest struct {
	// ConversationID 表示会话标识。
	ConversationID string `json:"conversationId"`
	// MessageType 表示消息。
	MessageType string `json:"messageType" binding:"required"`
	// Content 表示内容。
	Content string `json:"content"`
}

// InternalChatUser 表示可参与内部聊天的同事。
type InternalChatUser struct {
	// ID 表示标识。
	ID int `json:"id"`
	// Username 表示用户名。
	Username string `json:"username"`
	// Name 表示名称。
	Name string `json:"name"`
	// Department 表示部门。
	Department string `json:"department"`
	// Online 表示在线状态。
	Online bool `json:"online"`
}

// InternalChatMessage 表示经过鉴权的内部私聊或群发消息。
type InternalChatMessage struct {
	// ID 表示标识。
	ID int `json:"id"`
	// SenderID 表示标识。
	SenderID int `json:"senderId"`
	// SenderName 表示名称。
	SenderName string `json:"senderName"`
	// RecipientID 表示标识。
	RecipientID *int `json:"recipientId,omitempty"`
	// RecipientName 表示名称。
	RecipientName string `json:"recipientName,omitempty"`
	// Content 表示内容。
	Content string `json:"content"`
	// Attachments 表示附件。
	Attachments []InternalChatAttachment `json:"attachments"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
}

// InternalChatMessageRequest 包含消息文本和已上传附件 ID。
type InternalChatMessageRequest struct {
	// RecipientID 表示标识。
	RecipientID *int `json:"recipientId"`
	// Content 表示内容。
	Content string `json:"content"`
	// AttachmentIDs 表示附件标识列表。
	AttachmentIDs []int `json:"attachmentIds"`
}

// InternalChatAttachment 表示预览和下载均需参与者鉴权的内部聊天附件。
type InternalChatAttachment struct {
	// ID 表示标识。
	ID int `json:"id"`
	// MessageID 表示消息标识。
	MessageID *int `json:"-"`
	// OwnerID 表示所有者标识。
	OwnerID int `json:"-"`
	// OriginalName 表示名称。
	OriginalName string `json:"originalName"`
	// StoredName 表示名称。
	StoredName string `json:"-"`
	// MimeType 表示媒体类型类型。
	MimeType string `json:"mimeType"`
	// Size 表示大小。
	Size int64 `json:"size"`
	// IsImage 表示图片。
	IsImage bool `json:"isImage"`
	// PreviewURL 表示预览地址。
	PreviewURL string `json:"previewUrl"`
	// DownloadURL 表示地址。
	DownloadURL string `json:"downloadUrl"`
	// CreatedAt 表示创建时间。
	CreatedAt time.Time `json:"createdAt"`
}
