/**
 * 门户公开 API 的类型定义，对应后端 /api/public/* 接口的响应结构。
 * 字段与后端 JSON 保持一致，供页面与组件消费公开内容数据。
 */

/** PublicPagination 表示公开列表的分页信息。 */
export interface PublicPagination {
  /** page 当前页码，从 1 开始。 */
  page: number;
  /** pageSize 每页条数。 */
  pageSize: number;
  /** total 总条数。 */
  total: number;
  /** totalPages 总页数。 */
  totalPages: number;
}

/** PublicArticleListItem 表示公开文章列表中的单篇文章。 */
export interface PublicArticleListItem {
  /** id 文章唯一标识。 */
  id: number;
  /** title 文章标题。 */
  title: string;
  /** category 文章分类。 */
  category: string;
  /** author 文章作者。 */
  author: string;
  /** summary 文章摘要。 */
  summary: string;
  /** slug 文章标题的 URL 友好标识。 */
  slug: string;
  /** coverImage 封面图片地址，可能为空。 */
  coverImage?: string;
  /** contentLocale 正文实际语言。 */
  contentLocale: string;
  /** views 浏览次数。 */
  views: number;
  /** publishedAt 首次发布到门户的时间（UTC RFC 3339）。 */
  publishedAt: string;
  /** updatedAt 最后更新时间（UTC RFC 3339）。 */
  updatedAt: string;
}

/** PublicTocEntry 表示文章目录中的一条条目。 */
export interface PublicTocEntry {
  /** id 目录条目锚点 ID，与正文标题 id 对应。 */
  id: string;
  /** level 标题层级。 */
  level: number;
  /** text 标题文本。 */
  text: string;
}

/** PublicArticleDetail 表示公开文章详情。 */
export interface PublicArticleDetail extends PublicArticleListItem {
  /** content 文章正文 HTML。 */
  content: string;
  /** tableOfContents 文章目录。 */
  tableOfContents: PublicTocEntry[];
  /** relatedArticles 关联文章。 */
  relatedArticles: PublicArticleListItem[];
}

/** PublicFileListItem 表示公开文件列表中的单条文件。 */
export interface PublicFileListItem {
  /** id 文件唯一标识。 */
  id: number;
  /** displayName 文件显示名称。 */
  displayName: string;
  /** category 文件分类。 */
  category: string;
  /** description 文件描述。 */
  description: string;
  /** contentType 文件 MIME 类型。 */
  contentType: string;
  /** size 文件大小（字节）。 */
  size: number;
  /** previewUrl 文件预览地址，可能为空。 */
  previewUrl?: string;
  /** downloadUrl 文件下载地址，可能为空。 */
  downloadUrl?: string;
  /** imageWidth 图片宽度，非图片时为 0。 */
  imageWidth?: number;
  /** imageHeight 图片高度，非图片时为 0。 */
  imageHeight?: number;
  /** altText 图片替代文本。 */
  altText?: string;
  /** publishedAt 首次发布到门户的时间。 */
  publishedAt: string;
  /** updatedAt 最后更新时间。 */
  updatedAt: string;
}

/** PublicCategory 表示公开分类聚合信息。 */
export interface PublicCategory {
  /** name 分类名称，同时作为分类页 URL 标识。 */
  name: string;
  /** articleCount 分类下的文章数量。 */
  articleCount: number;
  /** imageCount 分类下的图片数量。 */
  imageCount: number;
  /** resourceCount 分类下的资源数量。 */
  resourceCount: number;
}

/** PublicSiteSummary 表示站点首页聚合概览数据。 */
export interface PublicSiteSummary {
  /** articleCount 文章总数。 */
  articleCount: number;
  /** imageCount 图片总数。 */
  imageCount: number;
  /** resourceCount 资源总数。 */
  resourceCount: number;
  /** categoryCount 分类总数。 */
  categoryCount: number;
  /** latestArticles 最新文章列表。 */
  latestArticles: PublicArticleListItem[];
  /** featuredImages 精选图片列表。 */
  featuredImages: PublicFileListItem[];
  /** popularCategories 热门分类列表。 */
  popularCategories: PublicCategory[];
}

/** PublicSearchResult 表示聚合搜索结果。 */
export interface PublicSearchResult {
  /** articles 匹配的文章。 */
  articles: PublicArticleListItem[];
  /** images 匹配的图片。 */
  images: PublicFileListItem[];
  /** resources 匹配的资源。 */
  resources: PublicFileListItem[];
}

/** PublicListResponse 表示公开列表响应的通用结构。 */
export interface PublicListResponse<T> {
  /** items 当前页条目。 */
  items: T[];
  /** pagination 分页信息。 */
  pagination: PublicPagination;
}

/** PublicDetailResponse 表示公开详情响应的通用结构。 */
export interface PublicDetailResponse<T> {
  /** item 详情数据。 */
  item: T;
}

/** PublicError 表示公开 API 的错误响应。 */
export interface PublicError {
  /** code 错误编码，供前端判断错误类型。 */
  code: string;
  /** error 错误描述信息。 */
  error: string;
}