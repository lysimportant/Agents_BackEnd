# frontend/AGENTS.md

## 适用范围

本文件适用于 `frontend/` 及其所有子目录，并补充仓库根目录的 `AGENTS.md`。

## 目录职责

本目录是采集数据管理平台的浏览器端管理后台，使用 Next.js 16 App Router、React 和 strict TypeScript，界面文案默认使用简体中文。

- `app/layout.tsx`：全局布局、metadata、Ant Design registry 和全局样式入口。
- `app/page.tsx`：当前单页管理后台的客户端编排入口；未登录时展示认证页，登录后在主布局内切换各功能页。
- `app/hooks/useAdminWorkspace.ts`：认证状态、业务状态以及 CRUD/API 流程的核心编排。
- `app/layout/MainLayout.tsx`：导航、响应式侧栏、主题和全屏交互。
- `app/auth/`、`app/dashboard/`、`app/users/`、`app/departments/`、`app/roles/`、`app/menus/`、`app/articles/`、`app/files/`：各业务功能页面。
- `app/lib/`：请求、常量、菜单树、文件 API 和状态工具；`app/types/`：共享业务类型。
- `app/theme/`：10 套全站主题定义、CSS 变量映射、首屏初始化和本地持久化。
- `components/ui/`、根 `hooks/` 和根 `lib/`：shadcn/ui 共享基础设施。

## 实现约定

- 现有界面同时使用 Ant Design、Lucide、shadcn/ui、Tailwind CSS 4 和原生 CSS；修改时优先沿用相邻代码的组件和样式体系，避免为单个功能再引入一套 UI 依赖。
- 仅在需要浏览器 API、状态或事件时使用 Client Component。业务状态和 API 流程优先集中在 `useAdminWorkspace`，纯展示逻辑保留在功能组件。
- 共享业务类型、常量和请求逻辑分别放入 `app/types/` 与 `app/lib/`，不要在多个页面重复定义 API 结构。
- 后端地址取自 `NEXT_PUBLIC_API_BASE_URL`，默认 `http://localhost:8080`。
- 需要会话的请求复用 `requestWithSession`，保持 `credentials: 'include'`、现有超时行为和仅对幂等读取请求重试的约束。
- API JSON 字段保持 camelCase。接口变更时同步检查后端模型、路由、权限以及 `README.md`。
- 用户有效菜单是所属部门、所属角色和个人附加菜单的并集；部门权限在部门页维护，角色权限在角色页维护，个人额外权限保留在用户页维护。
- 工作台图表使用 ECharts，数字动画使用 react-countup；图表颜色必须跟随当前主题变量。
- 保持桌面端和移动端布局可用，新增交互时检查加载、空数据、错误、禁用和提交中状态。

## 开发与验证

```powershell
cd frontend
npm install
npm run dev
```

默认访问 `http://localhost:3000`，联调时后端通常运行在 `http://localhost:8080`。提交前至少执行：

```powershell
npx tsc --noEmit --incremental false
npm run build
```

当前没有自动化测试脚本。`package.json` 中的 `npm run lint` 仍调用已被 Next.js 16 CLI 移除的 `next lint`，在改为有效的 ESLint CLI 命令前，不把它当作可用验收项。

涉及布局、颜色、动画、响应式或交互状态的修改，构建通过后还必须使用官方 Browser 插件打开实际页面，在桌面端和移动端检查可见性、对比度、溢出、遮挡和交互结果。若官方插件或检测依赖异常，按根目录的恢复规则诊断、安装缺失的官方依赖并重试；不得改用未获用户指定的第三方浏览器工具或把插件错误当作视觉验收完成。

不要手工编辑 `.next/`、`node_modules/`、`tsconfig.tsbuildinfo` 等生成内容。涉及文件上传、删除或恢复的联调和浏览器验收时，后端必须使用独立临时数据库与上传目录，禁止借助真实业务 API 清理用户数据。
