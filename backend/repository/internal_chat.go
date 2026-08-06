package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"collector-backend/models"
)

const maxInternalChatAttachmentsPerMessage = 10

func (s *SQLiteStore) ListInternalChatUsers(currentUserID int) ([]models.InternalChatUser, error) {
	rows, err := s.db.Query(`
		SELECT id, username, name, department
		FROM users
		WHERE id<>? AND status<>'停用' AND can_login=1
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
	} else if peerID == -1 {
		query += ` AND (message.recipient_id IS NULL OR message.sender_id=? OR message.recipient_id=?)`
		args = append(args, currentUserID, currentUserID)
	} else {
		query += ` AND ((message.sender_id=? AND message.recipient_id=?) OR (message.sender_id=? AND message.recipient_id=?))`
		args = append(args, currentUserID, peerID, peerID, currentUserID)
	}
	query += ` ORDER BY message.id ASC LIMIT 500`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	messages, err := scanInternalChatMessages(rows)
	if err != nil {
		return nil, err
	}
	if err := s.loadInternalChatAttachments(messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *SQLiteStore) CreateInternalChatAttachment(ownerID int, originalName, storedName, mimeType string, size int64, isImage bool, now time.Time) (models.InternalChatAttachment, error) {
	result, err := s.db.Exec(`
		INSERT INTO internal_chat_attachments(owner_id,original_name,stored_name,mime_type,size,is_image,created_at)
		VALUES(?,?,?,?,?,?,?)
	`, ownerID, originalName, storedName, mimeType, size, isImage, timeText(now))
	if err != nil {
		return models.InternalChatAttachment{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.InternalChatAttachment{}, err
	}
	attachment, found, err := s.FindInternalChatAttachment(int(id))
	if err != nil {
		return models.InternalChatAttachment{}, err
	}
	if !found {
		return models.InternalChatAttachment{}, errors.New("附件创建后读取失败")
	}
	return attachment, nil
}

func (s *SQLiteStore) FindInternalChatAttachment(id int) (models.InternalChatAttachment, bool, error) {
	var attachment models.InternalChatAttachment
	var messageID sql.NullInt64
	var isImage bool
	var createdAt string
	err := s.db.QueryRow(`
		SELECT id,message_id,owner_id,original_name,stored_name,mime_type,size,is_image,created_at
		FROM internal_chat_attachments WHERE id=?
	`, id).Scan(
		&attachment.ID, &messageID, &attachment.OwnerID, &attachment.OriginalName,
		&attachment.StoredName, &attachment.MimeType, &attachment.Size, &isImage, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.InternalChatAttachment{}, false, nil
	}
	if err != nil {
		return models.InternalChatAttachment{}, false, err
	}
	if messageID.Valid {
		value := int(messageID.Int64)
		attachment.MessageID = &value
	}
	attachment.IsImage = isImage
	attachment.CreatedAt = parseTime(createdAt)
	setInternalChatAttachmentURLs(&attachment)
	return attachment, true, nil
}

func (s *SQLiteStore) CanAccessInternalChatAttachment(id, userID int, administrator bool) (bool, error) {
	if administrator {
		var exists bool
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM internal_chat_attachments WHERE id=?)`, id).Scan(&exists)
		return exists, err
	}
	var allowed bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM internal_chat_attachments AS attachment
			LEFT JOIN internal_chat_messages AS message ON message.id=attachment.message_id
			WHERE attachment.id=? AND (
				(attachment.message_id IS NULL AND attachment.owner_id=?)
				OR message.sender_id=?
				OR message.recipient_id=?
				OR message.recipient_id IS NULL
			)
		)
	`, id, userID, userID, userID).Scan(&allowed)
	return allowed, err
}

func (s *SQLiteStore) CreateInternalChatMessage(senderID int, recipientID *int, content string, attachmentIDs []int, now time.Time) (models.InternalChatMessage, error) {
	content = strings.TrimSpace(content)
	attachmentIDs, err := normalizeAttachmentIDs(attachmentIDs)
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	if content == "" && len(attachmentIDs) == 0 {
		return models.InternalChatMessage{}, errors.New("消息内容和附件不能同时为空")
	}
	if len([]rune(content)) > 2000 {
		return models.InternalChatMessage{}, errors.New("消息内容不能超过 2000 个字符")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if recipientID != nil {
		if *recipientID == senderID {
			return models.InternalChatMessage{}, errors.New("不能向自己发起私聊")
		}
		var available bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id=? AND status<>'停用' AND can_login=1)`, *recipientID).Scan(&available); err != nil {
			return models.InternalChatMessage{}, err
		}
		if !available {
			return models.InternalChatMessage{}, errors.New("私聊用户不存在或不可用")
		}
	}

	if len(attachmentIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(attachmentIDs)), ",")
		args := make([]any, 0, len(attachmentIDs)+1)
		args = append(args, senderID)
		for _, id := range attachmentIDs {
			args = append(args, id)
		}
		var count int
		query := fmt.Sprintf(`SELECT COUNT(*) FROM internal_chat_attachments WHERE owner_id=? AND message_id IS NULL AND id IN (%s)`, placeholders)
		if err := tx.QueryRow(query, args...).Scan(&count); err != nil {
			return models.InternalChatMessage{}, err
		}
		if count != len(attachmentIDs) {
			return models.InternalChatMessage{}, errors.New("附件不存在、已被使用或不属于当前用户")
		}
	}

	result, err := tx.Exec(`INSERT INTO internal_chat_messages(sender_id,recipient_id,content,created_at) VALUES(?,?,?,?)`, senderID, recipientID, content, timeText(now))
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	messageID, err := result.LastInsertId()
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	if len(attachmentIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(attachmentIDs)), ",")
		args := make([]any, 0, len(attachmentIDs)+2)
		args = append(args, messageID, senderID)
		for _, id := range attachmentIDs {
			args = append(args, id)
		}
		query := fmt.Sprintf(`UPDATE internal_chat_attachments SET message_id=? WHERE owner_id=? AND message_id IS NULL AND id IN (%s)`, placeholders)
		updated, err := tx.Exec(query, args...)
		if err != nil {
			return models.InternalChatMessage{}, err
		}
		count, err := updated.RowsAffected()
		if err != nil || count != int64(len(attachmentIDs)) {
			return models.InternalChatMessage{}, errors.New("附件关联消息失败")
		}
	}
	if err := tx.Commit(); err != nil {
		return models.InternalChatMessage{}, err
	}

	messages, err := s.listInternalChatMessagesByID(int(messageID))
	if err != nil {
		return models.InternalChatMessage{}, err
	}
	if len(messages) != 1 {
		return models.InternalChatMessage{}, errors.New("消息创建后读取失败")
	}
	return messages[0], nil
}

func normalizeAttachmentIDs(ids []int) ([]int, error) {
	if len(ids) > maxInternalChatAttachmentsPerMessage {
		return nil, fmt.Errorf("每条消息最多发送 %d 个附件", maxInternalChatAttachmentsPerMessage)
	}
	result := make([]int, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("附件 ID 无效")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
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
	messages, err := scanInternalChatMessages(rows)
	if err != nil {
		return nil, err
	}
	if err := s.loadInternalChatAttachments(messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func scanInternalChatMessages(rows *sql.Rows) ([]models.InternalChatMessage, error) {
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
		message.Attachments = make([]models.InternalChatAttachment, 0)
		message.CreatedAt = parseTime(createdAt)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *SQLiteStore) loadInternalChatAttachments(messages []models.InternalChatMessage) error {
	if len(messages) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(messages)), ",")
	args := make([]any, 0, len(messages))
	byID := make(map[int]int, len(messages))
	for index := range messages {
		args = append(args, messages[index].ID)
		byID[messages[index].ID] = index
	}
	query := fmt.Sprintf(`
		SELECT id,message_id,owner_id,original_name,stored_name,mime_type,size,is_image,created_at
		FROM internal_chat_attachments WHERE message_id IN (%s) ORDER BY id
	`, placeholders)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var attachment models.InternalChatAttachment
		var messageID int
		var isImage bool
		var createdAt string
		if err := rows.Scan(
			&attachment.ID, &messageID, &attachment.OwnerID, &attachment.OriginalName,
			&attachment.StoredName, &attachment.MimeType, &attachment.Size, &isImage, &createdAt,
		); err != nil {
			return err
		}
		attachment.MessageID = &messageID
		attachment.IsImage = isImage
		attachment.CreatedAt = parseTime(createdAt)
		setInternalChatAttachmentURLs(&attachment)
		if index, ok := byID[messageID]; ok {
			messages[index].Attachments = append(messages[index].Attachments, attachment)
		}
	}
	return rows.Err()
}

func setInternalChatAttachmentURLs(attachment *models.InternalChatAttachment) {
	attachment.DownloadURL = fmt.Sprintf("/api/internal-chat/attachments/%d/download", attachment.ID)
	if attachment.IsImage {
		attachment.PreviewURL = fmt.Sprintf("/api/internal-chat/attachments/%d/preview", attachment.ID)
	}
}
