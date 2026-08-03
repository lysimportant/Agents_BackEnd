package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"collector-backend/models"
)

func (s *SQLiteStore) ListInternalChatUsers(currentUserID int) ([]models.InternalChatUser, error) {
	rows, err := s.db.Query(`
		SELECT id, username, name, department
		FROM users
		WHERE id<>?
		ORDER BY name, username
	`, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.InternalChatUser, 0)
	for rows.Next() {
		var user models.InternalChatUser
		if err := rows.Scan(&user.ID, &user.Username, &user.Name, &user.Department); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *SQLiteStore) ListInternalChatMessages(currentUserID, peerID, afterID int) ([]models.InternalChatMessage, error) {
	query := `
		SELECT message.id, message.sender_id, sender.name, message.recipient_id,
		       COALESCE(recipient.name, ''), message.content, message.created_at
		FROM internal_chat_messages AS message
		JOIN users AS sender ON sender.id=message.sender_id
		LEFT JOIN users AS recipient ON recipient.id=message.recipient_id
		WHERE message.id>?
	`
	args := []any{afterID}
	if peerID == 0 {
		query += ` AND message.recipient_id IS NULL`
	} else {
		query += ` AND ((message.sender_id=? AND message.recipient_id=?) OR (message.sender_id=? AND message.recipient_id=?))`
		args = append(args, currentUserID, peerID, peerID, currentUserID)
	}
	query += ` ORDER BY message.id ASC LIMIT 500`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]models.InternalChatMessage, 0)
	for rows.Next() {
		var message models.InternalChatMessage
		var recipientID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&message.ID, &message.SenderID, &message.SenderName, &recipientID, &message.RecipientName, &message.Content, &createdAt); err != nil {
			return nil, err
		}
		if recipientID.Valid {
			value := int(recipientID.Int64)
			message.RecipientID = &value
		}
		message.CreatedAt = parseTime(createdAt)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *SQLiteStore) CreateInternalChatMessage(senderID int, recipientID *int, content string, now time.Time) (models.InternalChatMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return models.InternalChatMessage{}, errors.New("消息内容不能为空")
	}
	if len([]rune(content)) > 2000 {
		return models.InternalChatMessage{}, errors.New("消息内容不能超过 2000 个字符")
	}
	if recipientID != nil {
		if *recipientID == senderID {
			return models.InternalChatMessage{}, errors.New("不能向自己发起私聊")
		}
		var available bool
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id=?)`, *recipientID).Scan(&available); err != nil {
			return models.InternalChatMessage{}, err
		}
		if !available {
			return models.InternalChatMessage{}, errors.New("私聊用户不存在或不可用")
		}
	}

	result, err := s.db.Exec(`INSERT INTO internal_chat_messages(sender_id,recipient_id,content,created_at) VALUES(?,?,?,?)`, senderID, recipientID, content, timeText(now))
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	messages, err := s.listInternalChatMessagesByID(int(id))
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	if len(messages) != 1 {
		return models.InternalChatMessage{}, errors.New("消息创建后读取失败")
	}
	return messages[0], nil
}

func (s *SQLiteStore) listInternalChatMessagesByID(id int) ([]models.InternalChatMessage, error) {
	rows, err := s.db.Query(`
		SELECT message.id, message.sender_id, sender.name, message.recipient_id,
		       COALESCE(recipient.name, ''), message.content, message.created_at
		FROM internal_chat_messages AS message
		JOIN users AS sender ON sender.id=message.sender_id
		LEFT JOIN users AS recipient ON recipient.id=message.recipient_id
		WHERE message.id=?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]models.InternalChatMessage, 0, 1)
	for rows.Next() {
		var message models.InternalChatMessage
		var recipient sql.NullInt64
		var createdAt string
		if err := rows.Scan(&message.ID, &message.SenderID, &message.SenderName, &recipient, &message.RecipientName, &message.Content, &createdAt); err != nil {
			return nil, err
		}
		if recipient.Valid {
			value := int(recipient.Int64)
			message.RecipientID = &value
		}
		message.CreatedAt = parseTime(createdAt)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
