package repository

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"collector-backend/models"
	"collector-backend/utils"
)

// isPublishedArticleSQL 保存公开文章的过滤条件，要求非私密且已发布。
const isPublishedArticleSQL = "is_private=0 AND status='已发布'"

// articlePublicCondition 构建公开文章可见条件，includeR18 为 false 时排除 18R 文章。
func articlePublicCondition(includeR18 bool) string {
	if includeR18 {
		return isPublishedArticleSQL
	}
	return isPublishedArticleSQL + " AND is_18r=0"
}

// scanPublicArticle 从查询行扫描公开文章列表项，并校验公开条件。
func scanPublicArticle(row scanner, coverImage string) (models.PublicArticleListItem, bool) {
	// item 保存扫描得到的公开文章列表项。
	var item models.PublicArticleListItem
	// isPrivate 保存文章私密状态。
	var isPrivate int
	// contentLocale 保存正文实际语言。
	// status 保存文章状态。
	var status string
	var contentLocale string
	// publishedAt 保存首次发布到门户的时间。
	var publishedAt sql.NullString
	// c、up 保存创建与更新时间。
	var c, up string
	// err 保存扫描过程的错误。
	err := row.Scan(&item.ID, &item.Title, &item.Category, &item.Author, &status, &item.Summary, &isPrivate, &publishedAt, &contentLocale, &item.Views, &c, &up)
	if err != nil {
		return models.PublicArticleListItem{}, false
	}
	// 校验文章确实满足公开条件，否则不作为公开内容返回。
	if intToBool(isPrivate) || status != "已发布" {
		return models.PublicArticleListItem{}, false
	}
	item.Slug = slugify(item.Title)
	item.CoverImage = coverImage
	item.ContentLocale = contentLocale
	if contentLocale == "" {
		item.ContentLocale = "zh-CN"
	}
	if publishedAt.Valid {
		item.PublishedAt = parseTime(publishedAt.String)
	} else {
		// 文章不再维护首次发布时间，缺省时回退为更新时间。
		item.PublishedAt = parseTime(up)
	}
	item.UpdatedAt = parseTime(up)
	// 返回校验通过的文章列表项。
	return item, true
}

// articlePublicSelect 保存公开文章列表查询需要扫描的列。
const articlePublicSelect = "a.id,a.title,a.category,a.author,a.status,a.summary,a.is_private,a.portal_published_at,a.content_locale,a.views,a.created_at,a.updated_at"

// ListPublicArticles 返回公开文章列表及分页信息。
func (s *SQLiteStore) ListPublicArticles(keyword, category string, page, pageSize int, includeR18 bool) ([]models.PublicArticleListItem, int, int) {
	// where 保存动态拼接的查询条件。
	where := []string{articlePublicCondition(includeR18)}
	// args 保存查询条件对应的参数。
	args := []interface{}{}
	if keyword != "" {
		where = append(where, "(a.title LIKE ? OR a.summary LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" {
		where = append(where, "a.category=?")
		args = append(args, category)
	}
	// whereSQL 保存 WHERE 子句的最终文本。
	whereSQL := strings.Join(where, " AND ")
	// total 保存符合条件的总条数。
	var total int
	// err 保存计数查询的错误。
	if err := s.db.QueryRow("SELECT COUNT(1) FROM articles a WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return []models.PublicArticleListItem{}, 0, 0
	}
	// offset 保存分页偏移量。
	offset := (page - 1) * pageSize
	// rows、err 保存分页查询结果与查询错误。
	rows, err := s.db.Query("SELECT "+articlePublicSelect+" FROM articles a WHERE "+whereSQL+" ORDER BY a.id DESC LIMIT ? OFFSET ?", append(args, pageSize, offset)...)
	if err != nil {
		return []models.PublicArticleListItem{}, 0, 0
	}
	defer rows.Close()
	// items 保存当前页的文章列表。
	items := []models.PublicArticleListItem{}
	for rows.Next() {
		// item 保存扫描得到的单篇文章。
		item, ok := scanPublicArticle(rows, "")
		if ok {
			items = append(items, item)
		}
	}
	// totalPages 保存总页数。
	totalPages := (total + pageSize - 1) / pageSize
	return items, total, totalPages
}

// FindPublicArticleDetail 返回公开文章详情，并校验公开条件。
func (s *SQLiteStore) FindPublicArticleDetail(id int, includeR18 bool) (models.PublicArticleDetail, bool) {
	// row 保存查询结果行。
	row := s.db.QueryRow("SELECT a.id,a.title,a.category,a.author,a.status,a.summary,a.content,a.is_private,a.portal_published_at,a.content_locale,a.views,a.created_at,a.updated_at FROM articles a WHERE a.id=? AND "+articlePublicCondition(includeR18), id)
	// item 保存扫描得到的公开文章详情。
	var item models.PublicArticleDetail
	// isPrivate 保存文章私密状态。
	var isPrivate int
	// contentLocale 保存正文实际语言。
	var contentLocale string
	// status 保存文章状态。
	var status string
	// publishedAt 保存首次发布到门户的时间。
	var publishedAt sql.NullString
	// c、up 保存创建与更新时间。
	var c, up string
	// err 保存扫描过程的错误。
	err := row.Scan(&item.ID, &item.Title, &item.Category, &item.Author, &status, &item.Summary, &item.Content, &isPrivate, &publishedAt, &contentLocale, &item.Views, &c, &up)
	if err != nil {
		return models.PublicArticleDetail{}, false
	}
	// 校验公开条件，未发布或私密文章不返回。
	if intToBool(isPrivate) || status != "已发布" {
		return models.PublicArticleDetail{}, false
	}
	item.Slug = slugify(item.Title)
	item.ContentLocale = contentLocale
	if contentLocale == "" {
		item.ContentLocale = "zh-CN"
	}
	if publishedAt.Valid {
		item.PublishedAt = parseTime(publishedAt.String)
	} else {
		item.PublishedAt = parseTime(up)
	}
	item.UpdatedAt = parseTime(up)
	return item, true
}

// ListRelatedPublicArticles 返回同分类下的关联文章列表。
func (s *SQLiteStore) ListRelatedPublicArticles(articleID int, category string, limit int, includeR18 bool) []models.PublicArticleListItem {
	// rows、err 保存关联文章查询结果与查询错误。
	rows, err := s.db.Query("SELECT "+articlePublicSelect+" FROM articles a WHERE "+articlePublicCondition(includeR18)+" AND a.id<>? AND a.category=? ORDER BY a.id DESC LIMIT ?", articleID, category, limit)
	if err != nil {
		return []models.PublicArticleListItem{}
	}
	defer rows.Close()
	// items 保存关联文章列表。
	items := []models.PublicArticleListItem{}
	for rows.Next() {
		// item 保存扫描得到的单篇关联文章。
		item, ok := scanPublicArticle(rows, "")
		if ok {
			items = append(items, item)
		}
	}
	return items
}

// listPublicCategoriesFallback 通过聚合统计构造公开分类列表。
func (s *SQLiteStore) listPublicCategoriesFallback(includeR18 bool) []models.PublicCategory {
	// fileCond 保存公开文件的可见条件。
	fileCond := publicFileCondition(includeR18)
	// rows、err 保存分类聚合查询结果与查询错误。
	rows, err := s.db.Query("SELECT article_counts.category, COALESCE(article_counts.cnt,0), COALESCE(image_counts.cnt,0), COALESCE(resource_counts.cnt,0) FROM (" +
		"SELECT category, COUNT(1) AS cnt FROM articles WHERE " + articlePublicCondition(includeR18) + " GROUP BY category" +
		") article_counts LEFT JOIN (" +
		"SELECT category, COUNT(1) AS cnt FROM files WHERE " + fileCond + " AND content_type LIKE 'image/%' GROUP BY category" +
		") image_counts ON image_counts.category=article_counts.category LEFT JOIN (" +
		"SELECT category, COUNT(1) AS cnt FROM files WHERE " + fileCond + " AND content_type NOT LIKE 'image/%' GROUP BY category" +
		") resource_counts ON resource_counts.category=article_counts.category ORDER BY (article_counts.cnt+COALESCE(image_counts.cnt,0)+COALESCE(resource_counts.cnt,0)) DESC, article_counts.category ASC")
	if err != nil {
		return []models.PublicCategory{}
	}
	defer rows.Close()
	// categories 保存聚合得到的分类列表。
	categories := []models.PublicCategory{}
	for rows.Next() {
		// category 保存当前分类。
		var category models.PublicCategory
		// articleCount、imageCount、resourceCount 保存分类下的各类数量。
		var articleCount, imageCount, resourceCount int
		if err := rows.Scan(&category.Name, &articleCount, &imageCount, &resourceCount); err != nil {
			continue
		}
		category.ArticleCount = articleCount
		category.ImageCount = imageCount
		category.ResourceCount = resourceCount
		categories = append(categories, category)
	}
	return categories
}

// ListPublicCategories 返回公开分类列表。
func (s *SQLiteStore) ListPublicCategories(includeR18 bool) []models.PublicCategory {
	return s.listPublicCategoriesFallback(includeR18)
}

// slugify 将文章标题转换为 URL 友好的 slug，保留非 ASCII 字符。
func slugify(title string) string {
	// trimmed 保存去除首尾空白后的标题。
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "article"
	}
	// builder 保存拼接的 slug 结果。
	var builder strings.Builder
	// lastWasDash 记录是否刚写入连接符，避免连续短横线。
	lastWasDash := false
	// r 保存当前遍历的标题字符。
	for _, r := range trimmed {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastWasDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastWasDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastWasDash = true
			}
		default:
			// 非 ASCII 字符保留原样，交由 URL 编码处理。
			builder.WriteRune(r)
			lastWasDash = false
		}
	}
	// slug 保存去除首尾连接符后的最终结果。
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "article"
	}
	return slug
}

// isPublicFileSQL 保存公开文件的过滤条件，要求非私密且未删除。
const isPublicFileSQL = "is_private=0 AND deleted_at IS NULL"

// publicFileCondition 构建公开文件可见条件，includeR18 为 false 时排除 18R 文件。
func publicFileCondition(includeR18 bool) string {
	if includeR18 {
		return isPublicFileSQL
	}
	return isPublicFileSQL + " AND is_18r=0"
}

// scanPublicFile 从查询行扫描公开文件列表项，并校验公开条件。
func scanPublicFile(row scanner) (models.PublicFileListItem, bool) {
	// item 保存扫描得到的公开文件列表项。
	var item models.PublicFileListItem
	// isPrivate 保存文件私密状态。
	var isPrivate int
	// c、up 保存创建与更新时间。
	var c, up string
	// deleted 保存软删除时间，非空表示已删除。
	var deleted sql.NullString
	// encodedTags 保存 SQLite 中的 JSON 标签文本。
	var encodedTags string
	// err 保存扫描过程的错误。
	err := row.Scan(&item.ID, &item.DisplayName, &item.Category, &item.Description, &encodedTags, &item.ContentType, &item.Size, &isPrivate, &item.ImageWidth, &item.ImageHeight, &c, &up, &deleted, &item.LikeCount)
	if err != nil {
		return models.PublicFileListItem{}, false
	}
	// 校验公开条件，私密或已删除的文件不返回。
	if intToBool(isPrivate) || deleted.Valid {
		return models.PublicFileListItem{}, false
	}
	// 文件不再维护首次发布时间，统一使用更新时间。
	item.PublishedAt = parseTime(up)
	item.UpdatedAt = parseTime(up)
	item.Tags = utils.DecodeFileTags(encodedTags)

	// 使用相对公开地址，由 C 端结合基础地址拼接，不暴露存储结构。
	item.PreviewURL = "/api/public/files/" + strconv.Itoa(item.ID) + "/preview"
	item.DownloadURL = "/api/public/files/" + strconv.Itoa(item.ID) + "/download"
	// 图片使用文件名作为替代文本，并生成缩略图地址供 C 端省流量加载。
	if strings.HasPrefix(item.ContentType, "image/") {
		item.AltText = item.DisplayName
		item.ThumbnailURL = "/api/public/files/" + strconv.Itoa(item.ID) + "/thumbnail"
		item.MediumURL = "/api/public/files/" + strconv.Itoa(item.ID) + "/medium"
	}
	return item, true
}

// publicFileSelect 保存公开文件列表查询需要扫描的列。
const publicFileSelect = "f.id,f.display_name,f.category,f.description,f.tags,f.content_type,f.size,f.is_private,f.image_width,f.image_height,f.created_at,f.updated_at,f.deleted_at,(SELECT COUNT(1) FROM public_file_likes pfl WHERE pfl.file_id=f.id)"

// ListPublicFiles 返回公开文件列表及分页信息，isImageOnly 为 true 时仅返回图片。
func (s *SQLiteStore) ListPublicFiles(isImageOnly bool, keyword, category string, page, pageSize int, includeR18 bool) ([]models.PublicFileListItem, int, int) {
	// where 保存动态拼接的查询条件。
	where := []string{publicFileCondition(includeR18)}
	// args 保存查询条件对应的参数。
	args := []interface{}{}
	if isImageOnly {
		where = append(where, "f.content_type LIKE 'image/%'")
	} else {
		where = append(where, "f.content_type NOT LIKE 'image/%'")
	}
	if keyword != "" {
		where = append(where, "(f.display_name LIKE ? OR f.description LIKE ? OR f.original_name LIKE ? OR f.tags LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" {
		where = append(where, "f.category=?")
		args = append(args, category)
	}
	// whereSQL 保存 WHERE 子句的最终文本。
	whereSQL := strings.Join(where, " AND ")
	// total 保存符合条件的总条数。
	var total int
	// err 保存计数查询的错误。
	if err := s.db.QueryRow("SELECT COUNT(1) FROM files f WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return []models.PublicFileListItem{}, 0, 0
	}
	// offset 保存分页偏移量。
	offset := (page - 1) * pageSize
	// rows、err 保存分页查询结果与查询错误。
	rows, err := s.db.Query("SELECT "+publicFileSelect+" FROM files f WHERE "+whereSQL+" ORDER BY f.id DESC LIMIT ? OFFSET ?", append(args, pageSize, offset)...)
	if err != nil {
		return []models.PublicFileListItem{}, 0, 0
	}
	defer rows.Close()
	// items 保存当前页的文件列表。
	items := []models.PublicFileListItem{}
	for rows.Next() {
		// item 保存扫描得到的单个文件。
		item, ok := scanPublicFile(rows)
		if ok {
			items = append(items, item)
		}
	}
	// totalPages 保存总页数。
	totalPages := (total + pageSize - 1) / pageSize
	return items, total, totalPages
}

// FindPublicFile 返回公开文件摘要，并校验公开条件。
func (s *SQLiteStore) FindPublicFile(id int, includeR18 bool) (models.PublicFileListItem, bool) {
	// row 保存查询结果行。
	row := s.db.QueryRow("SELECT "+publicFileSelect+" FROM files f WHERE f.id=? AND "+publicFileCondition(includeR18), id)
	// item 保存扫描得到的公开文件。
	item, ok := scanPublicFile(row)
	return item, ok
}

// FeaturedPublicImages 返回最新的公开图片列表（已废除精选逻辑）。
func (s *SQLiteStore) FeaturedPublicImages(limit int, includeR18 bool) []models.PublicFileListItem {
	// rows、err 保存最新图片查询结果与查询错误。
	rows, err := s.db.Query("SELECT "+publicFileSelect+" FROM files f WHERE "+publicFileCondition(includeR18)+" AND f.content_type LIKE 'image/%' ORDER BY f.id DESC LIMIT ?", limit)
	if err != nil {
		return []models.PublicFileListItem{}
	}
	defer rows.Close()
	// items 保存最新图片列表。
	items := []models.PublicFileListItem{}
	for rows.Next() {
		// item 保存扫描得到的图片。
		item, ok := scanPublicFile(rows)
		if ok {
			items = append(items, item)
		}
	}
	return items
}

// LatestPublicImages 返回最新的公开图片列表。
func (s *SQLiteStore) LatestPublicImages(keyword, category string, page, pageSize int, includeR18 bool) ([]models.PublicFileListItem, int, int) {
	return s.ListPublicFiles(true, keyword, category, page, pageSize, includeR18)
}

// SiteSummary 返回站点首页聚合概览数据。
func (s *SQLiteStore) SiteSummary(includeR18 bool) models.PublicSiteSummary {
	// summary 保存聚合概览结果。
	var summary models.PublicSiteSummary
	// fileCond 保存公开文件的可见条件。
	fileCond := publicFileCondition(includeR18)
	// 统计公开文章总数。
	if err := s.db.QueryRow("SELECT COUNT(1) FROM articles WHERE " + isPublishedArticleSQL).Scan(&summary.ArticleCount); err != nil {
		return summary
	}
	// 统计公开图片总数。
	if err := s.db.QueryRow("SELECT COUNT(1) FROM files WHERE " + fileCond + " AND content_type LIKE 'image/%'").Scan(&summary.ImageCount); err != nil {
		return summary
	}
	// 统计公开资源总数。
	if err := s.db.QueryRow("SELECT COUNT(1) FROM files WHERE " + fileCond + " AND content_type NOT LIKE 'image/%'").Scan(&summary.ResourceCount); err != nil {
		return summary
	}
	// categories 保存热门分类列表。
	categories := s.ListPublicCategories(includeR18)
	summary.CategoryCount = len(categories)
	// 只保留前 6 个热门分类。
	if len(categories) > 6 {
		categories = categories[:6]
	}
	summary.PopularCategories = categories
	// 补充最新 6 篇文章。
	articles, _, _ := s.ListPublicArticles("", "", 1, 6, includeR18)
	summary.LatestArticles = articles
	// 补充最新 8 张图片。
	summary.FeaturedImages = s.FeaturedPublicImages(8, includeR18)
	return summary
}

// SearchPublic 返回聚合搜索结果，按文章、图片、资源分组。
func (s *SQLiteStore) SearchPublic(keyword string, limit int, includeR18 bool) models.PublicSearchResult {
	// result 保存聚合搜索结果。
	var result models.PublicSearchResult
	// articles 保存匹配的文章列表。
	articles, _, _ := s.ListPublicArticles(keyword, "", 1, limit, includeR18)
	result.Articles = articles
	// images 保存匹配的图片列表。
	images, _, _ := s.ListPublicFiles(true, keyword, "", 1, limit, includeR18)
	result.Images = images
	// resources 保存匹配的资源列表。
	resources, _, _ := s.ListPublicFiles(false, keyword, "", 1, limit, includeR18)
	result.Resources = resources
	return result
}

// FindPublicFileStorageName 根据公开文件 ID 返回其物理存储文件名。
func (s *SQLiteStore) FindPublicFileStorageName(id int, includeR18 bool) (string, bool) {
	// storageName 保存查询到的物理存储文件名。
	var storageName string
	// err 保存查询过程的错误。
	err := s.db.QueryRow("SELECT f.storage_name FROM files f WHERE f.id=? AND "+publicFileCondition(includeR18), id).Scan(&storageName)
	if err != nil {
		return "", false
	}
	return storageName, true
}

// GetPublicFileInteraction 返回公开图片的点赞状态与最近一百条评论。
func (s *SQLiteStore) GetPublicFileInteraction(fileID, userID int, includeR18 bool) (models.PublicFileInteraction, bool) {
	// exists 保存目标文件是否仍满足当前访客的公开图片条件。
	var exists int
	if scanErr := s.db.QueryRow("SELECT COUNT(1) FROM files f WHERE f.id=? AND "+publicFileCondition(includeR18)+" AND f.content_type LIKE 'image/%'", fileID).Scan(&exists); scanErr != nil || exists == 0 {
		return models.PublicFileInteraction{}, false
	}

	// interaction 保存图片互动汇总结果。
	interaction := models.PublicFileInteraction{Comments: []models.PublicFileComment{}}
	if countErr := s.db.QueryRow(`SELECT COUNT(1) FROM public_file_likes WHERE file_id=?`, fileID).Scan(&interaction.LikeCount); countErr != nil {
		return models.PublicFileInteraction{}, false
	}
	if userID > 0 {
		// likedCount 保存当前用户与图片之间是否存在点赞关系。
		var likedCount int
		if likedErr := s.db.QueryRow(`SELECT COUNT(1) FROM public_file_likes WHERE file_id=? AND user_id=?`, fileID, userID).Scan(&likedCount); likedErr == nil {
			interaction.LikedByCurrentUser = likedCount > 0
		}
	}

	// rows、queryErr 保存最近一百条评论查询结果与错误。
	rows, queryErr := s.db.Query(`
		SELECT recent.id,COALESCE(NULLIF(u.name,''),u.username),recent.content,recent.created_at
		FROM (
			SELECT id,user_id,content,created_at
			FROM public_file_comments
			WHERE file_id=?
			ORDER BY id DESC
			LIMIT 100
		) recent
		JOIN users u ON u.id=recent.user_id
		ORDER BY recent.id ASC
	`, fileID)
	if queryErr != nil {
		return models.PublicFileInteraction{}, false
	}
	defer rows.Close()
	for rows.Next() {
		// comment 保存当前扫描得到的公开评论。
		var comment models.PublicFileComment
		// createdAt 保存 SQLite 中的评论时间文本。
		var createdAt string
		if scanErr := rows.Scan(&comment.ID, &comment.UserName, &comment.Content, &createdAt); scanErr != nil {
			continue
		}
		comment.CreatedAt = parseTime(createdAt)
		interaction.Comments = append(interaction.Comments, comment)
	}
	return interaction, true
}

// TogglePublicFileLike 切换登录用户对公开图片的唯一点赞关系。
func (s *SQLiteStore) TogglePublicFileLike(fileID, userID int, includeR18 bool) (models.PublicFileInteraction, bool) {
	// publicFile、found 保存目标图片的公开可见性。
	publicFile, found := s.FindPublicFile(fileID, includeR18)
	if !found || !strings.HasPrefix(publicFile.ContentType, "image/") || userID <= 0 {
		return models.PublicFileInteraction{}, false
	}
	// transaction、beginErr 保存点赞切换事务与开启错误。
	transaction, beginErr := s.db.Begin()
	if beginErr != nil {
		return models.PublicFileInteraction{}, false
	}
	defer transaction.Rollback()
	// existingCount 保存当前用户是否已经点赞。
	var existingCount int
	if queryErr := transaction.QueryRow(`SELECT COUNT(1) FROM public_file_likes WHERE file_id=? AND user_id=?`, fileID, userID).Scan(&existingCount); queryErr != nil {
		return models.PublicFileInteraction{}, false
	}
	if existingCount > 0 {
		if _, deleteErr := transaction.Exec(`DELETE FROM public_file_likes WHERE file_id=? AND user_id=?`, fileID, userID); deleteErr != nil {
			return models.PublicFileInteraction{}, false
		}
	} else {
		if _, insertErr := transaction.Exec(`INSERT INTO public_file_likes(file_id,user_id,created_at) VALUES(?,?,?)`, fileID, userID, timeText(time.Now().UTC())); insertErr != nil {
			return models.PublicFileInteraction{}, false
		}
	}
	if commitErr := transaction.Commit(); commitErr != nil {
		return models.PublicFileInteraction{}, false
	}
	return s.GetPublicFileInteraction(fileID, userID, includeR18)
}

// CreatePublicFileComment 保存登录用户对公开图片发送的纯文本评论。
func (s *SQLiteStore) CreatePublicFileComment(fileID, userID int, content string, includeR18 bool) (models.PublicFileComment, bool) {
	// publicFile、found 保存目标图片的公开可见性。
	publicFile, found := s.FindPublicFile(fileID, includeR18)
	if !found || !strings.HasPrefix(publicFile.ContentType, "image/") || userID <= 0 {
		return models.PublicFileComment{}, false
	}
	// createdAt 保存评论统一使用的 UTC 时间。
	createdAt := time.Now().UTC()
	// result、insertErr 保存评论写入结果与错误。
	result, insertErr := s.db.Exec(`INSERT INTO public_file_comments(file_id,user_id,content,created_at) VALUES(?,?,?,?)`, fileID, userID, content, timeText(createdAt))
	if insertErr != nil {
		return models.PublicFileComment{}, false
	}
	// commentID 保存新评论的自增编号。
	commentID, idErr := result.LastInsertId()
	if idErr != nil {
		return models.PublicFileComment{}, false
	}
	// comment 保存包含作者展示名称的新评论。
	var comment models.PublicFileComment
	// storedCreatedAt 保存数据库返回的评论时间文本。
	var storedCreatedAt string
	if queryErr := s.db.QueryRow(`
		SELECT c.id,COALESCE(NULLIF(u.name,''),u.username),c.content,c.created_at
		FROM public_file_comments c
		JOIN users u ON u.id=c.user_id
		WHERE c.id=?
	`, commentID).Scan(&comment.ID, &comment.UserName, &comment.Content, &storedCreatedAt); queryErr != nil {
		return models.PublicFileComment{}, false
	}
	comment.CreatedAt = parseTime(storedCreatedAt)
	return comment, true
}

// AppendPublicFileTag 为登录用户可见的公开图片追加标签，并保留已有标签与统一数量边界。
func (s *SQLiteStore) AppendPublicFileTag(fileID, userID int, tag string, includeR18 bool) ([]string, bool, bool) {
	if userID <= 0 {
		return []string{}, false, false
	}
	// transaction、beginErr 保存标签合并事务与开启错误。
	transaction, beginErr := s.db.Begin()
	if beginErr != nil {
		return []string{}, false, false
	}
	defer transaction.Rollback()
	// encodedTags 保存目标公开图片当前的 JSON 标签文本。
	var encodedTags string
	if queryErr := transaction.QueryRow(
		"SELECT f.tags FROM files f WHERE f.id=? AND "+publicFileCondition(includeR18)+" AND f.content_type LIKE 'image/%'",
		fileID,
	).Scan(&encodedTags); queryErr != nil {
		return []string{}, false, false
	}
	// existingTags 保存管理端与门户端此前已经写入的规范标签。
	existingTags := utils.DecodeFileTags(encodedTags)
	// mergedTags 保存追加候选标签并再次执行去重和数量限制后的权威列表。
	mergedTags := utils.NormalizeFileTags(append(append([]string{}, existingTags...), tag))
	// added 表示合并后标签数量确实增加，重复或已满时不执行写入。
	added := len(mergedTags) > len(existingTags)
	if !added {
		return existingTags, false, true
	}
	// updateResult、updateErr 保存标签元数据更新结果与错误。
	updateResult, updateErr := transaction.Exec(
		`UPDATE files SET tags=?,updated_at=? WHERE id=? AND deleted_at IS NULL`,
		utils.EncodeFileTags(mergedTags), timeText(time.Now().UTC()), fileID,
	)
	if updateErr != nil {
		return []string{}, false, false
	}
	// affectedRows、affectedErr 保存更新命中的图片数量与读取错误。
	affectedRows, affectedErr := updateResult.RowsAffected()
	if affectedErr != nil || affectedRows == 0 {
		return []string{}, false, false
	}
	if commitErr := transaction.Commit(); commitErr != nil {
		return []string{}, false, false
	}
	return mergedTags, true, true
}
