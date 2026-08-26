package models

import "time"

// Daily 表示 C 端发布的一条日常内容及其可见性和浏览量。
type Daily struct {
	// ID 表示日常唯一标识。
	ID int `json:"id"`
	// Content 表示经过白名单清洗的日常富文本 HTML；兼容历史纯文本记录。
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
	// LikeCount 表示日常收到的点赞数量。
	LikeCount int `json:"likeCount"`
	// CoverFileID 保存封面关联的公开图片编号，仅供服务端解析，不对外暴露。
	CoverFileID int `json:"-"`
	// CoverImage 表示封面公开中图地址。
	CoverImage string `json:"coverImage,omitempty"`
	// CoverAlt 表示封面替代文本。
	CoverAlt string `json:"coverAlt,omitempty"`
	// CoverWidth 表示封面原始宽度。
	CoverWidth int `json:"coverWidth,omitempty"`
	// CoverHeight 表示封面原始高度。
	CoverHeight int `json:"coverHeight,omitempty"`
	// CreatedAt 表示发布时间。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 表示最后更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// DailyRequest 表示 C 端发布日常时提交的正文和隐私选项。
type DailyRequest struct {
	// Content 表示待发布的日常富文本 HTML，服务端会再次清洗。
	Content string `json:"content" binding:"required"`
	// IsPrivate 表示是否仅个人可见。
	IsPrivate bool `json:"isPrivate"`
	// CoverFileID 表示手动选择的公开图片封面编号；缺省时服务端随机选择公开图片。
	CoverFileID int `json:"coverFileId"`
}

// PublicDailyComment 表示公开日常下的一条登录用户评论。
type PublicDailyComment struct {
	// ID 表示评论唯一标识。
	ID int `json:"id"`
	// UserName 表示评论作者展示名称。
	UserName string `json:"userName"`
	// Content 表示评论纯文本内容。
	Content string `json:"content"`
	// CreatedAt 表示评论时间。
	CreatedAt time.Time `json:"createdAt"`
}

// PublicDailyInteraction 表示日常点赞状态与最新评论。
type PublicDailyInteraction struct {
	// LikeCount 表示日常点赞总数。
	LikeCount int `json:"likeCount"`
	// LikedByCurrentUser 表示当前登录用户是否已点赞。
	LikedByCurrentUser bool `json:"likedByCurrentUser"`
	// Comments 表示最近一百条评论。
	Comments []PublicDailyComment `json:"comments"`
}
