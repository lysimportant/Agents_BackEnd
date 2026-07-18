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
- `app/components/`：项目级共享组件，包括富文本编辑器、关联用户弹窗和全站 3D 卡片增强。
- `components/ui/`、根 `hooks/` 和根 `lib/`：shadcn/ui 共享基础设施。

## 页面与状态模型

- App Router 只有根页面 `/`；`app/page.tsx` 根据 `activePage` 条件渲染各业务页，不要假设 `/users`、`/files` 等独立 URL 已存在。
- 业务状态管理使用 React hooks，核心集中在 `app/hooks/useAdminWorkspace.ts`，当前没有 Redux、Zustand、MobX 或 React Query。
- `useAdminWorkspace` 负责会话恢复、按有效菜单加载资源、CRUD、筛选、表单、权限保存、加载状态和全局错误；页面组件通过 props 接收状态与动作。
- 当前页面写入 `sessionStorage` 的 `collector:active-page`，刷新后恢复；退出登录会清理。侧栏父菜单展开根据异步菜单树与 `activePage` 推导，修改导航时必须验证刷新恢复与父菜单展开。
- 主题使用 `localStorage` 持久化，并通过 `ADMIN_THEME_BOOTSTRAP_SCRIPT` 在首屏应用，避免水合前闪烁。新增主题时同步主题定义、CSS 变量和 Ant Design token。
- 后端 HttpOnly Cookie 是唯一认证凭据；前端不得把密码、会话 ID 或验证码持久化到 Web Storage。

## 实现约定

- 现有界面同时使用 Ant Design、Lucide、shadcn/ui、Tailwind CSS 4 和原生 CSS；修改时优先沿用相邻代码的组件和样式体系，避免为单个功能再引入一套 UI 依赖。
- 仅在需要浏览器 API、状态或事件时使用 Client Component。业务状态和 API 流程优先集中在 `useAdminWorkspace`，纯展示逻辑保留在功能组件。
- 共享业务类型、常量和请求逻辑分别放入 `app/types/` 与 `app/lib/`，不要在多个页面重复定义 API 结构。
- 后端地址取自 `NEXT_PUBLIC_API_BASE_URL`，默认 `http://localhost:8080`。
- 需要会话的请求复用 `requestWithSession`：固定 `credentials: 'include'`，单次超时 12 秒，仅 GET/HEAD/OPTIONS 在网络错误时按 350ms、900ms 重试；写请求禁止自动重试，以免产生重复副作用。
- API JSON 字段保持 camelCase。接口变更时同步检查后端模型、路由、权限以及 `README.md`。
- 用户有效菜单是所属部门、所属角色和个人附加菜单的并集；部门权限在部门页维护，角色权限在角色页维护，个人额外权限保留在用户页维护。
- 工作台图表使用 ECharts，数字动画使用 react-countup；图表颜色必须跟随当前主题变量。
- 保持桌面端和移动端布局可用，新增交互时检查加载、空数据、错误、禁用和提交中状态。

## 权限与导航约定

- `app/types/admin.ts` 中的 `PageKey`、`app/lib/constants.ts` 中的 `pageKeys/pageTitles`、`MainLayout.resolvePageKey` 与 `app/page.tsx` 的条件渲染必须保持一致。
- 菜单节点只有在 `code` 与去除斜杠后的 `path` 都匹配受支持页面时才映射为 `PageKey`；父级分组可以没有页面路径。
- 动作编码后端权威源是 `backend/permissions/actions.go`，前端镜像位于 `app/lib/actionPermissions.ts`。新增或重命名动作时两端与按钮显隐要同步。
- `super-admin`、`system-admin` 的判断使用 `app/lib/roleAccess.ts` 的稳定编码。不要用“超级管理员”等显示文字做安全判断。
- 页面不可只靠隐藏按钮保护操作；所有写请求必须接受并正确呈现后端 401/403/4xx 响应。

## 样式、组件与动画

- `app/globals.css` 是主样式入口，同时导入 Tailwind、shadcn 和 `article-file-enhancements.css`。修改全局选择器前先搜索同名规则和后置覆盖，避免因层叠顺序使布局失效。
- 相邻业务页主要使用 Ant Design 和原生 CSS，`components/ui/` 是 shadcn 基础组件；不要为了单个页面混入第三套新组件库。
- Ant Design 响应式布局优先使用真实 `Row/Col`，避免用宽泛的子元素 `display` 规则覆盖其 flex/grid 行为。
- 全站卡片效果由根布局中的 `TiltCardEffects` 通过事件委托和单个 `requestAnimationFrame` 自动增强；显式新卡可使用 `TiltCard/TiltCardLayer`。通过 `data-tilt-disabled="true"` 排除表单或交互密集容器。
- 3D 卡片只更新 CSS 变量和 transform；保持触摸设备与 `prefers-reduced-motion` 禁用逻辑。文件管理外层 `.file-browser-panel` 必须保持普通静态 Card，内部 `.file-card` 才启用效果。
- 弹窗和折叠权限区域要检查文字截断、Tooltip、左右留白以及 390px 移动端无横向溢出。

## 文章与文件能力

- 后端 `/api/articles/export` 是文章集合 CSV/PDF；前端 `articleExport.ts` 是单篇文章 CSV、打印/PDF、Word、PNG、Markdown、SEO HTML，两者不是同一实现。
- Markdown 目录与锚点逻辑位于 `articleMarkdown.ts`：正文有标题时生成目录、显式锚点和重复标题唯一后缀；无标题时不生成目录。
- 文章导出会处理跨源媒体、图片内联、分页画布和可见内容检查；修改时必须实际导出至少一个含标题和图片的样本验证文件内容。
- 前端和后端上传限制均为 32 MiB。文件读取/元数据/文本内容/永久删除复用 `app/lib/fileApi.ts`，业务编排仍位于 `useAdminWorkspace` 与 `FilesPage`。
- 文件删除默认移入回收站；永久删除按钮和请求只能在用户明确确认后触发，测试不能用正式业务 API 清理样本。

## 开发与验证

```powershell
cd frontend
npm install
npm run dev
```

已有 `node_modules` 且锁文件未变化时无需重复安装；全新环境或 CI 使用 `npm ci` 验证锁文件可复现安装，新增/升级依赖使用明确的 `npm install <package>` 并检查 `package.json`、`package-lock.json`。默认访问 `http://localhost:3000`，联调时后端通常运行在 `http://localhost:8080`。

提交前至少执行：

```powershell
.\node_modules\.bin\tsc.cmd --noEmit --incremental false
npm run build
```

当前没有自动化测试脚本。`package.json` 中的 `npm run lint` 仍调用已被 Next.js 16 CLI 移除的 `next lint`，在改为有效的 ESLint CLI 命令前，不把它当作可用验收项。

生产模式本地验证应先完成构建，再运行 `npm start`；开发服务器使用 `npm run dev`。构建与开发服务共享 `.next`，不要在开发服务写入 `.next` 时并发执行生产构建；需要构建时先停止开发服务，构建后再重启目标模式。

涉及布局、颜色、动画、响应式或交互状态的修改，构建通过后还必须使用官方 Browser 插件打开实际页面，在桌面端和移动端检查可见性、对比度、溢出、遮挡和交互结果。若官方插件或检测依赖异常，按根目录的恢复规则诊断、安装缺失的官方依赖并重试；不得改用未获用户指定的第三方浏览器工具或把插件错误当作视觉验收完成。

浏览器验收至少检查目标页面、刷新后的当前位置与侧栏展开、控制台 warning/error、`documentElement.scrollWidth - clientWidth`、390px 移动端，以及 `prefers-reduced-motion`/触摸降级（涉及动画时）。文件/文章/用户权限联调使用隔离后端数据，不对正式数据创建再批量删除验收记录。

不要手工编辑 `.next/`、`node_modules/`、`tsconfig.tsbuildinfo` 等生成内容。涉及文件上传、删除或恢复的联调和浏览器验收时，后端必须使用独立临时数据库与上传目录，禁止借助真实业务 API 清理用户数据。

`next-env.d.ts` 由 Next.js 构建生成，可能在 dev/build 类型路径之间变化；不要手工修改，提交前只在确有必要时纳入。`frontend/file-manager-preview.png` 是已跟踪的视觉参考，不要在普通验收中覆盖。
