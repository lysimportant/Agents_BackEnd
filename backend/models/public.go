package models

import "time"

// PublicPagination 表示公开列表的分页信息。
type PublicPagination struct {
	// Page 当前页码，从 1 开始。
	Page int `json:"page"`
	// PageSize 每页条目数。
	PageSize int `json:"pageSize"`
	// Total 符合条件的总条目数。
	Total int `json:"total"`
	// TotalPages 总页数。
	TotalPages int `json:"totalPages"`
}

// PublicListResponse 表示公开列表的通用响应结构。
type PublicListResponse struct {
	// Items 当前页的条目列表。
	Items []interface{} `json:"items"`
	// Pagination 分页信息。
	Pagination PublicPagination `json:"pagination"`
}

// PublicArticleListItem 表示公开文章列表中的单篇文章摘要。
type PublicArticleListItem struct {
	// ID 文章唯一标识。
	ID int `json:"id"`
	// Title 文章标题。
	Title string `json:"title"`
	// Category 文章分类。
	Category string `json:"category"`
	// Author 文章作者。
	Author string `json:"author"`
	// Summary 文章摘要。
	Summary string `json:"summary"`
	// Slug 文章标题的 URL 友好标识。
	Slug string `json:"slug"`
	// CoverImage 文章封面图片地址，无封面时为空。
	CoverImage string `json:"coverImage,omitempty"`
	// ContentLocale 正文实际语言。
	ContentLocale string `json:"contentLocale"`
	// Views 文章浏览次数。
	Views int `json:"views"`
	// PublishedAt 首次发布到门户的时间。
	PublishedAt time.Time `json:"publishedAt"`
	// UpdatedAt 最后更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// PublicArticleDetail 表示公开文章详情，包含正文与关联信息。
type PublicArticleDetail struct {
	PublicArticleListItem
	// Content 文章正文 HTML。
	Content string `json:"content"`
	// TableOfContents 文章目录导航。
	TableOfContents []PublicTocEntry `json:"tableOfContents"`
	// RelatedArticles 关联文章推荐。
	RelatedArticles []PublicArticleListItem `json:"relatedArticles"`
}

// PublicTocEntry 表示文章目录中的一条条目。
type PublicTocEntry struct {
	// ID 目录条目锚点标识。
	ID string `json:"id"`
	// Level 标题层级。
	Level int `json:"level"`
	// Text 标题文本。
	Text string `json:"text"`
}

// PublicFileListItem 表示公开文件列表中的单条文件摘要。
type PublicFileListItem struct {
	// ID 文件唯一标识。
	ID int `json:"id"`
	// DisplayName 文件显示名称。
	DisplayName string `json:"displayName"`
	// Category 文件分类。
	Category string `json:"category"`
	// Description 文件描述。
	Description string `json:"description"`
	// ContentType 文件 MIME 类型。
	ContentType string `json:"contentType"`
	// Size 文件大小（字节）。
	Size int64 `json:"size"`
	// PreviewURL 文件预览地址。
	PreviewURL string `json:"previewUrl,omitempty"`
	// DownloadURL 文件下载地址。
	DownloadURL string `json:"downloadUrl,omitempty"`
	// ImageWidth 图片宽度，非图片时为 0。
	ImageWidth int `json:"imageWidth,omitempty"`
	// ImageHeight 图片高度，非图片时为 0。
	ImageHeight int `json:"imageHeight,omitempty"`
	// AltText 图片替代文本。
	AltText string `json:"altText,omitempty"`
	// PublishedAt 首次发布到门户的时间。
	PublishedAt time.Time `json:"publishedAt"`
	// UpdatedAt 最后更新时间。
	UpdatedAt time.Time `json:"updatedAt"`
}

// PublicCategory 表示公开分类的聚合信息。
type PublicCategory struct {
	// Name 分类名称。
	Name string `json:"name"`
	// ArticleCount 该分类下的文章数量。
	ArticleCount int `json:"articleCount"`
	// ImageCount 该分类下的图片数量。
	ImageCount int `json:"imageCount"`
	// ResourceCount 该分类下的资源数量。
	ResourceCount int `json:"resourceCount"`
}

// PublicSiteSummary 表示站点首页聚合概览数据。
type PublicSiteSummary struct {
	// ArticleCount 文章总数。
	ArticleCount int `json:"articleCount"`
	// ImageCount 图片总数。
	ImageCount int `json:"imageCount"`
	// ResourceCount 资源总数。
	ResourceCount int `json:"resourceCount"`
	// CategoryCount 分类总数。
	CategoryCount int `json:"categoryCount"`
	// LatestArticles 最新文章列表。
	LatestArticles []PublicArticleListItem `json:"latestArticles"`
	// FeaturedImages 精选图片列表。
	FeaturedImages []PublicFileListItem `json:"featuredImages"`
	// PopularCategories 热门分类列表。
	PopularCategories []PublicCategory `json:"popularCategories"`
}

// PublicSearchResult 表示聚合搜索结果。
type PublicSearchResult struct {
	// Articles 匹配的文章列表。
	Articles []PublicArticleListItem `json:"articles"`
	// Images 匹配的图片列表。
	Images []PublicFileListItem `json:"images"`
	// Resources 匹配的资源列表。
	Resources []PublicFileListItem `json:"resources"`
}

// PublicDetailResponse 表示公开详情响应的通用结构。
type PublicDetailResponse struct {
	// Item 详情数据。
	Item interface{} `json:"item"`
}
