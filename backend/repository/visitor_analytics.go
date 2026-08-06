package repository

import (
	"database/sql"
	"strings"
	"time"

	"collector-backend/models"
)

// RecordVisitorAccess 执行对应业务流程。
func (s *SQLiteStore) RecordVisitorAccess(record models.VisitorAccessRecord) error {
	// err 保存当前操作结果以及可能返回的错误状态。
	_, err := s.db.Exec(`
		INSERT INTO visitor_access_logs
		(ip,forwarded_ip,country,region,city,isp,host,method,path,status_code,duration_ms,bytes,user_agent,browser,os,device,referer,accept_language,user_id,user_name,authenticated,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, record.IP, record.ForwardedIP, record.Country, record.Region, record.City, record.ISP, record.Host, record.Method, record.Path,
		record.StatusCode, record.DurationMS, record.Bytes, record.UserAgent, record.Browser, record.OS, record.Device,
		record.Referer, record.AcceptLanguage, nullableVisitorUserID(record.UserID), record.UserName, boolToInt(record.Authenticated), timeText(record.CreatedAt))
	return err
}

// PruneVisitorAccessBefore 实现对应业务逻辑。
func (s *SQLiteStore) PruneVisitorAccessBefore(before time.Time) error {
	// err 保存当前操作结果以及可能返回的错误状态。
	_, err := s.db.Exec(`DELETE FROM visitor_access_logs WHERE created_at < ?`, timeText(before))
	return err
}

// ListVisitorAnalytics 查询并返回对应业务列表。
func (s *SQLiteStore) ListVisitorAnalytics(filter models.VisitorAnalyticsFilter) (models.VisitorAnalyticsResponse, error) {
	// where、args 保存查询条件、调用参数。
	where, args := visitorAnalyticsWhere(filter)
	// response 保存接口响应及其关联状态。
	response := models.VisitorAnalyticsResponse{
		Records:  []models.VisitorAccessRecord{},
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Summary: models.VisitorAnalyticsSummary{
			Countries: []models.VisitorAnalyticsDimension{},
			Paths:     []models.VisitorAnalyticsDimension{},
			Timeline:  []models.VisitorAnalyticsPoint{},
		},
	}

	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM visitor_access_logs WHERE `+where, args...).Scan(&response.Total); err != nil {
		return response, err
	}
	// total 保存总数。
	var total, uniqueIPs, authenticated, errorsCount int64
	// average 保存变量 average。
	var average sql.NullFloat64
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`
		SELECT COUNT(1), COUNT(DISTINCT ip), COALESCE(SUM(authenticated),0),
		COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END),0), AVG(duration_ms)
		FROM visitor_access_logs WHERE `+where, args...).Scan(&total, &uniqueIPs, &authenticated, &errorsCount, &average); err != nil {
		return response, err
	}
	response.Summary.TotalRequests = total
	response.Summary.UniqueIPs = uniqueIPs
	response.Summary.AuthenticatedRequests = authenticated
	response.Summary.ErrorRequests = errorsCount
	if average.Valid {
		response.Summary.AverageDurationMS = int64(average.Float64 + 0.5)
	}

	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.scanVisitorDimensions(&response.Summary.Countries, `
		SELECT CASE WHEN trim(country)='' THEN '未知' ELSE country END AS label, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY label ORDER BY COUNT(1) DESC, label LIMIT 8`, args); err != nil {
		return response, err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.scanVisitorDimensions(&response.Summary.Paths, `
		SELECT path, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY path ORDER BY COUNT(1) DESC, path LIMIT 8`, args); err != nil {
		return response, err
	}

	// bucketExpression 保存变量 bucketExpression。
	bucketExpression := "substr(created_at,1,10)"
	if filter.Range == "24h" {
		bucketExpression = "substr(created_at,1,13)"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.scanVisitorTimeline(&response.Summary.Timeline, `
		SELECT `+bucketExpression+`, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY 1 ORDER BY 1`, args); err != nil {
		return response, err
	}

	// offset 保存偏移量。
	offset := (filter.Page - 1) * filter.PageSize
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT id,ip,forwarded_ip,country,region,city,isp,host,method,path,status_code,duration_ms,bytes,
		       user_agent,browser,os,device,referer,accept_language,user_id,user_name,authenticated,created_at
		FROM visitor_access_logs WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, filter.PageSize, offset)...)
	if err != nil {
		return response, err
	}
	defer rows.Close()
	for rows.Next() {
		// record 保存记录。
		var record models.VisitorAccessRecord
		// userID 保存用户标识。
		var userID sql.NullInt64
		// authenticated 保存变量 authenticated。
		var authenticated int
		// created 保存创建时间。
		var created string
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&record.ID, &record.IP, &record.ForwardedIP, &record.Country, &record.Region, &record.City, &record.ISP,
			&record.Host, &record.Method, &record.Path, &record.StatusCode, &record.DurationMS, &record.Bytes, &record.UserAgent,
			&record.Browser, &record.OS, &record.Device, &record.Referer, &record.AcceptLanguage, &userID, &record.UserName,
			&authenticated, &created); err != nil {
			continue
		}
		if userID.Valid {
			// id 保存标识。
			id := int(userID.Int64)
			record.UserID = &id
		}
		record.Authenticated = authenticated != 0
		record.CreatedAt = parseTime(created)
		response.Records = append(response.Records, record)
	}
	return response, rows.Err()
}

// visitorAnalyticsWhere 实现对应业务逻辑。
func visitorAnalyticsWhere(filter models.VisitorAnalyticsFilter) (string, []any) {
	// where 保存查询条件。
	where := "created_at >= ? AND created_at < ?"
	// args 保存调用参数。
	args := []any{timeText(filter.From), timeText(filter.To)}
	// keyword 保存搜索关键词。
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		// pattern 保存变量 pattern。
		pattern := "%" + keyword + "%"
		where += " AND (ip LIKE ? OR forwarded_ip LIKE ? OR country LIKE ? OR region LIKE ? OR city LIKE ? OR path LIKE ? OR user_agent LIKE ? OR referer LIKE ? OR user_name LIKE ?)"
		for range 9 {
			args = append(args, pattern)
		}
	}
	if filter.StatusCode != nil {
		where += " AND status_code = ?"
		args = append(args, *filter.StatusCode)
	}
	return where, args
}

// scanVisitorDimensions 解析对应业务数据。
func (s *SQLiteStore) scanVisitorDimensions(target *[]models.VisitorAnalyticsDimension, query string, args []any) error {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		// analyticsDimension 保存分析数据。
		var analyticsDimension models.VisitorAnalyticsDimension
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&analyticsDimension.Name, &analyticsDimension.Value); err != nil {
			continue
		}
		*target = append(*target, analyticsDimension)
	}
	return rows.Err()
}

// scanVisitorTimeline 解析对应业务数据。
func (s *SQLiteStore) scanVisitorTimeline(target *[]models.VisitorAnalyticsPoint, query string, args []any) error {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		// timelinePoint 保存变量 timelinePoint。
		var timelinePoint models.VisitorAnalyticsPoint
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&timelinePoint.Label, &timelinePoint.Value); err != nil {
			continue
		}
		*target = append(*target, timelinePoint)
	}
	return rows.Err()
}

// nullableVisitorUserID 实现对应业务逻辑。
func nullableVisitorUserID(id *int) any {
	if id == nil {
		return nil
	}
	return *id
}
