package repository

import "collector-backend/permissions"

func (s *SQLiteStore) listUserActionCodes(userID int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT action_code
		FROM user_action_permissions
		WHERE user_id=?
		ORDER BY action_code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions.MergeCodes(codes), nil
}

func (s *SQLiteStore) listEffectiveUserActionCodes(userID int) ([]string, string) {
	user, ok := s.FindUserByID(userID)
	if !ok {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return permissions.AllCodes(), ""
	}

	roleCodes := []string{}
	if user.RoleID != nil {
		role, found := s.FindRoleByID(*user.RoleID)
		if found && role.Status == "启用" {
			var err error
			roleCodes, err = s.listAssignedRoleActionCodes(user.RoleID)
			if err != nil {
				return nil, "查询角色动作权限失败"
			}
		}
	}
	userCodes, err := s.listUserActionCodes(userID)
	if err != nil {
		return nil, "查询用户动作权限失败"
	}
	return permissions.MergeCodes(roleCodes, userCodes), ""
}

func (s *SQLiteStore) UpdateUserActions(userID int, actionCodes []string) ([]string, string) {
	user, ok := s.FindUserByID(userID)
	if !ok {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return nil, "超级管理员和系统管理员动作权限固定为全部，不能修改"
	}
	codes, valid := permissions.NormalizeCodes(actionCodes)
	if !valid {
		return nil, "包含不存在的动作权限"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新用户动作权限失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM user_action_permissions WHERE user_id=?`, userID); err != nil {
		return nil, "更新用户动作权限失败"
	}
	for _, code := range codes {
		if _, err := tx.Exec(
			`INSERT INTO user_action_permissions(user_id,action_code) VALUES(?,?)`,
			userID, code,
		); err != nil {
			return nil, "更新用户动作权限失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "更新用户动作权限失败"
	}
	return codes, ""
}
