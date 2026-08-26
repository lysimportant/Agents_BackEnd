package models

import "time"

// Daily 表示 C 端发布的一条日常内容及其可见性和浏览量。
type Daily struct {
	// ID 表示日常唯一标识。
	ID int `json:"id"`
	// Content 表示日常正文纯文本。
	Content string `json:"content"`
	// OwnerID 表示发布用户标识；该字段仅用于服务端所有权判断。
	OwnerID int `json:"-"`
	// AuthorName 表示发布人的展示名称。
	AuthorName string `json:"authorName"`
	// OwnerName 为前端兼容保留的发布人展示名称别名。
	OwnerName string `json:"ownerName,omitempty"`
	// IsPrivate 表示是否仅发布人本人可见。
	IsPrivate bool `json:"isPrivate"`
	// Views 表示日常详情被查看的次数。
	Views int `json:"views"`
	// CreatedAt 表示发布时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示最后更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// DailyRequest 表示 C 端发布日常时提交的正文和隐私选项。
type DailyRequest struct {
	// Content 表示待发布的日常正文。
	Content string `json:"content" binding:"required"`
	// IsPrivate 表示是否仅个人可见。
	IsPrivate bool `json:"isPrivate"`
}
