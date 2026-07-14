package store

import (
	"strings"
	"sync"
	"time"

	"collector-backend/auth"
	"collector-backend/models"
)

type MemoryStore struct {
	mu            sync.Mutex
	dataPoints    []models.DataPoint
	users         []models.User
	menus         []models.Menu
	articles      []models.Article
	files         []models.ManagedFile
	userMenuIDs   map[int][]int
	nextUserID    int
	nextMenuID    int
	nextArticleID int
	nextFileID    int
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
			{ID: 1, Username: "MH", Name: "MH", Role: "系统管理员", Department: "信息中心", Status: "在岗", Shift: "常白班", Phone: "", Email: "mh@example.com", PasswordHash: auth.MustHashPassword("123"), CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{ID: 2, Username: "zhang.gong", Name: "张工", Role: "产线主管", Department: "总装一线", Status: "在岗", Shift: "早班", Phone: "13800000001", Email: "zhang.gong@example.com", PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: 3, Username: "li.min", Name: "李敏", Role: "质量工程师", Department: "质量中心", Status: "巡检", Shift: "中班", Phone: "13800000002", Email: "li.min@example.com", PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-90 * time.Minute)},
			{ID: 4, Username: "wang.qiang", Name: "王强", Role: "设备维护", Department: "设备保障", Status: "待命", Shift: "夜班", Phone: "13800000003", Email: "wang.qiang@example.com", PasswordHash: auth.MustHashPassword("123456"), CreatedAt: now.Add(-36 * time.Hour), UpdatedAt: now.Add(-30 * time.Minute)},
		},
		menus: []models.Menu{
			{ID: 1, Name: "生产看板", Code: "production.dashboard", Path: "/production", Icon: "数据", ParentID: nil, Sort: 10, Status: "启用", CreatedAt: now.Add(-96 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
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
		nextUserID:    5,
		nextMenuID:    5,
		nextArticleID: 4,
		nextFileID:    1,
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
	user := models.User{ID: s.nextUserID, Username: username, Name: request.Name, Role: request.Role, Department: request.Department, Status: request.Status, Shift: request.Shift, Phone: request.Phone, Email: request.Email, PasswordHash: passwordHash, CreatedAt: time.Now(), UpdatedAt: time.Now()}
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
	username := strings.TrimSpace(request.Username)
	if existing, exists := s.findUserByUsername(username); exists && existing.ID != id {
		return models.User{}, "账号已存在"
	}
	s.users[index].Username = username
	s.users[index].Name = request.Name
	s.users[index].Role = request.Role
	s.users[index].Department = request.Department
	s.users[index].Status = request.Status
	s.users[index].Shift = request.Shift
	s.users[index].Phone = request.Phone
	s.users[index].Email = request.Email
	if passwordHash != "" {
		s.users[index].PasswordHash = passwordHash
	}
	s.users[index].UpdatedAt = time.Now()
	return s.users[index], ""
}

func (s *MemoryStore) DeleteUser(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, found := s.findUserIndexByID(id)
	if !found {
		return false
	}
	s.users = append(s.users[:index], s.users[index+1:]...)
	delete(s.userMenuIDs, id)
	return true
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
		if *request.ParentID == id {
			return models.Menu{}, "父级菜单不能是自身"
		}
		if !s.menuExists(*request.ParentID) {
			return models.Menu{}, "父级菜单不存在"
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
	return ""
}

func (s *MemoryStore) ListUserMenus(userID int) ([]models.Menu, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.userExists(userID) {
		return nil, "用户不存在"
	}
	assignedMenus := make([]models.Menu, 0, len(s.userMenuIDs[userID]))
	for _, menuID := range s.userMenuIDs[userID] {
		if menu, found := s.findMenuByID(menuID); found {
			assignedMenus = append(assignedMenus, menu)
		}
	}
	return assignedMenus, ""
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

func (s *MemoryStore) findMenuByID(id int) (models.Menu, bool) {
	for _, menu := range s.menus {
		if menu.ID == id {
			return menu, true
		}
	}
	return models.Menu{}, false
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

func intPtr(value int) *int {
	return &value
}
