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

	ArticlesPortalPublish = "articles.portal-publish"
	FilesPortalPublish = "files.portal-publish"

	SocketQuery  = "socket.query"
	SocketView   = "socket.view"
	SocketSend   = "socket.send"
	SocketDelete = "socket.delete"

	VisitorAnalyticsQuery = "visitor-analytics.query"
	VisitorAnalyticsView  = "visitor-analytics.view"
)

// IsSuperAdminRoleCode 校验对应业务条件。
func IsSuperAdminRoleCode(code string) bool {
	return strings.EqualFold(strings.TrimSpace(code), SuperAdminRoleCode)
}

// IsAdministratorRoleCode 校验对应业务条件。
func IsAdministratorRoleCode(code string) bool {
	code = strings.TrimSpace(code)
	return strings.EqualFold(code, SuperAdminRoleCode) || strings.EqualFold(code, SystemAdminRoleCode)
}

// Definition 定义对应业务的数据结构与调用契约。
type Definition struct {
	// Code 表示编码。
	Code string `json:"code"`
	// Resource 表示变量 Resource。
	Resource string `json:"resource"`
	// Action 表示操作权限。
	Action string `json:"action"`
	// Label 表示显示标签。
	Label string `json:"label"`
	// ReadOnly 表示只读状态。
	ReadOnly bool `json:"readOnly"`
}

// definitions 保存模块使用的固定配置或共享状态。
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
	{ArticlesPortalPublish, "articles", "portal-publish", "门户发布文章", false},
	{FilesPortalPublish, "files", "portal-publish", "门户发布文件", false},
	{SocketQuery, "socket", "query", "查询客服会话", true},
	{SocketView, "socket", "view", "查看客服聊天", true},
	{SocketSend, "socket", "send", "回复客服消息", false},
	{SocketDelete, "socket", "delete", "删除客服会话", false},
	{VisitorAnalyticsQuery, "visitor-analytics", "query", "查询访问分析", true},
	{VisitorAnalyticsView, "visitor-analytics", "view", "查看访问明细", true},
}

// Definitions 实现对应业务逻辑。
func Definitions() []Definition {
	// result 保存操作结果。
	result := make([]Definition, len(definitions))
	copy(result, definitions)
	return result
}

// AllCodes 实现对应业务逻辑。
func AllCodes() []string {
	return filterCodes(func(Definition) bool { return true })
}

// DefaultRoleCodes 实现对应业务逻辑。
func DefaultRoleCodes() []string {
	return filterCodes(func(definition Definition) bool { return definition.ReadOnly })
}

// RoleCodes 实现对应业务逻辑。
func RoleCodes(roleCode string) []string {
	if IsAdministratorRoleCode(roleCode) {
		return AllCodes()
	}
	return DefaultRoleCodes()
}

// IsKnown 校验对应业务条件。
func IsKnown(code string) bool {
	code = strings.TrimSpace(code)
	// definition 表示当前循环中的索引、键或业务元素。
	for _, definition := range definitions {
		if definition.Code == code {
			return true
		}
	}
	return false
}

// IsReadOnly 校验对应业务条件。
func IsReadOnly(code string) bool {
	code = strings.TrimSpace(code)
	// definition 表示当前循环中的索引、键或业务元素。
	for _, definition := range definitions {
		if definition.Code == code {
			return definition.ReadOnly
		}
	}
	return false
}

// Contains 实现对应业务逻辑。
func Contains(codes []string, required string) bool {
	// code 表示当前循环中的索引、键或业务元素。
	for _, code := range codes {
		if code == required {
			return true
		}
	}
	return false
}

// NormalizeCodes 校验动作权限编码，并按权限目录顺序去重返回。
// 空切片是有效输入，用于清空用户个人附加权限。
func NormalizeCodes(codes []string) ([]string, bool) {
	// selected 保存已选择。
	selected := make(map[string]bool, len(codes))
	// rawCode 表示当前循环中的索引、键或业务元素。
	for _, rawCode := range codes {
		// code 保存编码。
		code := strings.TrimSpace(rawCode)
		if !IsKnown(code) {
			return nil, false
		}
		selected[code] = true
	}
	return selectedCodes(selected), true
}

// MergeCodes 合并权限并仅按目录顺序返回已知编码。
func MergeCodes(codeGroups ...[]string) []string {
	// selected 保存已选择。
	selected := map[string]bool{}
	// codes 表示当前循环中的索引、键或业务元素。
	for _, codes := range codeGroups {
		// rawCode 表示当前循环中的索引、键或业务元素。
		for _, rawCode := range codes {
			// code 保存编码。
			code := strings.TrimSpace(rawCode)
			if IsKnown(code) {
				selected[code] = true
			}
		}
	}
	return selectedCodes(selected)
}

// selectedCodes 实现对应业务逻辑。
func selectedCodes(selected map[string]bool) []string {
	// codes 保存编码。
	codes := make([]string, 0, len(selected))
	// definition 表示当前循环中的索引、键或业务元素。
	for _, definition := range definitions {
		if selected[definition.Code] {
			codes = append(codes, definition.Code)
		}
	}
	return codes
}

// filterCodes 实现对应业务逻辑。
func filterCodes(keep func(Definition) bool) []string {
	// codes 保存编码。
	codes := make([]string, 0, len(definitions))
	// definition 表示当前循环中的索引、键或业务元素。
	for _, definition := range definitions {
		if keep(definition) {
			codes = append(codes, definition.Code)
		}
	}
	return codes
}
