package repository

import (
	"database/sql"
	"strings"
	"time"

	"collector-backend/models"
)

func (s *SQLiteStore) RecordVisitorAccess(record models.VisitorAccessRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO visitor_access_logs
		(ip,forwarded_ip,country,region,city,isp,host,method,path,status_code,duration_ms,bytes,user_agent,browser,os,device,referer,accept_language,user_id,user_name,authenticated,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, record.IP, record.ForwardedIP, record.Country, record.Region, record.City, record.ISP, record.Host, record.Method, record.Path,
		record.StatusCode, record.DurationMS, record.Bytes, record.UserAgent, record.Browser, record.OS, record.Device,
		record.Referer, record.AcceptLanguage, nullableVisitorUserID(record.UserID), record.UserName, boolToInt(record.Authenticated), timeText(record.CreatedAt))
	return err
}

func (s *SQLiteStore) PruneVisitorAccessBefore(before time.Time) error {
	_, err := s.db.Exec(`DELETE FROM visitor_access_logs WHERE created_at < ?`, timeText(before))
	return err
}

func (s *SQLiteStore) ListVisitorAnalytics(filter models.VisitorAnalyticsFilter) (models.VisitorAnalyticsResponse, error) {
	where, args := visitorAnalyticsWhere(filter)
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

	if err := s.db.QueryRow(`SELECT COUNT(1) FROM visitor_access_logs WHERE `+where, args...).Scan(&response.Total); err != nil {
		return response, err
	}
	var total, uniqueIPs, authenticated, errorsCount int64
	var average sql.NullFloat64
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

	if err := s.scanVisitorDimensions(&response.Summary.Countries, `
		SELECT CASE WHEN trim(country)='' THEN '未知' ELSE country END AS label, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY label ORDER BY COUNT(1) DESC, label LIMIT 8`, args); err != nil {
		return response, err
	}
	if err := s.scanVisitorDimensions(&response.Summary.Paths, `
		SELECT path, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY path ORDER BY COUNT(1) DESC, path LIMIT 8`, args); err != nil {
		return response, err
	}

	bucketExpression := "substr(created_at,1,10)"
	if filter.Range == "24h" {
		bucketExpression = "substr(created_at,1,13)"
	}
	if err := s.scanVisitorTimeline(&response.Summary.Timeline, `
		SELECT `+bucketExpression+`, COUNT(1)
		FROM visitor_access_logs WHERE `+where+`
		GROUP BY 1 ORDER BY 1`, args); err != nil {
		return response, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	rows, err := s.db.Query(`
		SELECT id,ip,forwarded_ip,country,region,city,isp,host,method,path,status_code,duration_ms,bytes,
		       user_agent,browser,os,device,referer,accept_language,user_id,user_name,authenticated,created_at
		FROM visitor_access_logs WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, filter.PageSize, offset)...)
	if err != nil {
		return response, err
	}
	defer rows.Close()
	for rows.Next() {
		var record models.VisitorAccessRecord
		var userID sql.NullInt64
		var authenticated int
		var created string
		if err := rows.Scan(&record.ID, &record.IP, &record.ForwardedIP, &record.Country, &record.Region, &record.City, &record.ISP,
			&record.Host, &record.Method, &record.Path, &record.StatusCode, &record.DurationMS, &record.Bytes, &record.UserAgent,
			&record.Browser, &record.OS, &record.Device, &record.Referer, &record.AcceptLanguage, &userID, &record.UserName,
			&authenticated, &created); err != nil {
			continue
		}
		if userID.Valid {
			id := int(userID.Int64)
			record.UserID = &id
		}
		record.Authenticated = authenticated != 0
		record.CreatedAt = parseTime(created)
		response.Records = append(response.Records, record)
	}
	return response, rows.Err()
}

func visitorAnalyticsWhere(filter models.VisitorAnalyticsFilter) (string, []any) {
	where := "created_at >= ? AND created_at < ?"
	args := []any{timeText(filter.From), timeText(filter.To)}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
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

func (s *SQLiteStore) scanVisitorDimensions(target *[]models.VisitorAnalyticsDimension, query string, args []any) error {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.VisitorAnalyticsDimension
		if err := rows.Scan(&item.Name, &item.Value); err != nil {
			continue
		}
		*target = append(*target, item)
	}
	return rows.Err()
}

func (s *SQLiteStore) scanVisitorTimeline(target *[]models.VisitorAnalyticsPoint, query string, args []any) error {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.VisitorAnalyticsPoint
		if err := rows.Scan(&item.Label, &item.Value); err != nil {
			continue
		}
		*target = append(*target, item)
	}
	return rows.Err()
}

func nullableVisitorUserID(id *int) any {
	if id == nil {
		return nil
	}
	return *id
}
