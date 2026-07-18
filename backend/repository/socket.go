package repository

import (
	"strings"
	"time"

	"collector-backend/models"
)

func (s *SQLiteStore) CreateSocketConversation(id, visitorName, tokenHash string) (models.SocketConversation, bool) {
	now := time.Now().UTC()
	visitorName = strings.TrimSpace(visitorName)
	if visitorName == "" {
		visitorName = "访客"
	}
	_, err := s.db.Exec(`
		INSERT INTO socket_conversations(id,visitor_name,visitor_token_hash,status,online,last_seen_at,created_at,updated_at)
		VALUES(?,?,?,'open',1,?,?,?)
	`, id, visitorName, tokenHash, timeText(now), timeText(now), timeText(now))
	if err != nil {
		return models.SocketConversation{}, false
	}
	return s.FindSocketConversation(id)
}

func (s *SQLiteStore) FindSocketConversation(id string) (models.SocketConversation, bool) {
	return scanSocketConversation(s.db.QueryRow(`
		SELECT c.id,c.visitor_name,c.status,c.online,c.last_seen_at,c.created_at,c.updated_at,
			COALESCE((SELECT CASE WHEN m.content<>'' THEN m.content ELSE m.attachment_name END FROM socket_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
			(SELECT COUNT(1) FROM socket_messages m WHERE m.conversation_id=c.id)
		FROM socket_conversations c WHERE c.id=?
	`, strings.TrimSpace(id)))
}

func (s *SQLiteStore) ValidateSocketConversationToken(id, tokenHash string) bool {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM socket_conversations WHERE id=? AND visitor_token_hash=?`, strings.TrimSpace(id), tokenHash).Scan(&count); err != nil {
		return false
	}
	return count == 1
}

func (s *SQLiteStore) SetSocketConversationOnline(id string, online bool) bool {
	now := timeText(time.Now().UTC())
	result, err := s.db.Exec(`UPDATE socket_conversations SET online=?,last_seen_at=?,updated_at=? WHERE id=?`, online, now, now, strings.TrimSpace(id))
	if err != nil {
		return false
	}
	rows, _ := result.RowsAffected()
	return rows == 1
}

func (s *SQLiteStore) ListSocketConversations() []models.SocketConversation {
	rows, err := s.db.Query(`
		SELECT c.id,c.visitor_name,c.status,c.online,c.last_seen_at,c.created_at,c.updated_at,
			COALESCE((SELECT CASE WHEN m.content<>'' THEN m.content ELSE m.attachment_name END FROM socket_messages m WHERE m.conversation_id=c.id ORDER BY m.id DESC LIMIT 1),''),
			(SELECT COUNT(1) FROM socket_messages m WHERE m.conversation_id=c.id)
		FROM socket_conversations c ORDER BY c.online DESC,c.updated_at DESC
	`)
	if err != nil {
		return []models.SocketConversation{}
	}
	defer rows.Close()
	items := []models.SocketConversation{}
	for rows.Next() {
		if item, ok := scanSocketConversation(rows); ok {
			items = append(items, item)
		}
	}
	return items
}

func (s *SQLiteStore) CreateSocketMessage(message models.SocketMessage) (models.SocketMessage, bool) {
	message.ConversationID = strings.TrimSpace(message.ConversationID)
	message.Content = strings.TrimSpace(message.Content)
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}
	result, err := s.db.Exec(`
		INSERT INTO socket_messages(conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)
	`, message.ConversationID, message.SenderType, message.SenderName, message.MessageType, message.Content, message.AttachmentName, message.AttachmentType, message.AttachmentSize, message.AttachmentStorage, timeText(message.CreatedAt))
	if err != nil {
		return models.SocketMessage{}, false
	}
	id, _ := result.LastInsertId()
	message.ID = int(id)
	_, _ = s.db.Exec(`UPDATE socket_conversations SET updated_at=? WHERE id=?`, timeText(message.CreatedAt), message.ConversationID)
	return message, true
}

func (s *SQLiteStore) ListSocketMessages(conversationID string) []models.SocketMessage {
	rows, err := s.db.Query(`
		SELECT id,conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
		FROM socket_messages WHERE conversation_id=? ORDER BY id
	`, strings.TrimSpace(conversationID))
	if err != nil {
		return []models.SocketMessage{}
	}
	defer rows.Close()
	items := []models.SocketMessage{}
	for rows.Next() {
		if item, ok := scanSocketMessage(rows); ok {
			items = append(items, item)
		}
	}
	return items
}

func (s *SQLiteStore) FindSocketMessage(id int) (models.SocketMessage, bool) {
	return scanSocketMessage(s.db.QueryRow(`
		SELECT id,conversation_id,sender_type,sender_name,message_type,content,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
		FROM socket_messages WHERE id=?
	`, id))
}

func scanSocketConversation(row scanner) (models.SocketConversation, bool) {
	var item models.SocketConversation
	var online int
	var lastSeen, created, updated string
	if err := row.Scan(&item.ID, &item.VisitorName, &item.Status, &online, &lastSeen, &created, &updated, &item.LastMessage, &item.MessageCount); err != nil {
		return models.SocketConversation{}, false
	}
	item.Online = online != 0
	item.LastSeenAt = parseTime(lastSeen)
	item.CreatedAt = parseTime(created)
	item.UpdatedAt = parseTime(updated)
	return item, true
}

func scanSocketMessage(row scanner) (models.SocketMessage, bool) {
	var item models.SocketMessage
	var created string
	if err := row.Scan(&item.ID, &item.ConversationID, &item.SenderType, &item.SenderName, &item.MessageType, &item.Content, &item.AttachmentName, &item.AttachmentType, &item.AttachmentSize, &item.AttachmentStorage, &created); err != nil {
		return models.SocketMessage{}, false
	}
	item.CreatedAt = parseTime(created)
	return item, true
}
