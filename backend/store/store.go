package store

import (
	"strings"
	"sync"
	"time"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/permissions"
)

const (
	superAdminRoleCode  = permissions.SuperAdminRoleCode
	systemAdminRoleCode = permissions.SystemAdminRoleCode
)

type MemoryStore struct {
	mu                sync.Mutex
	dataPoints        []models.DataPoint
	users             []models.User
	departments       []models.Department
	roles             []models.Role
	menus             []models.Menu
	articles          []models.Article
	files             []models.ManagedFile
	userMenuIDs       map[int][]int
	userActionCodes   map[int][]string
	departmentMenuIDs map[int][]int
	roleMenuIDs       map[int][]int
	nextUserID        int
	nextDepartmentID  int
	nextRoleID        int
	nextMenuID        int
	nextArticleID     int
	nextFileID        int
}

func NewMemoryStore() *MemoryStore {
	now := time.Now()
	return &MemoryStore{
		dataPoints: []models.DataPoint{
			{ID: 1, Source: "collector-a", Metric: "temperature", Value: 24.6, Unit: "°C", CreatedAt: now.Add(-2 * time.Hour)},
			{ID: 2, Source: "collector-b", Metric: "humidity", Value: 63.2, Unit: "%", CreatedAt: now.Add(-1 * time.Hour)},
			{ID: 3, Source: "collector-a", Metric: "temperature", Value: 25.1, Unit: "°C", CreatedAt: now},
		},
		users: []models.User{
			{ID: 1, Username: "MH", Name: "MH", RoleID: intPtr(1), Role: "超级管理员", RoleCode: superAdminRoleCode, DepartmentID: intPtr(1), Department: "HuaJian技术有限公司", Status: "在岗", Shift: "常白班", Phone: "", Email: "mh@example.com", Age: 0, CanLogin: true, PasswordHash: auth.MustHashPassword("123"), CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 2, Username: "zhang.gong", Name: "张工", Role: "产线主管", DepartmentID: intPtr(2), Department: "制造部", Status: "在岗", Shift: "早班", Phone: "13800000001", Email: "zhang.gong@example.com", CanLogin: true, PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: 3, Username: "li.min", Name: "李敏", Role: "质量工程师", DepartmentID: intPtr(3), Department: "质量与流程IT部", Status: "巡检", Shift: "中班", Phone: "13800000002", Email: "li.min@example.com", CanLogin: true, PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-90 * time.Minute)},
			{ID: 4, Username: "wang.qiang", Name: "王强", Role: "设备维护", DepartmentID: intPtr(2), Department: "制造部", Status: "待命", Shift: "夜班", Phone: "13800000003", Email: "wang.qiang@example.com", CanLogin: true, PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-36 * time.Hour), UpdatedAt: now.Add(-30 * time.Minute)},
		},
		departments: []models.Department{
			{ID: 1, Name: "HuaJian技术有限公司", Code: "huajian", Sort: 10, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 2, Name: "制造部", Code: "manufacturing", ParentID: intPtr(1), Sort: 20, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 3, Name: "质量与流程IT部", Code: "quality-process-it", ParentID: intPtr(1), Sort: 30, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		},
		roles: []models.Role{
			{ID: 1, Name: "超级管理员", Code: superAdminRoleCode, Description: "系统最高权限，仅用于平台最高级管理", Sort: 10, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 2, Name: "系统管理员", Code: systemAdminRoleCode, Description: "负责用户、部门、角色、菜单和权限配置", Sort: 20, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 3, Name: "部门管理员", Code: "department-admin", Description: "负责本部门用户与业务数据管理", Sort: 30, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 4, Name: "内容编辑", Code: "content-editor", Description: "负责内容创建、编辑与维护", Sort: 40, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 5, Name: "审核员", Code: "auditor", Description: "负责内容审核与合规查看", Sort: 50, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 6, Name: "普通用户", Code: "viewer", Description: "基础查询与查看角色", Sort: 60, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 7, Name: "商品管理员", Code: "product-manager", Description: "负责商品、分类、品牌和上下架管理", Sort: 110, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 8, Name: "订单管理员", Code: "order-manager", Description: "负责订单处理、发货与售后流转", Sort: 120, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 9, Name: "仓库管理员", Code: "warehouse-manager", Description: "负责库存、入库、出库和盘点", Sort: 130, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 10, Name: "客服专员", Code: "customer-service", Description: "负责客户咨询、退款与售后服务", Sort: 140, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 11, Name: "财务人员", Code: "finance", Description: "负责支付、对账、退款和财务报表", Sort: 150, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		},
		menus: []models.Menu{
			{ID: 1, Name: "工作台", Code: "dashboard", Path: "dashboard", Icon: "dashboard", ParentID: nil, Sort: 10, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
			{ID: 2, Name: "工单管理", Code: "workorder.manage", Path: "/production/workorders", Icon: "单据", ParentID: intPtr(1), Sort: 20, Status: "启用", CreatedAt: now.Add(-90 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: 3, Name: "质量追溯", Code: "quality.trace", Path: "/quality", Icon: "质检", ParentID: nil, Sort: 30, Status: "启用", CreatedAt: now.Add(-84 * time.Hour), UpdatedAt: now.Add(-90 * time.Minute)},
			{ID: 4, Name: "设备点检", Code: "equipment.inspection", Path: "/equipment/inspection", Icon: "设备", ParentID: nil, Sort: 40, Status: "启用", CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		},
		articles: []models.Article{
			{ID: 1, Title: "生产早会质量通报", Category: "通知公告", Author: "管理员", Status: "已发布", Summary: "汇总昨日生产质量问题与今日重点关注事项。", Content: "请各产线主管关注关键工序质量波动，及时完成闭环。", Views: 328, CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: 2, Title: "设备点检制度修订说明", Category: "制度文档", Author: "设备部", Status: "待审核", Summary: "设备点检制度新增夜班巡检要求。", Content: "新版制度要求关键设备每日三班点检并上传记录。", Views: 96, CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-90 * time.Minute)},
			{ID: 3, Title: "新员工 MES 操作指南", Category: "帮助中心", Author: "培训专员", Status: "草稿", Summary: "面向新员工的 MES 基础操作说明。", Content: "包含登录、看板查看、工单处理与异常反馈流程。", Views: 54, CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-30 * time.Minute)},
		},
		files: []models.ManagedFile{},
		userMenuIDs: map[int][]int{
			1: {1, 2, 3, 4},
			2: {1, 2, 3, 4},
			3: {1, 3},
			4: {1, 4},
		},
		userActionCodes: map[int][]string{},
		departmentMenuIDs: map[int][]int{
			1: {1, 2, 3, 4},
			2: {1},
			3: {1},
		},
		roleMenuIDs: map[int][]int{
			1:  {1, 2, 3, 4},
			2:  {1, 2, 3, 4},
			3:  {1},
			4:  {1},
			5:  {1},
			6:  {1},
			7:  {1},
			8:  {1},
			9:  {1},
			10: {1},
			11: {1},
		},
		nextUserID:       5,
		nextDepartmentID: 4,
		nextRoleID:       12,
		nextMenuID:       5,
		nextArticleID:    4,
		nextFileID:       1,
	}
}

func (s *MemoryStore) ListDataPoints() []models.DataPoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.DataPoint(nil), s.dataPoints...)
}

func (s *MemoryStore) CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	dataPoint := models.DataPoint{ID: len(s.dataPoints) + 1, Source: request.Source, Metric: request.Metric, Value: request.Value, Unit: request.Unit, CreatedAt: time.Now()}
	s.dataPoints = append(s.dataPoints, dataPoint)
	return dataPoint
}

func (s *MemoryStore) ListUsers() []models.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.User(nil), s.users...)
}

func (s *MemoryStore) FindUserByID(id int) (models.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findUserByID(id)
}

func (s *MemoryStore) FindUserByUsername(username string) (models.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findUserByUsername(username)
}

func (s *MemoryStore) CreateUser(request models.UserRequest, passwordHash string) (models.User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	username := strings.TrimSpace(request.Username)
	if _, exists := s.findUserByUsername(username); exists {
		return models.User{}, "账号已存在"
	}
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	roleID, roleName, roleCode, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	canLogin := true
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "在岗"
	}
	if status == "停用" {
		canLogin = false
	}
	age := 0
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	description, avatarURL := "", ""
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	user := models.User{ID: s.nextUserID, Username: username, Name: request.Name, RoleID: roleID, Role: roleName, RoleCode: roleCode, DepartmentID: departmentID, Department: departmentName, Status: status, Shift: request.Shift, Phone: request.Phone, Email: request.Email, Age: age, Description: description, AvatarURL: avatarURL, CanLogin: canLogin, PasswordHash: passwordHash, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	s.nextUserID++
	s.users = append(s.users, user)
	s.userMenuIDs[user.ID] = []int{}
	return user, ""
}

func (s *MemoryStore) UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findUserIndexByID(id)
	if !found {
		return models.User{}, "用户不存在"
	}
	if strings.EqualFold(s.users[index].Username, "MH") {
		root, exists := s.findDepartmentByCode("huajian")
		if !exists {
			return models.User{}, "根部门不存在"
		}
		canLogin := true
		systemRole, exists := s.findRoleByCode(superAdminRoleCode)
		if !exists {
			return models.User{}, "超级管理员角色不存在"
		}
		request.Username = "MH"
		request.RoleID = &systemRole.ID
		request.Role = systemRole.Name
		request.DepartmentID = &root.ID
		request.Department = root.Name
		request.CanLogin = &canLogin
	}
	username := strings.TrimSpace(request.Username)
	if existing, exists := s.findUserByUsername(username); exists && existing.ID != id {
		return models.User{}, "账号已存在"
	}
	if strings.TrimSpace(request.Status) == "" {
		request.Status = s.users[index].Status
	}
	if strings.EqualFold(s.users[index].Username, "MH") {
		request.Status = "在岗"
	}
	canLogin := s.users[index].CanLogin
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	if request.Status == "停用" {
		canLogin = false
	}
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	roleID, roleName, roleCode, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	age := s.users[index].Age
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	s.users[index].Username = username
	s.users[index].Name = request.Name
	s.users[index].RoleID = roleID
	s.users[index].Role = roleName
	s.users[index].RoleCode = roleCode
	s.users[index].DepartmentID = departmentID
	s.users[index].Department = departmentName
	s.users[index].Status = request.Status
	s.users[index].Shift = request.Shift
	s.users[index].Phone = request.Phone
	s.users[index].Email = request.Email
	s.users[index].CanLogin = canLogin
	s.users[index].Age = age
	if request.Description != nil {
		s.users[index].Description = strings.TrimSpace(*request.Description)
	}
	if request.AvatarURL != nil {
		s.users[index].AvatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	if passwordHash != "" {
		s.users[index].PasswordHash = passwordHash
	}
	s.users[index].UpdatedAt = time.Now()
	return s.users[index], ""
}

func (s *MemoryStore) DeleteUser(id int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findUserIndexByID(id)
	if !found {
		return "用户不存在"
	}
	if strings.EqualFold(s.users[index].Username, "MH") {
		return "默认管理员 MH 不能删除"
	}
	s.users = append(s.users[:index], s.users[index+1:]...)
	delete(s.userMenuIDs, id)
	delete(s.userActionCodes, id)
	return ""
}

func (s *MemoryStore) UpdateUserPassword(id int, passwordHash string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(passwordHash) == "" {
		return "密码不能为空"
	}
	index, found := s.findUserIndexByID(id)
	if !found {
		return "用户不存在"
	}
	s.users[index].PasswordHash = passwordHash
	s.users[index].UpdatedAt = time.Now()
	return ""
}

func (s *MemoryStore) UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findUserIndexByID(id)
	if !found {
		return models.User{}, "用户不存在"
	}
	name, email, phone := s.users[index].Name, s.users[index].Email, s.users[index].Phone
	age, description, avatarURL := s.users[index].Age, s.users[index].Description, s.users[index].AvatarURL
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
		if name == "" {
			return models.User{}, "姓名不能为空"
		}
	}
	if request.Email != nil {
		email = strings.TrimSpace(*request.Email)
	}
	if request.Phone != nil {
		phone = strings.TrimSpace(*request.Phone)
	}
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	s.users[index].Name = name
	s.users[index].Email = email
	s.users[index].Phone = phone
	s.users[index].Age = age
	s.users[index].Description = description
	s.users[index].AvatarURL = avatarURL
	s.users[index].UpdatedAt = time.Now()
	return s.users[index], ""
}

func (s *MemoryStore) ListRoleUsers(roleID int) ([]models.User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.findRoleByID(roleID); !found {
		return nil, "角色不存在"
	}
	users := []models.User{}
	for _, user := range s.users {
		if user.RoleID != nil && *user.RoleID == roleID {
			users = append(users, user)
		}
	}
	return users, ""
}

func (s *MemoryStore) ListDepartmentUsers(departmentID int) ([]models.User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.findDepartmentByID(departmentID); !found {
		return nil, "部门不存在"
	}
	users := []models.User{}
	for _, user := range s.users {
		if user.DepartmentID != nil && *user.DepartmentID == departmentID {
			users = append(users, user)
		}
	}
	return users, ""
}

func (s *MemoryStore) ListDepartments() []models.Department {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.Department(nil), s.departments...)
}

func (s *MemoryStore) FindDepartmentByID(id int) (models.Department, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findDepartmentByID(id)
}

func (s *MemoryStore) CreateDepartment(request models.DepartmentRequest) (models.Department, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if _, exists := s.findDepartmentByCode(code); exists {
		return models.Department{}, "部门编码已存在"
	}
	if request.ParentID != nil {
		if _, exists := s.findDepartmentByID(*request.ParentID); !exists {
			return models.Department{}, "上级部门不存在"
		}
	}
	now := time.Now()
	department := models.Department{
		ID: s.nextDepartmentID, Name: strings.TrimSpace(request.Name), Code: code, ParentID: request.ParentID,
		Leader: request.Leader, Phone: request.Phone, Email: request.Email, Sort: request.Sort, Status: request.Status,
		CreatedAt: now, UpdatedAt: now,
	}
	s.nextDepartmentID++
	s.departments = append(s.departments, department)
	if department.Code == "board-office" {
		s.departmentMenuIDs[department.ID] = s.allMenuIDs()
	} else {
		s.departmentMenuIDs[department.ID] = s.dashboardMenuIDs()
	}
	return department, ""
}

func (s *MemoryStore) UpdateDepartment(id int, request models.DepartmentRequest) (models.Department, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, exists := s.findDepartmentIndexByID(id)
	if !exists {
		return models.Department{}, "部门不存在"
	}
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if s.departments[index].Code == "huajian" && code != "huajian" {
		return models.Department{}, "根部门编码不可修改"
	}
	if s.departments[index].Code == "huajian" && request.ParentID != nil {
		return models.Department{}, "根部门不能设置上级部门"
	}
	if s.departments[index].Code == "huajian" && request.Status != "启用" {
		return models.Department{}, "根部门必须保持启用"
	}
	if other, exists := s.findDepartmentByCode(code); exists && other.ID != id {
		return models.Department{}, "部门编码已存在"
	}
	for parentID := request.ParentID; parentID != nil; {
		if *parentID == id {
			return models.Department{}, "上级部门不能是当前部门的下级"
		}
		parent, exists := s.findDepartmentByID(*parentID)
		if !exists {
			return models.Department{}, "上级部门不存在"
		}
		parentID = parent.ParentID
	}
	oldName := s.departments[index].Name
	s.departments[index].Name = strings.TrimSpace(request.Name)
	s.departments[index].Code = code
	s.departments[index].ParentID = request.ParentID
	s.departments[index].Leader = request.Leader
	s.departments[index].Phone = request.Phone
	s.departments[index].Email = request.Email
	s.departments[index].Sort = request.Sort
	s.departments[index].Status = request.Status
	s.departments[index].UpdatedAt = time.Now()
	if oldName != s.departments[index].Name {
		for userIndex := range s.users {
			if s.users[userIndex].DepartmentID != nil && *s.users[userIndex].DepartmentID == id {
				s.users[userIndex].Department = s.departments[index].Name
			}
		}
	}
	return s.departments[index], ""
}

func (s *MemoryStore) DeleteDepartment(id int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, exists := s.findDepartmentIndexByID(id)
	if !exists {
		return "部门不存在"
	}
	for _, department := range s.departments {
		if department.ParentID != nil && *department.ParentID == id {
			return "请先处理下级部门"
		}
	}
	for _, user := range s.users {
		if user.DepartmentID != nil && *user.DepartmentID == id {
			return "请先转移该部门用户"
		}
	}
	s.departments = append(s.departments[:index], s.departments[index+1:]...)
	delete(s.departmentMenuIDs, id)
	return ""
}

func (s *MemoryStore) ListDepartmentMenus(departmentID int) ([]models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.findDepartmentByID(departmentID); !exists {
		return nil, "部门不存在"
	}
	menus := make([]models.Menu, 0, len(s.departmentMenuIDs[departmentID]))
	for _, menuID := range s.departmentMenuIDs[departmentID] {
		if menu, found := s.findMenuByID(menuID); found {
			menus = append(menus, menu)
		}
	}
	return menus, ""
}

func (s *MemoryStore) UpdateDepartmentMenus(departmentID int, menuIDs []int) ([]int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	department, exists := s.findDepartmentByID(departmentID)
	if !exists {
		return nil, "部门不存在"
	}
	for _, menuID := range menuIDs {
		if !s.menuExists(menuID) {
			return nil, "菜单不存在"
		}
	}
	ids := uniqueIDs(menuIDs)
	if department.Code == "huajian" {
		allMenuIDs := make([]int, 0, len(s.menus))
		for _, menu := range s.menus {
			allMenuIDs = append(allMenuIDs, menu.ID)
		}
		if len(ids) != len(uniqueIDs(allMenuIDs)) {
			return nil, "根部门必须保留全部菜单权限"
		}
	}
	s.departmentMenuIDs[departmentID] = ids
	return append([]int(nil), ids...), ""
}

func (s *MemoryStore) ListRoles() []models.Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.Role(nil), s.roles...)
}

func (s *MemoryStore) FindRoleByID(id int) (models.Role, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findRoleByID(id)
}

func (s *MemoryStore) CreateRole(request models.RoleRequest) (models.Role, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if _, exists := s.findRoleByCode(code); exists {
		return models.Role{}, "角色编码已存在"
	}
	now := time.Now()
	role := models.Role{
		ID: s.nextRoleID, Name: strings.TrimSpace(request.Name), Code: code,
		Description: strings.TrimSpace(request.Description), Sort: request.Sort, Status: request.Status,
		CreatedAt: now, UpdatedAt: now,
	}
	s.nextRoleID++
	s.roles = append(s.roles, role)
	s.roleMenuIDs[role.ID] = s.dashboardMenuIDs()
	return role, ""
}

func (s *MemoryStore) UpdateRole(id int, request models.RoleRequest) (models.Role, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, exists := s.findRoleIndexByID(id)
	if !exists {
		return models.Role{}, "角色不存在"
	}
	existing := s.roles[index]
	code := strings.ToLower(strings.TrimSpace(request.Code))
	name := strings.TrimSpace(request.Name)
	if code != existing.Code {
		return models.Role{}, "角色编码创建后不可修改"
	}
	if existing.Code == superAdminRoleCode && (code != superAdminRoleCode || name != "超级管理员" || request.Status != "启用") {
		return models.Role{}, "超级管理员角色的名称、编码和状态不可修改"
	}
	if existing.Code == systemAdminRoleCode && (code != systemAdminRoleCode || name != "系统管理员" || request.Status != "启用") {
		return models.Role{}, "系统管理员角色的名称、编码和状态不可修改"
	}
	if other, exists := s.findRoleByCode(code); exists && other.ID != id {
		return models.Role{}, "角色编码已存在"
	}
	s.roles[index].Name = name
	s.roles[index].Code = code
	s.roles[index].Description = strings.TrimSpace(request.Description)
	s.roles[index].Sort = request.Sort
	s.roles[index].Status = request.Status
	s.roles[index].UpdatedAt = time.Now()
	for userIndex := range s.users {
		if s.users[userIndex].RoleID != nil && *s.users[userIndex].RoleID == id {
			s.users[userIndex].Role = name
			s.users[userIndex].RoleCode = code
			s.users[userIndex].UpdatedAt = time.Now()
		}
	}
	return s.roles[index], ""
}

func (s *MemoryStore) DeleteRole(id int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, exists := s.findRoleIndexByID(id)
	if !exists {
		return "角色不存在"
	}
	if permissions.IsAdministratorRoleCode(s.roles[index].Code) {
		return "超级管理员和系统管理员角色不能删除"
	}
	for _, user := range s.users {
		if user.RoleID != nil && *user.RoleID == id {
			return "请先转移该角色用户"
		}
	}
	s.roles = append(s.roles[:index], s.roles[index+1:]...)
	delete(s.roleMenuIDs, id)
	return ""
}

func (s *MemoryStore) ListRoleMenuIDs(roleID int) ([]int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.findRoleByID(roleID); !exists {
		return nil, "角色不存在"
	}
	return append([]int(nil), s.roleMenuIDs[roleID]...), ""
}

func (s *MemoryStore) UpdateRoleMenus(roleID int, menuIDs []int) ([]int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	role, exists := s.findRoleByID(roleID)
	if !exists {
		return nil, "角色不存在"
	}
	for _, menuID := range menuIDs {
		if !s.menuExists(menuID) {
			return nil, "菜单不存在"
		}
	}
	ids := uniqueIDs(menuIDs)
	if permissions.IsAdministratorRoleCode(role.Code) {
		allMenuIDs := make([]int, 0, len(s.menus))
		for _, menu := range s.menus {
			allMenuIDs = append(allMenuIDs, menu.ID)
		}
		if !sameIDs(ids, allMenuIDs) {
			return nil, "超级管理员和系统管理员角色必须保留全部菜单权限"
		}
	}
	s.roleMenuIDs[roleID] = ids
	return append([]int(nil), ids...), ""
}

func (s *MemoryStore) ListMenus() []models.Menu {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.Menu(nil), s.menus...)
}

func (s *MemoryStore) FindMenuByID(id int) (models.Menu, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findMenuByID(id)
}

func (s *MemoryStore) CreateMenu(request models.MenuRequest) (models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.ParentID != nil && !s.menuExists(*request.ParentID) {
		return models.Menu{}, "父级菜单不存在"
	}
	menu := models.Menu{ID: s.nextMenuID, Name: request.Name, Code: request.Code, Path: request.Path, Icon: request.Icon, ParentID: request.ParentID, Sort: request.Sort, Status: request.Status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	s.nextMenuID++
	s.menus = append(s.menus, menu)
	if root, exists := s.findDepartmentByCode("huajian"); exists {
		s.departmentMenuIDs[root.ID] = append(s.departmentMenuIDs[root.ID], menu.ID)
	}
	if board, exists := s.findDepartmentByCode("board-office"); exists {
		s.departmentMenuIDs[board.ID] = append(s.departmentMenuIDs[board.ID], menu.ID)
	}
	for _, role := range s.roles {
		if permissions.IsAdministratorRoleCode(role.Code) {
			s.roleMenuIDs[role.ID] = append(s.roleMenuIDs[role.ID], menu.ID)
		}
	}
	return menu, ""
}

func (s *MemoryStore) UpdateMenu(id int, request models.MenuRequest) (models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findMenuIndexByID(id)
	if !found {
		return models.Menu{}, "菜单不存在"
	}
	if request.ParentID != nil {
		for parentID := request.ParentID; parentID != nil; {
			if *parentID == id {
				if *request.ParentID == id {
					return models.Menu{}, "父级菜单不能是自身"
				}
				return models.Menu{}, "父级菜单不能是当前菜单的下级"
			}
			parent, exists := s.findMenuByID(*parentID)
			if !exists {
				return models.Menu{}, "父级菜单不存在"
			}
			parentID = parent.ParentID
		}
	}
	s.menus[index].Name = request.Name
	s.menus[index].Code = request.Code
	s.menus[index].Path = request.Path
	s.menus[index].Icon = request.Icon
	s.menus[index].ParentID = request.ParentID
	s.menus[index].Sort = request.Sort
	s.menus[index].Status = request.Status
	s.menus[index].UpdatedAt = time.Now()
	return s.menus[index], ""
}

func (s *MemoryStore) DeleteMenu(id int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findMenuIndexByID(id)
	if !found {
		return "菜单不存在"
	}
	if s.hasChildMenu(id) {
		return "请先删除子菜单"
	}
	s.menus = append(s.menus[:index], s.menus[index+1:]...)
	for userID, menuIDs := range s.userMenuIDs {
		s.userMenuIDs[userID] = removeMenuID(menuIDs, id)
	}
	for departmentID, menuIDs := range s.departmentMenuIDs {
		s.departmentMenuIDs[departmentID] = removeMenuID(menuIDs, id)
	}
	for roleID, menuIDs := range s.roleMenuIDs {
		s.roleMenuIDs[roleID] = removeMenuID(menuIDs, id)
	}
	return ""
}

func (s *MemoryStore) ListUserMenus(userID int) ([]models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.findUserByID(userID)
	if !exists {
		return nil, "用户不存在"
	}
	menuIDs := append([]int(nil), s.userMenuIDs[userID]...)
	if user.DepartmentID != nil {
		if department, found := s.findDepartmentByID(*user.DepartmentID); found && department.Status == "启用" {
			menuIDs = append(menuIDs, s.departmentMenuIDs[*user.DepartmentID]...)
		}
	}
	if user.RoleID != nil {
		if role, found := s.findRoleByID(*user.RoleID); found && role.Status == "启用" {
			menuIDs = append(menuIDs, s.roleMenuIDs[*user.RoleID]...)
		}
	}
	menuIDs = s.expandMenuAncestors(menuIDs)
	assignedMenus := make([]models.Menu, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		if menu, found := s.findMenuByID(menuID); found {
			assignedMenus = append(assignedMenus, menu)
		}
	}
	return assignedMenus, ""
}

func (s *MemoryStore) ListUserActionPermissions(userID int) ([]string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.findUserByID(userID)
	if !exists {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return permissions.AllCodes(), ""
	}
	roleCodes := []string{}
	if user.RoleID != nil {
		if role, found := s.findRoleByID(*user.RoleID); found && role.Status == "启用" {
			roleCodes = permissions.RoleCodes(role.Code)
		}
	}
	return permissions.MergeCodes(roleCodes, s.userActionCodes[userID]), ""
}

func (s *MemoryStore) ListUserExtraMenus(userID int) ([]models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.userExists(userID) {
		return nil, "用户不存在"
	}
	menus := make([]models.Menu, 0, len(s.userMenuIDs[userID]))
	for _, menuID := range s.userMenuIDs[userID] {
		if menu, found := s.findMenuByID(menuID); found {
			menus = append(menus, menu)
		}
	}
	return menus, ""
}

func (s *MemoryStore) GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.findUserByID(userID)
	if !exists {
		return models.UserPermissionDetail{}, "用户不存在"
	}
	departmentMenuIDs := []int{}
	if user.DepartmentID != nil {
		departmentMenuIDs = append(departmentMenuIDs, s.departmentMenuIDs[*user.DepartmentID]...)
	}
	roleMenuIDs := []int{}
	if user.RoleID != nil {
		roleMenuIDs = append(roleMenuIDs, s.roleMenuIDs[*user.RoleID]...)
	}
	userMenuIDs := append([]int(nil), s.userMenuIDs[userID]...)
	effectiveDepartmentMenuIDs := departmentMenuIDs
	if user.DepartmentID != nil {
		if department, found := s.findDepartmentByID(*user.DepartmentID); !found || department.Status != "启用" {
			effectiveDepartmentMenuIDs = nil
		}
	}
	effectiveRoleMenuIDs := roleMenuIDs
	if user.RoleID != nil {
		if role, found := s.findRoleByID(*user.RoleID); !found || role.Status != "启用" {
			effectiveRoleMenuIDs = nil
		}
	}
	roleActionCodes := []string{}
	if user.RoleID != nil {
		if role, found := s.findRoleByID(*user.RoleID); found {
			roleActionCodes = permissions.RoleCodes(role.Code)
		}
	}
	userActionCodes := permissions.MergeCodes(s.userActionCodes[userID])
	effectiveActionCodes := permissions.MergeCodes(userActionCodes)
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		effectiveActionCodes = permissions.AllCodes()
	} else if user.RoleID != nil {
		if role, found := s.findRoleByID(*user.RoleID); found && role.Status == "启用" {
			effectiveActionCodes = permissions.MergeCodes(roleActionCodes, userActionCodes)
		}
	}
	return models.UserPermissionDetail{
		DepartmentMenuIDs:    uniqueIDs(departmentMenuIDs),
		RoleMenuIDs:          uniqueIDs(roleMenuIDs),
		UserMenuIDs:          uniqueIDs(userMenuIDs),
		EffectiveMenuIDs:     s.expandMenuAncestors(append(append(effectiveDepartmentMenuIDs, effectiveRoleMenuIDs...), userMenuIDs...)),
		RoleActionCodes:      roleActionCodes,
		UserActionCodes:      userActionCodes,
		EffectiveActionCodes: effectiveActionCodes,
	}, ""
}

func (s *MemoryStore) UpdateUserMenus(userID int, menuIDs []int) ([]int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.userExists(userID) {
		return nil, "用户不存在"
	}
	for _, menuID := range menuIDs {
		if !s.menuExists(menuID) {
			return nil, "包含不存在的菜单"
		}
	}
	s.userMenuIDs[userID] = uniqueIDs(menuIDs)
	return append([]int(nil), s.userMenuIDs[userID]...), ""
}

func (s *MemoryStore) UpdateUserActions(userID int, actionCodes []string) ([]string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.findUserByID(userID)
	if !exists {
		return nil, "用户不存在"
	}
	if permissions.IsAdministratorRoleCode(user.RoleCode) {
		return nil, "超级管理员和系统管理员动作权限固定为全部，不能修改"
	}
	codes, valid := permissions.NormalizeCodes(actionCodes)
	if !valid {
		return nil, "包含不存在的动作权限"
	}
	s.userActionCodes[userID] = codes
	return append([]string(nil), codes...), ""
}

func (s *MemoryStore) ListArticles() []models.Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.Article(nil), s.articles...)
}

func (s *MemoryStore) FindArticleByID(id int) (models.Article, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findArticleByID(id)
}

func (s *MemoryStore) CreateArticle(request models.ArticleRequest) models.Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	article := models.Article{ID: s.nextArticleID, Title: request.Title, Category: request.Category, Author: request.Author, Status: request.Status, Summary: request.Summary, Content: request.Content, Views: 0, CreatedAt: now, UpdatedAt: now}
	s.nextArticleID++
	s.articles = append(s.articles, article)
	return article
}

func (s *MemoryStore) UpdateArticle(id int, request models.ArticleRequest) (models.Article, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findArticleIndexByID(id)
	if !found {
		return models.Article{}, false
	}
	s.articles[index].Title = request.Title
	s.articles[index].Category = request.Category
	s.articles[index].Author = request.Author
	s.articles[index].Status = request.Status
	s.articles[index].Summary = request.Summary
	s.articles[index].Content = request.Content
	s.articles[index].UpdatedAt = time.Now()
	return s.articles[index], true
}

func (s *MemoryStore) DeleteArticle(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findArticleIndexByID(id)
	if !found {
		return false
	}
	s.articles = append(s.articles[:index], s.articles[index+1:]...)
	return true
}

func (s *MemoryStore) ListFiles() []models.ManagedFile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.ManagedFile(nil), s.files...)
}

func (s *MemoryStore) FindFileByID(id int) (models.ManagedFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findFileByID(id)
}

func (s *MemoryStore) CreateFile(file models.ManagedFile) models.ManagedFile {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	file.ID = s.nextFileID
	file.CreatedAt = now
	file.UpdatedAt = now
	s.nextFileID++
	s.files = append(s.files, file)
	return file
}

func (s *MemoryStore) UpdateFileMetadata(id int, request models.FileMetadataRequest) (models.ManagedFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findFileIndexByID(id)
	if !found {
		return models.ManagedFile{}, false
	}
	s.files[index].DisplayName = request.DisplayName
	s.files[index].Category = request.Category
	s.files[index].Description = request.Description
	s.files[index].UpdatedAt = time.Now()
	return s.files[index], true
}

func (s *MemoryStore) DeleteFile(id int) (models.ManagedFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findFileIndexByID(id)
	if !found {
		return models.ManagedFile{}, false
	}
	file := s.files[index]
	s.files = append(s.files[:index], s.files[index+1:]...)
	return file, true
}

func (s *MemoryStore) findUserByID(id int) (models.User, bool) {
	for _, user := range s.users {
		if user.ID == id {
			return user, true
		}
	}
	return models.User{}, false
}

func (s *MemoryStore) findUserByUsername(username string) (models.User, bool) {
	for _, user := range s.users {
		if strings.EqualFold(user.Username, strings.TrimSpace(username)) {
			return user, true
		}
	}
	return models.User{}, false
}

func (s *MemoryStore) findUserIndexByID(id int) (int, bool) {
	for index, user := range s.users {
		if user.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *MemoryStore) userExists(id int) bool {
	_, found := s.findUserByID(id)
	return found
}

func (s *MemoryStore) findDepartmentByID(id int) (models.Department, bool) {
	for _, department := range s.departments {
		if department.ID == id {
			return department, true
		}
	}
	return models.Department{}, false
}

func (s *MemoryStore) findDepartmentByCode(code string) (models.Department, bool) {
	for _, department := range s.departments {
		if strings.EqualFold(department.Code, strings.TrimSpace(code)) {
			return department, true
		}
	}
	return models.Department{}, false
}

func (s *MemoryStore) findDepartmentIndexByID(id int) (int, bool) {
	for index, department := range s.departments {
		if department.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *MemoryStore) resolveDepartment(departmentID *int, legacyName string) (*int, string, string) {
	if departmentID != nil {
		department, exists := s.findDepartmentByID(*departmentID)
		if !exists {
			return nil, "", "部门不存在"
		}
		id := department.ID
		return &id, department.Name, ""
	}
	name := strings.TrimSpace(legacyName)
	for _, department := range s.departments {
		if department.Name == name {
			id := department.ID
			return &id, department.Name, ""
		}
	}
	return nil, name, ""
}

func (s *MemoryStore) findRoleByID(id int) (models.Role, bool) {
	for _, role := range s.roles {
		if role.ID == id {
			return role, true
		}
	}
	return models.Role{}, false
}

func (s *MemoryStore) findRoleByCode(code string) (models.Role, bool) {
	for _, role := range s.roles {
		if strings.EqualFold(role.Code, strings.TrimSpace(code)) {
			return role, true
		}
	}
	return models.Role{}, false
}

func (s *MemoryStore) findRoleIndexByID(id int) (int, bool) {
	for index, role := range s.roles {
		if role.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *MemoryStore) resolveRole(roleID *int, legacyName string) (*int, string, string, string) {
	if roleID != nil {
		role, exists := s.findRoleByID(*roleID)
		if !exists {
			return nil, "", "", "角色不存在"
		}
		id := role.ID
		return &id, role.Name, role.Code, ""
	}
	name := strings.TrimSpace(legacyName)
	for _, role := range s.roles {
		if role.Name == name {
			id := role.ID
			return &id, role.Name, role.Code, ""
		}
	}
	return nil, "", "", "角色不存在"
}

func (s *MemoryStore) findMenuByID(id int) (models.Menu, bool) {
	for _, menu := range s.menus {
		if menu.ID == id {
			return menu, true
		}
	}
	return models.Menu{}, false
}

func (s *MemoryStore) dashboardMenuIDs() []int {
	for _, menu := range s.menus {
		if menu.Code == "dashboard" {
			return []int{menu.ID}
		}
	}
	return []int{}
}

func (s *MemoryStore) allMenuIDs() []int {
	ids := make([]int, 0, len(s.menus))
	for _, menu := range s.menus {
		ids = append(ids, menu.ID)
	}
	return ids
}

func (s *MemoryStore) findMenuIndexByID(id int) (int, bool) {
	for index, menu := range s.menus {
		if menu.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *MemoryStore) menuExists(id int) bool {
	_, found := s.findMenuByID(id)
	return found
}

func (s *MemoryStore) expandMenuAncestors(menuIDs []int) []int {
	expanded := uniqueIDs(menuIDs)
	seen := make(map[int]bool, len(expanded))
	for _, menuID := range expanded {
		seen[menuID] = true
	}
	for index := 0; index < len(expanded); index++ {
		menu, exists := s.findMenuByID(expanded[index])
		if !exists || menu.ParentID == nil || seen[*menu.ParentID] {
			continue
		}
		seen[*menu.ParentID] = true
		expanded = append(expanded, *menu.ParentID)
	}
	return expanded
}

func (s *MemoryStore) hasChildMenu(id int) bool {
	for _, menu := range s.menus {
		if menu.ParentID != nil && *menu.ParentID == id {
			return true
		}
	}
	return false
}

func (s *MemoryStore) findArticleByID(id int) (models.Article, bool) {
	for _, article := range s.articles {
		if article.ID == id {
			return article, true
		}
	}
	return models.Article{}, false
}

func (s *MemoryStore) findArticleIndexByID(id int) (int, bool) {
	for index, article := range s.articles {
		if article.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *MemoryStore) findFileByID(id int) (models.ManagedFile, bool) {
	for _, file := range s.files {
		if file.ID == id {
			return file, true
		}
	}
	return models.ManagedFile{}, false
}

func (s *MemoryStore) findFileIndexByID(id int) (int, bool) {
	for index, file := range s.files {
		if file.ID == id {
			return index, true
		}
	}
	return -1, false
}

func removeMenuID(menuIDs []int, removedID int) []int {
	filtered := make([]int, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		if menuID != removedID {
			filtered = append(filtered, menuID)
		}
	}
	return filtered
}

func uniqueIDs(ids []int) []int {
	seen := make(map[int]bool, len(ids))
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}

func sameIDs(left, right []int) bool {
	left = uniqueIDs(left)
	right = uniqueIDs(right)
	if len(left) != len(right) {
		return false
	}
	rightSet := make(map[int]bool, len(right))
	for _, id := range right {
		rightSet[id] = true
	}
	for _, id := range left {
		if !rightSet[id] {
			return false
		}
	}
	return true
}

func intPtr(value int) *int {
	return &value
}
