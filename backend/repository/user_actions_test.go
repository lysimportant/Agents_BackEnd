package repository

import (
	"reflect"
	"testing"

	"collector-backend/models"
	"collector-backend/permissions"
)

func TestUserActionPermissionsMergeAndPersist(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	var ordinaryRole models.Role
	for _, role := range store.ListRoles() {
		if !permissions.IsAdministratorRoleCode(role.Code) {
			ordinaryRole = role
			break
		}
	}
	if ordinaryRole.ID == 0 {
		t.Fatal("ordinary role seed missing")
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "personal-action-user",
		Name:     "个人动作权限用户",
		RoleID:   &ordinaryRole.ID,
		Status:   "在岗",
		CanLogin: &canLogin,
	}, "hash")
	if message != "" {
		t.Fatalf("create user: %s", message)
	}

	personalCodes, message := store.UpdateUserActions(user.ID, []string{
		permissions.FilesUpdate,
		permissions.ArticlesCreate,
		permissions.ArticlesCreate,
	})
	if message != "" {
		t.Fatalf("update user actions: %s", message)
	}
	wantPersonal := permissions.MergeCodes([]string{permissions.FilesUpdate, permissions.ArticlesCreate})
	if !reflect.DeepEqual(personalCodes, wantPersonal) {
		t.Fatalf("personal actions = %v, want %v", personalCodes, wantPersonal)
	}

	detail, message := store.GetUserPermissionDetail(user.ID)
	if message != "" {
		t.Fatalf("get detail: %s", message)
	}
	if !reflect.DeepEqual(detail.RoleActionCodes, permissions.DefaultRoleCodes()) {
		t.Fatalf("role actions = %v, want %v", detail.RoleActionCodes, permissions.DefaultRoleCodes())
	}
	if !reflect.DeepEqual(detail.UserActionCodes, wantPersonal) {
		t.Fatalf("detail personal actions = %v, want %v", detail.UserActionCodes, wantPersonal)
	}
	wantEffective := permissions.MergeCodes(permissions.DefaultRoleCodes(), wantPersonal)
	if !reflect.DeepEqual(detail.EffectiveActionCodes, wantEffective) {
		t.Fatalf("effective actions = %v, want %v", detail.EffectiveActionCodes, wantEffective)
	}

	if _, message := store.UpdateUserActions(user.ID, []string{"unknown.action"}); message == "" {
		t.Fatal("unknown action code was accepted")
	}
	persisted, message := store.ListUserActionPermissions(user.ID)
	if message != "" || !reflect.DeepEqual(persisted, wantEffective) {
		t.Fatalf("invalid update changed existing grants: message=%s actions=%v", message, persisted)
	}

	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	if _, message := store.UpdateUserActions(mh.ID, []string{}); message == "" {
		t.Fatal("system administrator action grants were mutable")
	}
	adminActions, message := store.ListUserActionPermissions(mh.ID)
	if message != "" || !reflect.DeepEqual(adminActions, permissions.AllCodes()) {
		t.Fatalf("system administrator actions = %v, message=%s", adminActions, message)
	}
}
