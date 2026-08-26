/**
 * C 端公开内容类型定义。
 * 与后端 /api/public/* 的响应契约严格对齐，字段统一使用 camelCase。
 */

/** 公开列表分页信息，对应后端 PublicPagination。 */
export interface PublicPagination {
  /** 当前页码，从 1 开始。 */
  page: number;
  /** 每页条目数。 */
  pageSize: number;
  /** 符合条件的总条目数。 */
  total: number;
  /** 总页数。 */
  totalPages: number;
}

/** 公开列表通用响应结构，items 为当前页条目列表。 */
export interface PublicListResponse<T> {
  /** 当前页条目列表。 */
  items: T[];
  /** 分页信息。 */
  pagination: PublicPagination;
}

/** 公开文章列表项，对应后端 PublicArticleListItem。 */
export interface PublicArticleListItem {
  /** 文章唯一标识。 */
  id: number;
  /** 文章标题。 */
  title: string;
  /** 文章分类。 */
  category: string;
  /** 文章作者。 */
  author: string;
  /** 文章摘要。 */
  summary: string;
  /** 标题派生的 URL 友好标识。 */
  slug: string;
  /** 文章封面图片地址，无封面时可能缺省。 */
  coverImage?: string;
  /** 正文实际语言。 */
  contentLocale: string;
  /** 文章浏览次数。 */
  views: number;
  /** 首次发布到门户的时间（RFC 3339）。 */
  publishedAt: string;
  /** 最后更新时间（RFC 3339）。 */
  updatedAt: string;
}

/** 公开文章目录条目，对应后端 PublicTocEntry。 */
export interface PublicTocEntry {
  /** 目录锚点标识。 */
  id: string;
  /** 标题层级。 */
  level: number;
  /** 标题文本。 */
  text: string;
}

/** 公开文章详情，对应后端 PublicArticleDetail。 */
export interface PublicArticleDetail extends PublicArticleListItem {
  /** 经过白名单清洗并重写媒体后的正文 HTML。 */
  content: string;
  /** 文章目录导航。 */
  tableOfContents: PublicTocEntry[];
  /** 同分类相关文章推荐。 */
  relatedArticles: PublicArticleListItem[];
}

/** 公开文件列表项，图片与资源共用，对应后端 PublicFileListItem。 */
export interface PublicFileListItem {
  /** 文件唯一标识。 */
  id: number;
  /** 文件显示名称。 */
  displayName: string;
  /** 文件分类。 */
  category: string;
  /** 文件描述。 */
  description: string;
  /** 文件标签，用于公开展示和搜索。 */
  tags: string[];
  /** 文件 MIME 类型。 */
  contentType: string;
  /** 文件大小（字节）。 */
  size: number;
  /** 文件预览地址（相对路径）。 */
  previewUrl?: string;
  /** 图片缩略图地址（相对路径），非图片时缺省。 */
  thumbnailUrl?: string;
  /** 图片屏幕适配地址（相对路径），非图片时缺省。 */
  mediumUrl?: string;
  /** 文件下载地址（相对路径）。 */
  downloadUrl?: string;
  /** 图片宽度，非图片时缺省。 */
  imageWidth?: number;
  /** 图片高度，非图片时缺省。 */
  imageHeight?: number;
  /** 图片替代文本，非图片时缺省。 */
  altText?: string;
  /** 首次发布到门户的时间（RFC 3339）。 */
  publishedAt: string;
  /** 最后更新时间（RFC 3339）。 */
  updatedAt: string;
  /** 图片当前点赞数量。 */
  likeCount: number;
  /** 文件上传人的公开展示名称。 */
  ownerName?: string;
  /** 文件被公开浏览的次数。 */
  views: number;
}

/** 公开图片下的一条登录用户评论。 */
export interface PublicFileComment {
  /** 评论唯一标识。 */
  id: number;
  /** 评论作者展示名称。 */
  userName: string;
  /** 评论纯文本内容。 */
  content: string;
  /** 评论发送时间。 */
  createdAt: string;
}

/** 当前图片的点赞状态和最近评论。 */
export interface PublicFileInteraction {
  /** 图片点赞总数。 */
  likeCount: number;
  /** 当前登录用户是否已经点赞。 */
  likedByCurrentUser: boolean;
  /** 最近评论，按发送时间正序排列。 */
  comments: PublicFileComment[];
}

/** 门户追加图片标签后的权威结果。 */
export interface PublicFileTagResponse {
  /** 图片当前全部标签。 */
  tags: string[];
  /** 本次请求是否实际增加了新标签。 */
  added: boolean;
}

/** 公开分类聚合信息，对应后端 PublicCategory。 */
export interface PublicCategory {
  /** 分类名称。 */
  name: string;
  /** 该分类下的文章数量。 */
  articleCount: number;
  /** 该分类下的图片数量。 */
  imageCount: number;
  /** 该分类下的资源数量。 */
  resourceCount: number;
}

/** 站点首页聚合概览，对应后端 PublicSiteSummary。 */
export interface PublicSiteSummary {
  /** 公开文章总数。 */
  articleCount: number;
  /** 公开图片总数。 */
  imageCount: number;
  /** 公开资源总数。 */
  resourceCount: number;
  /** 公开分类总数。 */
  categoryCount: number;
  /** 最新文章列表。 */
  latestArticles: PublicArticleListItem[];
  /** 精选图片列表。 */
  featuredImages: PublicFileListItem[];
  /** 热门分类列表。 */
  popularCategories: PublicCategory[];
}

/** 聚合搜索结果，对应后端 PublicSearchResult。 */
export interface PublicSearchResult {
  /** 匹配的文章列表。 */
  articles: PublicArticleListItem[];
  /** 匹配的图片列表。 */
  images: PublicFileListItem[];
  /** 匹配的资源列表。 */
  resources: PublicFileListItem[];
}

/** 公开详情响应结构，对应后端 PublicDetailResponse。 */
export interface PublicDetailResponse<T> {
  /** 详情数据。 */
  item: T;
}

/** 后端统一错误响应结构，code 用于映射本地化文案。 */
export interface PublicApiError {
  /** 稳定错误码。 */
  code?: string;
  /** 可记录的错误说明，不直接展示给用户。 */
  error?: string;
}

/** C 端日常内容条目，公开列表和个人列表共用。 */
export interface PublicDailyItem {
  /** 日常唯一标识。 */
  id: number;
  /** 日常正文纯文本。 */
  content: string;
  /** 发布人的公开名称。 */
  authorName: string;
  /** 是否仅发布人可见。 */
  isPrivate: boolean;
  /** 浏览量。 */
  views: number;
  /** 创建时间。 */
  createdAt: string;
  /** 更新时间。 */
  updatedAt: string;
}
