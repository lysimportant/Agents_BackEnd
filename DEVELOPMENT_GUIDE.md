# 全栈开发说明书

## 1. 项目结构

### 前端

`frontend/app/` 是 Next.js App Router 的入口层，只放路由、根布局和全局样式：

- `app/page.tsx`：登录后的工作台总挂载点，根据 `activePage` 渲染业务页面。
- `app/layout.tsx`：全局 metadata、主题初始化、Ant Design Provider、全局反馈和全局 CSS。
- `app/chat/[conversationId]/page.tsx`：公开在线聊天页面路由。
- `app/socket/chat/[conversationId]/page.tsx`：历史兼容路由，内部复用同一个聊天页面。

业务代码位于 `frontend/src/`：

- `src/admin-pages/`：用户、部门、角色、菜单、文章、文件、个人资料等管理页面；使用该名称避免与 Next.js Pages Router 冲突。
- `src/features/`：跨页面业务功能及状态，例如在线聊天、工作台 hooks。
- `src/components/`：可复用 UI、布局、富文本编辑器、表格和全站反馈组件。
- `src/services/`：API 请求封装和文件服务。
- `src/utils/`：无状态工具、权限判断、文章导出、菜单树等。
- `src/config/`：API 地址、页面 key、标题和默认表单。
- `src/types/`：共享 TypeScript 类型。
- `src/theme/`：主题定义和主题启动脚本。
- `src/styles/`：全局增强样式。

### 后端

- `main.go`：加载配置、数据库、迁移种子并启动 Gin。
- `routes/`：注册 URL、HTTP 方法和权限中间件。
- `handlers/`：解析请求、调用 repository、返回 JSON 或文件响应。
- `repository/`：SQLite 查询、写入、迁移和种子；handler 不直接写 SQL。
- `models/`：请求、响应和领域模型。
- `permissions/`：稳定的 `resource.action` 权限编码。
- `middleware/`：登录、菜单、动作权限和 CORS。
- `utils/`：无状态公共工具。
- `data/`、`uploads/`：正式业务数据，测试和验收不得清空或覆盖。

## 2. 新增前端管理页面

1. 在 `frontend/src/admin-pages/<domain>/` 新建 `<Domain>Page.tsx`，页面只负责展示、表单和交互。
2. 在 `frontend/src/types/admin.ts` 增加实体、表单和响应类型。
3. 在 `frontend/src/config/constants.ts` 增加 `PageKey`、`pageTitles` 和必要的表单默认值。
4. 如果页面需要菜单，在 `backend/repository/sqlite_store.go` 的应用菜单种子中增加 `code/path/name/icon/sort`。
5. 在 `frontend/src/features/workspace/useAdminWorkspace.ts` 增加加载、保存、删除、筛选和错误状态。
6. 在 `frontend/app/page.tsx` 导入页面，并通过 `workspace.activePage === '<page-key>'` 挂载。
7. 在 `frontend/src/components/layout/MainLayout.tsx` 增加图标映射；如果需要动作按钮，在前后端权限定义中同步增加。
8. 页面专属样式放在页面目录；跨页面样式放在 `src/styles/` 或 `app/globals.css`。

## 3. 新增后端接口

1. 在 `backend/models/models.go` 或领域模型文件定义请求和响应结构，JSON 使用 camelCase。
2. 在 `backend/repository/<domain>.go` 增加查询和写入方法，涉及 SQLite 表时同步检查迁移、扫描列顺序和幂等性。
3. 在 `backend/handlers/<domain>.go` 增加 handler：鉴权、参数校验、调用 repository、返回状态码和错误信息。
4. 在 `backend/routes/<domain>.go` 注册路径，按需要绑定 `RequireAuth`、`RequireMenu`、`RequireAction`。
5. 在 `frontend/src/services/` 增加 API 函数，统一复用 `requestWithSession` 并携带 Cookie。
6. 更新前端类型、workspace hook、页面调用方和 `README.md` 的公开 API 说明。
7. 后端接口测试必须使用 `t.TempDir()`、临时 SQLite 和临时上传目录；不得写入正式业务数据。

## 4. 在线聊天功能联动

在线聊天的内部实现仍位于 `src/features/chat/` 和后端 `handlers/socket.go`、`repository/socket.go`、`routes/socket.go`。`socket` 是技术实现名称，用户界面统一显示“在线聊天”。公开入口使用 `/chat/[conversationId]`，旧 `/socket/chat/[conversationId]` 保留兼容。

## 5. 验证与提交

```powershell
cd backend
go test ./...
go vet ./...

cd ../frontend
.\node_modules\.bin\tsc.cmd --noEmit --incremental false
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8000"
npm run build
```

联调服务使用前端 `3000`、后端 `8000`；检查 `/health` 和首页返回 200。用户验收后才提交，commit 摘要必须使用中文并以 `Pn` 开头，正文详细说明改动和验证结果。
