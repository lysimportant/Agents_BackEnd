package permissions

import "strings"

const (
	SuperAdminRoleCode  = "super-admin"
	SystemAdminRoleCode = "system-admin"

	DashboardQuery  = "dashboard.query"
	DashboardView   = "dashboard.view"
	DashboardCreate = "dashboard.create"

	UsersQuery             = "users.query"
	UsersView              = "users.view"
	UsersCreate            = "users.create"
	UsersUpdate            = "users.update"
	UsersDelete            = "users.delete"
	UsersPermissionsUpdate = "users.permissions.update"

	DepartmentsQuery             = "departments.query"
	DepartmentsView              = "departments.view"
	DepartmentsCreate            = "departments.create"
	DepartmentsUpdate            = "departments.update"
	DepartmentsDelete            = "departments.delete"
	DepartmentsPermissionsUpdate = "departments.permissions.update"

	RolesQuery             = "roles.query"
	RolesView              = "roles.view"
	RolesCreate            = "roles.create"
	RolesUpdate            = "roles.update"
	RolesDelete            = "roles.delete"
	RolesPermissionsUpdate = "roles.permissions.update"

	MenusQuery  = "menus.query"
	MenusView   = "menus.view"
	MenusCreate = "menus.create"
	MenusUpdate = "menus.update"
	MenusDelete = "menus.delete"

	ArticlesQuery  = "articles.query"
	ArticlesView   = "articles.view"
	ArticlesCreate = "articles.create"
	ArticlesUpdate = "articles.update"
	ArticlesDelete = "articles.delete"

	FilesQuery           = "files.query"
	FilesView            = "files.view"
	FilesCreate          = "files.create"
	FilesUpdate          = "files.update"
	FilesDelete          = "files.delete"
	FilesRestore         = "files.restore"
	FilesPermanentDelete = "files.permanent-delete"
)

func IsSuperAdminRoleCode(code string) bool {
	return strings.EqualFold(strings.TrimSpace(code), SuperAdminRoleCode)
}

func IsAdministratorRoleCode(code string) bool {
	code = strings.TrimSpace(code)
	return strings.EqualFold(code, SuperAdminRoleCode) || strings.EqualFold(code, SystemAdminRoleCode)
}

type Definition struct {
	Code     string `json:"code"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Label    string `json:"label"`
	ReadOnly bool   `json:"readOnly"`
}

var definitions = []Definition{
	{DashboardQuery, "dashboard", "query", "查询工作台", true},
	{DashboardView, "dashboard", "view", "查看工作台", true},
	{DashboardCreate, "dashboard", "create", "新增采集数据", false},
	{UsersQuery, "users", "query", "查询用户", true},
	{UsersView, "users", "view", "查看用户", true},
	{UsersCreate, "users", "create", "新增用户", false},
	{UsersUpdate, "users", "update", "编辑用户", false},
	{UsersDelete, "users", "delete", "删除用户", false},
	{UsersPermissionsUpdate, "users", "permissions.update", "配置用户权限", false},
	{DepartmentsQuery, "departments", "query", "查询部门", true},
	{DepartmentsView, "departments", "view", "查看部门", true},
	{DepartmentsCreate, "departments", "create", "新增部门", false},
	{DepartmentsUpdate, "departments", "update", "编辑部门", false},
	{DepartmentsDelete, "departments", "delete", "删除部门", false},
	{DepartmentsPermissionsUpdate, "departments", "permissions.update", "配置部门权限", false},
	{RolesQuery, "roles", "query", "查询角色", true},
	{RolesView, "roles", "view", "查看角色", true},
	{RolesCreate, "roles", "create", "新增角色", false},
	{RolesUpdate, "roles", "update", "编辑角色", false},
	{RolesDelete, "roles", "delete", "删除角色", false},
	{RolesPermissionsUpdate, "roles", "permissions.update", "配置角色权限", false},
	{MenusQuery, "menus", "query", "查询菜单", true},
	{MenusView, "menus", "view", "查看菜单", true},
	{MenusCreate, "menus", "create", "新增菜单", false},
	{MenusUpdate, "menus", "update", "编辑菜单", false},
	{MenusDelete, "menus", "delete", "删除菜单", false},
	{ArticlesQuery, "articles", "query", "查询文章", true},
	{ArticlesView, "articles", "view", "查看文章", true},
	{ArticlesCreate, "articles", "create", "新增文章", false},
	{ArticlesUpdate, "articles", "update", "编辑文章", false},
	{ArticlesDelete, "articles", "delete", "删除文章", false},
	{FilesQuery, "files", "query", "查询文件", true},
	{FilesView, "files", "view", "查看文件", true},
	{FilesCreate, "files", "create", "上传文件", false},
	{FilesUpdate, "files", "update", "编辑文件", false},
	{FilesDelete, "files", "delete", "删除文件", false},
	{FilesRestore, "files", "restore", "恢复文件", false},
	{FilesPermanentDelete, "files", "permanent-delete", "彻底删除文件", false},
}

func Definitions() []Definition {
	result := make([]Definition, len(definitions))
	copy(result, definitions)
	return result
}

func AllCodes() []string {
	return filterCodes(func(Definition) bool { return true })
}

func DefaultRoleCodes() []string {
	return filterCodes(func(definition Definition) bool { return definition.ReadOnly })
}

func RoleCodes(roleCode string) []string {
	if IsAdministratorRoleCode(roleCode) {
		return AllCodes()
	}
	return DefaultRoleCodes()
}

func IsKnown(code string) bool {
	code = strings.TrimSpace(code)
	for _, definition := range definitions {
		if definition.Code == code {
			return true
		}
	}
	return false
}

func IsReadOnly(code string) bool {
	code = strings.TrimSpace(code)
	for _, definition := range definitions {
		if definition.Code == code {
			return definition.ReadOnly
		}
	}
	return false
}

func Contains(codes []string, required string) bool {
	for _, code := range codes {
		if code == required {
			return true
		}
	}
	return false
}

// NormalizeCodes validates action codes and returns them once in catalog order.
// An empty slice is valid and is used to clear a user's personal grants.
func NormalizeCodes(codes []string) ([]string, bool) {
	selected := make(map[string]bool, len(codes))
	for _, rawCode := range codes {
		code := strings.TrimSpace(rawCode)
		if !IsKnown(code) {
			return nil, false
		}
		selected[code] = true
	}
	return selectedCodes(selected), true
}

// MergeCodes combines grants and returns only known codes in catalog order.
func MergeCodes(codeGroups ...[]string) []string {
	selected := map[string]bool{}
	for _, codes := range codeGroups {
		for _, rawCode := range codes {
			code := strings.TrimSpace(rawCode)
			if IsKnown(code) {
				selected[code] = true
			}
		}
	}
	return selectedCodes(selected)
}

func selectedCodes(selected map[string]bool) []string {
	codes := make([]string, 0, len(selected))
	for _, definition := range definitions {
		if selected[definition.Code] {
			codes = append(codes, definition.Code)
		}
	}
	return codes
}

func filterCodes(keep func(Definition) bool) []string {
	codes := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		if keep(definition) {
			codes = append(codes, definition.Code)
		}
	}
	return codes
}
