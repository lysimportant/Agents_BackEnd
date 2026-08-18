# 新建页面与后端创建接口开发说明

这份说明用于新增 B 端工作台业务模块、后端 API，或扩展 C 端公开页面。B 端示例名称使用 `products`（商品），实际开发时请替换成真实业务名称。

核心原则：先确定数据和权限，再实现后端接口，最后接入前端页面。前后端 JSON 字段统一使用 `camelCase`，所有写请求都必须在后端再次鉴权。

## 一、调用链

```text
浏览器页面
  -> src/services/<domain>Api.ts
  -> requestWithSession()
  -> /api/<domain>
  -> routes/<domain>.go
  -> RequireAuth / RequireMenu / RequireAction
  -> handlers/<domain>.go
  -> repository/<domain>.go
  -> SQLite
```

管理后台目前只有根页面 `/`。新增管理功能通常不是创建 `/products`，而是让 `app/page.tsx` 根据 `activePage === 'products'` 渲染 `ProductsPage`。

C 端 `portal/` 使用真实 App Router URL，不使用 `activePage`。公开页面位于 `portal/app/[locale]/`，内容请求集中在 `portal/src/services/publicApi.ts`，只能读取 `/api/public/*`；登录、会话恢复和退出是唯一允许调用的认证接口。

## 二、后端开发顺序

### 1. 模型和请求体

在 `backend/models/models.go` 添加返回模型和创建/更新请求体。字段要使用能直接表达业务含义的名称，并为每个字段补充中文注释：

```go
// Product 表示商品管理页面中的商品记录。
type Product struct {
    // ID 表示数据库主键。
    ID int `json:"id"`
    // Name 表示商品名称。
    Name string `json:"name"`
    // Price 表示商品单价。
    Price int64 `json:"price"`
    // Status 表示商品状态。
    Status string `json:"status"`
    // OwnerID 表示创建人的用户标识。
    OwnerID int `json:"ownerId"`
    // CreatedAt 表示创建时间。
    CreatedAt time.Time `json:"createdAt"`
    // UpdatedAt 表示最后更新时间。
    UpdatedAt time.Time `json:"updatedAt"`
}

// ProductRequest 表示创建或更新商品时允许客户端提交的字段。
type ProductRequest struct {
    // Name 表示商品名称，不能为空。
    Name string `json:"name" binding:"required"`
    // Price 表示商品单价，服务端仍需检查数值范围。
    Price int64 `json:"price" binding:"gte=0"`
    // Status 表示商品状态。
    Status string `json:"status" binding:"required"`
}
```

不要把密码、内部磁盘路径、存储文件名等内部字段放进对外响应；需要隐藏的 Go 字段使用 ``json:"-"``。

### 2. SQLite 表、迁移和菜单

在 `backend/repository/sqlite_store.go` 的 `MigrateAndSeed` 表定义区域增加 `CREATE TABLE IF NOT EXISTS`，必须做到可重复执行：

```sql
CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    price INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '启用',
    owner_id INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);
```

同时更新 `docs/database/schema.md`，记录表用途、字段含义、索引和删除语义。不要手工修改 `backend/data/app.db`；正式数据库由服务启动时迁移，测试数据库必须使用 `t.TempDir()`。

如果新页面需要出现在工作台，给 `seed()` 的菜单种子增加稳定的 `code`、`path`、父菜单和排序。菜单种子必须幂等，不能因为升级删除用户已有菜单或权限。

### 3. repository

简单模块可以放在 `sqlite_store.go`，有独立业务行为时建议创建 `backend/repository/products.go`。repository 只负责数据库读写和事务，不负责 HTTP 状态码：

```go
// ListProducts 按最新更新时间优先返回商品。
func (s *SQLiteStore) ListProducts() []models.Product { /* 查询并扫描字段 */ }

// CreateProduct 保存商品并返回带数据库 ID 的记录。
func (s *SQLiteStore) CreateProduct(product models.Product) models.Product { /* INSERT */ }

// UpdateProduct 更新指定商品；找不到记录时返回 false。
func (s *SQLiteStore) UpdateProduct(id int, request models.ProductRequest) (models.Product, bool) { /* UPDATE */ }

// DeleteProduct 删除指定商品；需要软删除时改为写入 deleted_at。
func (s *SQLiteStore) DeleteProduct(id int) bool { /* DELETE 或软删除 */ }
```

所有 SQL 参数使用占位符，不拼接用户输入。写入涉及多张表时使用事务，并在失败时回滚。

### 4. handler

在 `backend/handlers/products.go` 定义最小 store 接口和 handler。handler 负责解析请求、取得当前用户、调用 repository、选择 HTTP 状态码：

```go
// ProductStore 定义商品 handler 需要的持久化能力。
type ProductStore interface {
    ListProducts() []models.Product
    CreateProduct(product models.Product) models.Product
    UpdateProduct(id int, request models.ProductRequest) (models.Product, bool)
    DeleteProduct(id int) bool
}

// Create 创建商品并返回 201。
func (h *ProductHandler) Create(c *gin.Context) {
    var request models.ProductRequest
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    user, ok := middleware.CurrentUser(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
        return
    }
    created := h.store.CreateProduct(models.Product{
        Name: request.Name, Price: request.Price, Status: request.Status,
        OwnerID: user.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
    })
    c.JSON(http.StatusCreated, created)
}
```

列表、详情、更新、删除建议统一使用：

| 操作 | 方法 | 成功状态 |
| --- | --- | --- |
| 列表 | `GET /api/products` | `200` |
| 详情 | `GET /api/products/:id` | `200` |
| 创建 | `POST /api/products` | `201` |
| 更新 | `PUT /api/products/:id` | `200` |
| 删除 | `DELETE /api/products/:id` | `204` 或项目现有约定 |

### 5. 路由、权限和依赖

在 `backend/routes/products.go` 注册路由，并在 `backend/routes/routes.go` 的 `Store` 接口中加入 handler 依赖：

```go
func registerProductRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.ProductHandler) {
    requireMenu := middleware.RequireMenu(store, "products")
    routes.GET("/products", requireMenu, middleware.RequireAction(store, permissions.ProductsQuery), handler.List)
    routes.POST("/products", requireMenu, middleware.RequireAction(store, permissions.ProductsCreate), handler.Create)
    routes.PUT("/products/:id", requireMenu, middleware.RequireAction(store, permissions.ProductsUpdate), handler.Update)
    routes.DELETE("/products/:id", requireMenu, middleware.RequireAction(store, permissions.ProductsDelete), handler.Delete)
}
```

在 `backend/permissions/actions.go` 增加稳定动作编码，并在后端启动种子中给默认角色配置合理的查询/查看权限。安全判断使用 `roleCode`，不能根据角色显示名称判断。最后在 `routes.Setup` 的受保护路由组中调用 `registerProductRoutes(...)`；忘记这一步时，handler 虽然编译通过，接口仍然不存在。

### 6. OpenAPI 和测试

在 `docs/api/openapi.yaml` 增加路径、请求体、响应体、Cookie 鉴权方式和 `400/401/403/404` 错误响应。接口路径、JSON 字段或权限改变时，README 也要同步。

至少补充：

- repository 测试：使用 `t.TempDir()` 创建独立 SQLite 和上传目录。
- route 测试：登录用户可以创建；无菜单、无动作、未登录分别得到 `401/403`。
- 业务测试：必填字段、重复编码、越权修改、删除不存在记录等边界。

后端交付前执行：

```powershell
cd backend
gofmt -w models/models.go handlers/products.go repository/products.go routes/products.go permissions/actions.go
go test ./...
go vet ./...
```

## 三、前端开发顺序

### 1. 类型、PageKey 和默认表单

在 `frontend/src/types/admin.ts` 增加 `Product`、`ProductForm`。在 `frontend/src/config/constants.ts`：

1. 将 `'products'` 加入 `PageKey` 和 `pageKeys`。
2. 在 `pageTitles` 增加页面中文标题。
3. 增加 `emptyProductForm`，用于重置表单。

### 2. API 服务

在 `frontend/src/services/productsApi.ts` 集中封装请求。需要登录的请求统一调用 `requestWithSession`，它会携带 HttpOnly Cookie：

```ts
export async function createProduct(form: ProductForm) {
  const response = await requestWithSession(`${API_BASE_URL}/api/products`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(form),
  });
  if (!response.ok) throw new Error('创建商品失败');
  return (await response.json()) as Product;
}
```

不要在页面组件中重复拼接 API 地址，也不要把 Cookie、密码或会话 ID 存进 Web Storage。写请求不要自动重试，避免重复创建。

### 3. `useAdminWorkspace` 状态编排

新增 `products`、`productForm`、`editingProductId`、`isSavingProduct` 等状态，以及 `loadProducts`、`handleSaveProduct`、`handleDeleteProduct`、`resetProductForm` 等动作。

这个 hook 统一处理加载、错误、成功提示、刷新列表和权限失败；页面组件只负责展示和触发 props 回调。登录成功后由工作台按有效菜单加载资源，不要让页面自行绕过菜单权限请求全部数据。

### 4. 页面组件

在 `frontend/src/admin-pages/products/ProductsPage.tsx` 创建页面，在同目录创建 `ProductsPage.module.css`：

- 通过 props 接收列表、表单、加载状态、动作权限和回调。
- 表单提交按钮根据 `canCreate`/`canUpdate` 显示或禁用。
- 显示加载中、空数据、提交中、错误和成功后的刷新状态。
- 页面内部函数、类型字段、局部状态和回调补充中文 JSDoc/注释。
- 桌面端和 390px 移动端都不能横向溢出。

### 5. 挂载到根页面和侧栏

在 `frontend/app/page.tsx` 导入 `ProductsPage`，计算 `products.create/update/delete` 动作权限，增加 `workspace.activePage === 'products'` 的渲染分支，并把 workspace 状态和回调通过 props 传入。

在 `frontend/src/components/layout/MainLayout.tsx`：

- 在 `menuIconByCode` 增加 `products` 图标。
- 确认 `resolvePageKey` 同时匹配菜单 `code` 和去除斜杠后的 `path`。
- 检查父菜单展开、面包屑、刷新后 `sessionStorage` 恢复和移动端侧栏动画。

## 四、C 端公开功能开发

C 端页面使用 `/{locale}/...` 真实路由。新增或修改公开功能时按以下位置联动：

- 路由和 metadata：`portal/app/[locale]/`，同时检查 canonical、语言替换、sitemap、robots 和 RSS。
- 展示组件：`portal/src/components/`；跨页面交互状态放 `portal/src/features/`。
- API 与类型：`portal/src/services/publicApi.ts`、`portal/src/types/publicContent.ts`。
- 三语言文案：`portal/src/i18n/messages/zh-CN.json`、`en-US.json`、`ja-JP.json`。
- 公开契约：`backend/routes/public.go`、公开 handler/repository、`docs/api/openapi.yaml`。

图片页虽然不显示分页按钮，仍使用后端 `page/pageSize` 作为内部批次协议。接近底部时通过 `IntersectionObserver` 加载下一批；追加数据不得重新分配已加载卡片的瀑布流列。

C 端验证命令：

```powershell
cd portal
npm run typecheck
npm run lint
npm run docs
npm run build
```

## 五、一次完整的开发顺序

1. 先写字段、权限和 API 草图。
2. 后端增加模型、迁移、repository、handler、route、权限和测试。
3. 更新 OpenAPI、数据库字典和 README。
4. 前端增加类型、常量、API 服务、workspace 状态和页面组件。
5. 在根页面挂载组件，在侧栏补图标和菜单映射。
6. 启动隔离后端，登录后验证创建、列表、更新、删除和越权。
7. 再执行前端类型检查、生产构建和浏览器验收。

## 六、联调验收清单

后端：

- `GET /health` 正常。
- 未登录请求返回 `401`。
- 无菜单或动作权限返回 `403`。
- 创建成功返回 `201` 和完整记录。
- 刷新后数据仍存在，更新和删除结果正确。
- 测试只写入 `.workspace-temp/<任务名>/`。

前端：

- 侧栏可以打开新页面，刷新后仍停留在当前页。
- 表单字段与 API 的 `camelCase` 一致。
- 创建、编辑、删除按钮按权限显示。
- 加载、空数据、错误、提交中和成功提示完整。
- 桌面端无异常大留白，390px 宽度无横向滚动。
- 浏览器控制台没有新增 warning/error。

命令：

```powershell
cd backend
go test ./...
go vet ./...

cd ..\frontend
npx.cmd tsc --noEmit --incremental false
npm.cmd run build

cd ..\portal
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run docs
npm.cmd run build
```

只有涉及 C 端或公开 API 时才需要执行 portal 命令；只有涉及 B 端时才需要执行 frontend 命令。

## 七、常见遗漏

- 只写了页面，没有在 `routes.Setup` 注册接口。
- 只加了后端路由，没有把菜单种子和权限动作补齐。
- 把页面当成 `/products`，但本项目管理页实际由 `/` 的 `activePage` 切换。
- 把 C 端也做成 `activePage`，或从 C 端调用后台写接口。
- 图片自动加载时抛弃后端分页协议，导致无法限制单次响应和恢复加载状态。
- 写请求没有 `credentials: 'include'`，导致登录 Cookie 没有发送。
- 修改 JSON 字段却没有同步模型、前端类型、OpenAPI 和 README。
- 用角色显示名称做安全判断，或只隐藏按钮而没有后端鉴权。
- 测试时直接使用 `backend/data/app.db` 或 `backend/uploads/`。
