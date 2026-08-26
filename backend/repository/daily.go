package repository

import (
	"strings"
	"time"

	"collector-backend/models"
)

// dailySelectColumns 定义日常列表和详情统一使用的扫描顺序。
const dailySelectColumns = `d.id,d.content,d.owner_id,COALESCE(NULLIF(u.name,''),u.username),d.is_private,d.views,d.created_at,d.updated_at`

// scanDaily 从查询行扫描日常记录，并转换 SQLite 的时间和布尔值。
func scanDaily(row scanner) (models.Daily, bool) {
	// daily 保存扫描后的日常内容。
	var daily models.Daily
	// isPrivate 保存 SQLite 中的整数隐私标记。
	var isPrivate int
	// createdAt、updatedAt 保存 SQLite 中的时间文本。
	var createdAt, updatedAt string
	if err := row.Scan(&daily.ID, &daily.Content, &daily.OwnerID, &daily.AuthorName, &isPrivate, &daily.Views, &createdAt, &updatedAt); err != nil {
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

// CreateDaily 写入一条由登录用户拥有的已发布日常。
func (s *SQLiteStore) CreateDaily(ownerID int, request models.DailyRequest) (models.Daily, bool) {
	if ownerID <= 0 || strings.TrimSpace(request.Content) == "" {
		return models.Daily{}, false
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(`INSERT INTO dailies(content,owner_id,is_private,views,created_at,updated_at) VALUES(?,?,?,0,?,?)`, strings.TrimSpace(request.Content), ownerID, boolToInt(request.IsPrivate), timeText(now), timeText(now))
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
