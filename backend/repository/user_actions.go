package repository

import "collector-backend/permissions"

// listUserActionCodes 查询并返回对应业务列表。
func (s *SQLiteStore) listUserActionCodes(userID int) ([]string, error) {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
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

	// codes 保存编码。
	codes := []string{}
	for rows.Next() {
		// code 保存编码。
		var code string
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions.MergeCodes(codes), nil
}

// listEffectiveUserActionCodes 查询并返回对应业务列表。
func (s *SQLiteStore) listEffectiveUserActionCodes(userID int) ([]string, string) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := s.FindUserByID(userID)
	if !ok {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return permissions.AllCodes(), ""
	}

	// roleCodes 保存角色。
	roleCodes := []string{}
	if user.RoleID != nil {
		// role、found 保存业务值及其是否存在或处理成功的标记。
		role, found := s.FindRoleByID(*user.RoleID)
		if found && role.Status == "启用" {
			// err 保存当前操作结果以及可能返回的错误状态。
			var err error
			roleCodes, err = s.listAssignedRoleActionCodes(user.RoleID)
			if err != nil {
				return nil, "查询角色动作权限失败"
			}
		}
	}
	// userCodes、err 保存当前操作结果以及可能返回的错误状态。
	userCodes, err := s.listUserActionCodes(userID)
	if err != nil {
		return nil, "查询用户动作权限失败"
	}
	return permissions.MergeCodes(roleCodes, userCodes), ""
}

// UpdateUserActions 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateUserActions(userID int, actionCodes []string) ([]string, string) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := s.FindUserByID(userID)
	if !ok {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return nil, "超级管理员和系统管理员动作权限固定为全部，不能修改"
	}
	// codes、valid 保存编码、校验结果。
	codes, valid := permissions.NormalizeCodes(actionCodes)
	if !valid {
		return nil, "包含不存在的动作权限"
	}

	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新用户动作权限失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM user_action_permissions WHERE user_id=?`, userID); err != nil {
		return nil, "更新用户动作权限失败"
	}
	// code 表示当前循环中的索引、键或业务元素。
	for _, code := range codes {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(
			`INSERT INTO user_action_permissions(user_id,action_code) VALUES(?,?)`,
			userID, code,
		); err != nil {
			return nil, "更新用户动作权限失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return nil, "更新用户动作权限失败"
	}
	return codes, ""
}
