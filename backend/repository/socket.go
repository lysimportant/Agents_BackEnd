package repository

import (
	"strings"
	"time"

	"collector-backend/models"
)

// CreateSocketConversation 创建或追加对应业务记录。
func (s *SQLiteStore) CreateSocketConversation(id, visitorName, tokenHash string) (models.SocketConversation, bool) {
	// now 保存当前时间。
	now := time.Now().UTC()
	visitorName = strings.TrimSpace(visitorName)
	if visitorName == "" {
		visitorName = "访客"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	_, err := s.db.Exec(`
		INSERT INTO socket_conversations(id,visitor_name,visitor_token_hash,status,online,last_seen_at,created_at,updated_at)
		VALUES(?,?,?,'open',1,?,?,?)
	`, id, visitorName, tokenHash, timeText(now), timeText(now), timeText(now))
	if err != nil {
		return models.SocketConversation{}, false
	}
	return s.FindSocketConversation(id)
}

// FindSocketConversation 获取对应业务记录。
func (s *SQLiteStore) FindSocketConversation(id string) (models.SocketConversation, bool) {
	return scanSocketConversation(s.db.QueryRow(`
		SELECT c.id,c.visitor_name,c.title,c.status,c.online,c.last_seen_at,c.created_at,c.updated_at,
			COALESCE((SELECT CASE WHEN m.content<>'' THEN m.content ELSE m.attachment_name END FROM socket_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
			(SELECT COUNT(1) FROM socket_messages m WHERE m.conversation_id=c.id)
		FROM socket_conversations c WHERE c.id=?
	`, strings.TrimSpace(id)))
}

// ValidateSocketConversationToken 校验对应业务条件。
func (s *SQLiteStore) ValidateSocketConversationToken(id, tokenHash string) bool {
	// count 保存数量。
	var count int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM socket_conversations WHERE id=? AND visitor_token_hash=? AND status='open'`, strings.TrimSpace(id), tokenHash).Scan(&count); err != nil {
		return false
	}
	return count == 1
}

// CloseSocketConversation 删除或清理对应业务记录。
func (s *SQLiteStore) CloseSocketConversation(id string) (models.SocketConversation, bool) {
	// now 保存当前时间。
	now := timeText(time.Now().UTC())
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`UPDATE socket_conversations SET status='closed',online=0,last_seen_at=?,updated_at=? WHERE id=? AND status='open'`, now, now, strings.TrimSpace(id))
	if err != nil {
		return models.SocketConversation{}, false
	}
	// rows 保存数据库查询结果游标及其错误状态。
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return models.SocketConversation{}, false
	}
	return s.FindSocketConversation(id)
}

// SetSocketConversationOnline 更新并保存对应业务状态。
func (s *SQLiteStore) SetSocketConversationOnline(id string, online bool) bool {
	// now 保存当前时间。
	now := timeText(time.Now().UTC())
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`UPDATE socket_conversations SET online=?,last_seen_at=?,updated_at=? WHERE id=? AND status='open'`, online, now, now, strings.TrimSpace(id))
	if err != nil {
		return false
	}
	// rows 保存数据库查询结果游标及其错误状态。
	rows, _ := result.RowsAffected()
	return rows == 1
}

// ListSocketConversations 查询并返回对应业务列表。
func (s *SQLiteStore) ListSocketConversations() []models.SocketConversation {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT c.id,c.visitor_name,c.title,c.status,c.online,c.last_seen_at,c.created_at,c.updated_at,
			COALESCE((SELECT CASE WHEN m.content<>'' THEN m.content ELSE m.attachment_name END FROM socket_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
			(SELECT COUNT(1) FROM socket_messages m WHERE m.conversation_id=c.id)
		FROM socket_conversations c WHERE c.status<>'deleted' ORDER BY c.online DESC,c.updated_at DESC
	`)
	if err != nil {
		return []models.SocketConversation{}
	}
	defer rows.Close()
	// items 保存当前条目。
	items := []models.SocketConversation{}
	for rows.Next() {
		// item、ok 保存业务值及其是否存在或处理成功的标记。
		if item, ok := scanSocketConversation(rows); ok {
			items = append(items, item)
		}
	}
	return items
}

// SetSocketConversationTitle 更新并保存对应业务状态。
func (s *SQLiteStore) SetSocketConversationTitle(id, title string, onlyIfEmpty bool) (models.SocketConversation, bool) {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	if id == "" || title == "" {
		return models.SocketConversation{}, false
	}
	// now 保存当前时间。
	now := timeText(time.Now().UTC())
	// query 保存查询条件。
	query := `UPDATE socket_conversations SET title=?,updated_at=? WHERE id=? AND status<>'deleted'`
	if onlyIfEmpty {
		query += ` AND trim(title)=''`
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(query, title, now, id)
	if err != nil {
		return models.SocketConversation{}, false
	}
	// rows 保存数据库查询结果游标及其错误状态。
	rows, _ := result.RowsAffected()
	if rows == 0 && onlyIfEmpty {
		return s.FindSocketConversation(id)
	}
	if rows != 1 {
		return models.SocketConversation{}, false
	}
	return s.FindSocketConversation(id)
}

// SoftDeleteSocketConversation 实现对应业务逻辑。
func (s *SQLiteStore) SoftDeleteSocketConversation(id string) bool {
	// now 保存当前时间。
	now := timeText(time.Now().UTC())
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`UPDATE socket_conversations SET status='deleted',online=0,last_seen_at=?,updated_at=? WHERE id=? AND status<>'deleted'`, now, now, strings.TrimSpace(id))
	if err != nil {
		return false
	}
	// rows 保存数据库查询结果游标及其错误状态。
	rows, _ := result.RowsAffected()
	return rows == 1
}

// CreateSocketMessage 创建或追加对应业务记录。
func (s *SQLiteStore) CreateSocketMessage(message models.SocketMessage) (models.SocketMessage, bool) {
	message.ConversationID = strings.TrimSpace(message.ConversationID)
	message.Content = strings.TrimSpace(message.Content)
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`
		INSERT INTO socket_messages(conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)
	`, message.ConversationID, message.SenderType, message.SenderName, message.MessageType, message.Content, message.AttachmentName, message.AttachmentType, message.AttachmentSize, message.AttachmentStorage, timeText(message.CreatedAt))
	if err != nil {
		return models.SocketMessage{}, false
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	message.ID = int(id)
	_, _ = s.db.Exec(`UPDATE socket_conversations SET updated_at=? WHERE id=?`, timeText(message.CreatedAt), message.ConversationID)
	return message, true
}

// ListSocketMessages 查询并返回对应业务列表。
func (s *SQLiteStore) ListSocketMessages(conversationID string) []models.SocketMessage {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT id,conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
		FROM socket_messages WHERE conversation_id=? ORDER BY id
	`, strings.TrimSpace(conversationID))
	if err != nil {
		return []models.SocketMessage{}
	}
	defer rows.Close()
	// items 保存当前条目。
	items := []models.SocketMessage{}
	for rows.Next() {
		// item、ok 保存业务值及其是否存在或处理成功的标记。
		if item, ok := scanSocketMessage(rows); ok {
			items = append(items, item)
		}
	}
	return items
}

// FindSocketMessage 获取对应业务记录。
func (s *SQLiteStore) FindSocketMessage(id int) (models.SocketMessage, bool) {
	return scanSocketMessage(s.db.QueryRow(`
		SELECT id,conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
		FROM socket_messages WHERE id=?
	`, id))
}

// scanSocketConversation 解析对应业务数据。
func scanSocketConversation(row scanner) (models.SocketConversation, bool) {
	// conversation 保存会话。
	var conversation models.SocketConversation
	// online 保存在线状态。
	var online int
	// lastSeen 保存已处理集合。
	var lastSeen, created, updated string
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := row.Scan(&conversation.ID, &conversation.VisitorName, &conversation.Title, &conversation.Status, &online, &lastSeen, &created, &updated, &conversation.LastMessage, &conversation.MessageCount); err != nil {
		return models.SocketConversation{}, false
	}
	conversation.Online = online != 0
	conversation.LastSeenAt = parseTime(lastSeen)
	conversation.CreatedAt = parseTime(created)
	conversation.UpdatedAt = parseTime(updated)
	return conversation, true
}

// scanSocketMessage 解析对应业务数据。
func scanSocketMessage(row scanner) (models.SocketMessage, bool) {
	// message 保存消息。
	var message models.SocketMessage
	// created 保存创建时间。
	var created string
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := row.Scan(&message.ID, &message.ConversationID, &message.SenderType, &message.SenderName, &message.MessageType, &message.Content, &message.AttachmentName, &message.AttachmentType, &message.AttachmentSize, &message.AttachmentStorage, &created); err != nil {
		return models.SocketMessage{}, false
	}
	message.CreatedAt = parseTime(created)
	return message, true
}
