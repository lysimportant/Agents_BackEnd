package repository

import (
	"strings"
	"time"

	"collector-backend/models"
)

// dailySelectColumns 定义日常列表和详情统一使用的扫描顺序。
const dailySelectColumns = `d.id,d.content,d.owner_id,COALESCE(NULLIF(u.name,''),u.username),d.is_private,d.views,d.cover_file_id,(SELECT COUNT(1) FROM daily_likes dl WHERE dl.daily_id=d.id),d.created_at,d.updated_at`

// scanDaily 从查询行扫描日常记录，并转换 SQLite 的时间和布尔值。
func scanDaily(row scanner) (models.Daily, bool) {
	// daily 保存扫描后的日常内容。
	var daily models.Daily
	// isPrivate 保存 SQLite 中的整数隐私标记。
	var isPrivate int
	// createdAt、updatedAt 保存 SQLite 中的时间文本。
	var createdAt, updatedAt string
	if err := row.Scan(&daily.ID, &daily.Content, &daily.OwnerID, &daily.AuthorName, &isPrivate, &daily.Views, &daily.CoverFileID, &daily.LikeCount, &createdAt, &updatedAt); err != nil {
		return models.Daily{}, false
	}
	daily.IsPrivate = intToBool(isPrivate)
	daily.OwnerName = daily.AuthorName
	daily.CreatedAt = parseTime(createdAt)
	daily.UpdatedAt = parseTime(updatedAt)
	return daily, true
}

// ListPublicDailies 返回当前访客可见的日常列表，私密内容仅对其发布人开放。
func (s *SQLiteStore) ListPublicDailies(userID int, keyword string, page, pageSize int) ([]models.Daily, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 24
	}
	if pageSize > 50 {
		pageSize = 50
	}
	// whereSQL 保证匿名访客只能读取公开日常，登录用户可额外读取自己的私密日常。
	whereSQL := `(d.is_private=0 OR (d.is_private<>0 AND d.owner_id=? AND ?>0))`
	args := []any{userID, userID}
	if strings.TrimSpace(keyword) != "" {
		whereSQL += ` AND (d.content LIKE ? OR COALESCE(NULLIF(u.name,''),u.username) LIKE ?)`
		pattern := "%" + strings.TrimSpace(keyword) + "%"
		args = append(args, pattern, pattern)
	}
	// total 保存满足当前访客可见条件的日常总数。
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM dailies d JOIN users u ON u.id=d.owner_id WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return []models.Daily{}, 0, 0
	}
	// offset 保存当前页对应的 SQLite 偏移量。
	offset := (page - 1) * pageSize
	rows, err := s.db.Query(`SELECT `+dailySelectColumns+` FROM dailies d JOIN users u ON u.id=d.owner_id WHERE `+whereSQL+` ORDER BY d.created_at DESC,d.id DESC LIMIT ? OFFSET ?`, append(args, pageSize, offset)...)
	if err != nil {
		return []models.Daily{}, 0, 0
	}
	defer rows.Close()
	// dailies 保存当前页日常记录。
	dailies := make([]models.Daily, 0)
	for rows.Next() {
		if daily, ok := scanDaily(rows); ok {
			dailies = append(dailies, daily)
		}
	}
	totalPages := (total + pageSize - 1) / pageSize
	return dailies, total, totalPages
}

// FindPublicDaily 返回当前访客可见的日常详情并原子增加浏览量。
func (s *SQLiteStore) FindPublicDaily(id, userID int) (models.Daily, bool) {
	if id <= 0 {
		return models.Daily{}, false
	}
	transaction, err := s.db.Begin()
	if err != nil {
		return models.Daily{}, false
	}
	defer transaction.Rollback()
	// result 保存仅在访客有权查看时命中的浏览量更新结果。
	result, err := transaction.Exec(`UPDATE dailies SET views=views+1 WHERE id=? AND (is_private=0 OR (is_private<>0 AND owner_id=? AND ?>0))`, id, userID, userID)
	if err != nil {
		return models.Daily{}, false
	}
	// affected 保存浏览量更新命中的记录数，用于统一隐藏不存在和无权访问的日常。
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return models.Daily{}, false
	}
	daily, ok := scanDaily(transaction.QueryRow(`SELECT `+dailySelectColumns+` FROM dailies d JOIN users u ON u.id=d.owner_id WHERE d.id=?`, id))
	if !ok {
		return models.Daily{}, false
	}
	if err := transaction.Commit(); err != nil {
		return models.Daily{}, false
	}
	return daily, true
}

// CreateDaily 写入一条由登录用户拥有的已发布日常；正文由 handler 负责白名单清洗。
func (s *SQLiteStore) CreateDaily(ownerID int, request models.DailyRequest) (models.Daily, bool) {
	if ownerID <= 0 || strings.TrimSpace(request.Content) == "" {
		return models.Daily{}, false
	}
	now := time.Now().UTC()
	coverFileID := request.CoverFileID
	if coverFileID > 0 {
		var valid int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM files WHERE id=? AND is_private=0 AND deleted_at IS NULL AND is_18r=0 AND content_type LIKE 'image/%'`, coverFileID).Scan(&valid); err != nil || valid == 0 {
			return models.Daily{}, false
		}
	} else {
		// 未指定封面时仅从匿名可见的公开图片中随机选择，避免私密或 18R 图片泄露。
		_ = s.db.QueryRow(`SELECT id FROM files WHERE is_private=0 AND deleted_at IS NULL AND is_18r=0 AND content_type LIKE 'image/%' ORDER BY RANDOM() LIMIT 1`).Scan(&coverFileID)
	}
	result, err := s.db.Exec(`INSERT INTO dailies(content,owner_id,is_private,views,cover_file_id,created_at,updated_at) VALUES(?,?,?,0,?,?,?)`, strings.TrimSpace(request.Content), ownerID, boolToInt(request.IsPrivate), coverFileID, timeText(now), timeText(now))
	if err != nil {
		return models.Daily{}, false
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Daily{}, false
	}
	return s.findDailyByID(int(id))
}

// findDailyByID 返回刚写入的日常及发布人展示名称，不执行浏览量累加。
func (s *SQLiteStore) findDailyByID(id int) (models.Daily, bool) {
	return scanDaily(s.db.QueryRow(`SELECT `+dailySelectColumns+` FROM dailies d JOIN users u ON u.id=d.owner_id WHERE d.id=?`, id))
}

// GetPublicDailyInteraction 返回当前访客可见日常的点赞状态与最近一百条评论，匿名请求允许读取。
func (s *SQLiteStore) GetPublicDailyInteraction(dailyID, userID int) (models.PublicDailyInteraction, bool) {
	var visible int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM dailies WHERE id=? AND (is_private=0 OR (is_private<>0 AND owner_id=? AND ?>0))`, dailyID, userID, userID).Scan(&visible); err != nil || visible == 0 {
		return models.PublicDailyInteraction{}, false
	}
	interaction := models.PublicDailyInteraction{Comments: []models.PublicDailyComment{}}
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM daily_likes WHERE daily_id=?`, dailyID).Scan(&interaction.LikeCount); err != nil {
		return models.PublicDailyInteraction{}, false
	}
	if userID > 0 {
		var liked int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM daily_likes WHERE daily_id=? AND user_id=?`, dailyID, userID).Scan(&liked); err == nil {
			interaction.LikedByCurrentUser = liked > 0
		}
	}
	rows, err := s.db.Query(`
		SELECT recent.id,COALESCE(NULLIF(u.name,''),u.username),recent.content,recent.created_at
		FROM (SELECT id,user_id,content,created_at FROM daily_comments WHERE daily_id=? ORDER BY id DESC LIMIT 100) recent
		JOIN users u ON u.id=recent.user_id ORDER BY recent.id ASC
	`, dailyID)
	if err != nil {
		return models.PublicDailyInteraction{}, false
	}
	defer rows.Close()
	for rows.Next() {
		var comment models.PublicDailyComment
		var createdAt string
		if err := rows.Scan(&comment.ID, &comment.UserName, &comment.Content, &createdAt); err == nil {
			comment.CreatedAt = parseTime(createdAt)
			interaction.Comments = append(interaction.Comments, comment)
		}
	}
	return interaction, true
}

// TogglePublicDailyLike 切换登录用户对日常的唯一点赞关系。
func (s *SQLiteStore) TogglePublicDailyLike(dailyID, userID int) (models.PublicDailyInteraction, bool) {
	if userID <= 0 {
		return models.PublicDailyInteraction{}, false
	}
	var visible int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM dailies WHERE id=? AND (is_private=0 OR (is_private<>0 AND owner_id=? AND ?>0))`, dailyID, userID, userID).Scan(&visible); err != nil || visible == 0 {
		return models.PublicDailyInteraction{}, false
	}
	transaction, err := s.db.Begin()
	if err != nil {
		return models.PublicDailyInteraction{}, false
	}
	defer transaction.Rollback()
	var existing int
	if err := transaction.QueryRow(`SELECT COUNT(1) FROM daily_likes WHERE daily_id=? AND user_id=?`, dailyID, userID).Scan(&existing); err != nil {
		return models.PublicDailyInteraction{}, false
	}
	if existing > 0 {
		if _, err := transaction.Exec(`DELETE FROM daily_likes WHERE daily_id=? AND user_id=?`, dailyID, userID); err != nil {
			return models.PublicDailyInteraction{}, false
		}
	} else if _, err := transaction.Exec(`INSERT INTO daily_likes(daily_id,user_id,created_at) VALUES(?,?,?)`, dailyID, userID, timeText(time.Now().UTC())); err != nil {
		return models.PublicDailyInteraction{}, false
	}
	if err := transaction.Commit(); err != nil {
		return models.PublicDailyInteraction{}, false
	}
	return s.GetPublicDailyInteraction(dailyID, userID)
}

// CreatePublicDailyComment 保存登录用户对日常发送的纯文本评论。
func (s *SQLiteStore) CreatePublicDailyComment(dailyID, userID int, commentText string) (models.PublicDailyComment, bool) {
	if userID <= 0 {
		return models.PublicDailyComment{}, false
	}
	var visible int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM dailies WHERE id=? AND (is_private=0 OR (is_private<>0 AND owner_id=? AND ?>0))`, dailyID, userID, userID).Scan(&visible); err != nil || visible == 0 {
		return models.PublicDailyComment{}, false
	}
	createdAt := time.Now().UTC()
	result, err := s.db.Exec(`INSERT INTO daily_comments(daily_id,user_id,content,created_at) VALUES(?,?,?,?)`, dailyID, userID, commentText, timeText(createdAt))
	if err != nil {
		return models.PublicDailyComment{}, false
	}
	commentID, err := result.LastInsertId()
	if err != nil {
		return models.PublicDailyComment{}, false
	}
	var comment models.PublicDailyComment
	var storedCreatedAt string
	if err := s.db.QueryRow(`SELECT c.id,COALESCE(NULLIF(u.name,''),u.username),c.content,c.created_at FROM daily_comments c JOIN users u ON u.id=c.user_id WHERE c.id=?`, commentID).Scan(&comment.ID, &comment.UserName, &comment.Content, &storedCreatedAt); err != nil {
		return models.PublicDailyComment{}, false
	}
	comment.CreatedAt = parseTime(storedCreatedAt)
	return comment, true
}
