# frontend/AGENTS.md

## 操作前规则读取

- 每次收到新的用户任务、补充要求或任务方向变化后，在对 `frontend/` 执行任何文件检索、读取、命令、编辑、构建、测试、提交或推送之前，必须重新读取仓库根目录的 `../AGENTS.md` 和本文件。
- 不得仅依赖历史对话、上下文摘要或之前读取过的规则；每一轮前端任务操作都必须以工作区当前版本的两份 `AGENTS.md` 为准。

## 代码文档与命名

- 业务源码中的文档注释和解释性行内注释必须使用简体中文；协议名、库名、标识符和代码字面量可以保留英文。
- 类型、React 组件、钩子、服务函数、共享状态、模块级常量、页面内部函数、回调和局部变量都必须使用中文 JSDoc/TypeDoc 或解释性注释，说明用途、参数、返回值和副作用；页面外层已有注释不能代替页面内部声明的注释。
- 使用 `loadConversationMessages`、`selectedAttachmentIds`、`isEmojiPickerOpen` 等可直接理解的领域名称；避免新增 `data`、`info`、`item`、`handle` 等模糊名称。
- 注释必须说明变量保存的业务内容、函数执行的行为或状态变化；不得添加“保存数据”一类不能补充业务语义的注释。
- HTTP 契约必须与 `../docs/api/openapi.yaml` 保持同步；TypeDoc 配置位于 `typedoc.json`。

## 目录与开发流程

前端目录按职责组织：`app/` 只保留 Next.js App Router 路由入口、全局布局和全局样式；`src/admin-pages/` 存放管理页面；`src/features/` 存放跨页面业务功能；`src/components/` 存放共享组件；`src/services/` 存放 API 请求与后端服务封装；`src/utils/` 存放无状态工具；`src/config/` 存放常量；`src/types/` 存放共享类型；`src/theme/` 存放主题；`src/styles/` 存放全局增强样式。管理页面目录不能命名为 `src/pages/`，避免与 Next.js Pages Router 冲突。

新增管理页面时，在 `src/admin-pages/<domain>/` 创建页面组件；在 `src/types/admin.ts` 与 `src/config/constants.ts` 增加类型、PageKey、标题及默认表单；在后端 `repository/sqlite_store.go` 增加菜单种子；然后在 `app/page.tsx` 按 `activePage` 挂载，并在 `src/components/layout/MainLayout.tsx` 补充导航图标映射。页面数据加载、保存、删除和错误状态统一放在 `src/features/workspace/useAdminWorkspace.ts`，再通过 props 传给页面。

新增后端接口时，在 `backend/models/` 定义请求/响应模型，在 `backend/handlers/<domain>.go` 编写 handler，在 `backend/repository/<domain>.go` 编写持久化逻辑，在 `backend/routes/<domain>.go` 注册路由并绑定认证、菜单和动作权限。前端在 `src/services/` 增加请求函数并复用 `requestWithSession`，同步更新类型、权限和 `README.md`。后端至少执行 `go test ./...`、`go vet ./...`，前端执行严格类型检查和生产构建。

## 适用范围

本文件适用于 `frontend/` 及其所有子目录，并补充仓库根目录的 `AGENTS.md`。

当前目录是已运行的 B 端管理前端，不是规划中的 `portal/` C 端。它同时承载后台工作台、内部聊天真实路由和 Socket 客服访客/管理入口，但三者必须保持各自状态与鉴权边界。

## 任务编号

当前 `Pn` 以最近一次已经创建的 Git commit 摘要为准。工作区尚未提交的连续前端改动必须继续使用同一个 `Pn`，不能因为补充需求自动递增；只有前一个 `Pn` 已成功提交后，下一项独立任务才使用下一个编号。提交前不得把未提交改动标记为新的已完成 `P`。

## 目录职责

本目录是采集数据管理平台的浏览器端管理后台，使用 Next.js 16 App Router、React 和 strict TypeScript，界面文案默认使用简体中文。

- `app/layout.tsx`：全局布局、metadata、Ant Design registry 和全局样式入口。
- `app/page.tsx`：管理后台客户端编排入口；未登录显示 `src/admin-pages/auth/AuthPage.tsx`，登录后按 `activePage` 挂载管理页面。
- `app/chat/`：内部员工聊天真实路由；`app/socket/chat/`：Socket 客服访客聊天真实路由与兼容入口。
- `src/admin-pages/`：认证、预览台、业务资源、用户、部门、角色、菜单、文章、文件、资料和访问分析页面。
- `src/features/workspace/`：会话恢复、菜单驱动加载、CRUD、筛选和反馈等工作台状态；`src/features/chat/`：Socket 客服管理端与访客端业务。
- `src/components/layout/MainLayout.tsx`：导航、响应式侧栏、主题、全局服务器终端入口和终端弹窗宿主。
- `src/components/shared/` 与 `src/components/table/`：共享反馈、富文本、关联用户、3D 卡片和表格组件。
- `src/services/`：认证业务请求、文件、访问分析和服务器/SSH API；`src/types/`：共享业务类型。
- `src/config/`、`src/utils/`、`src/theme/`、`src/styles/`：常量、权限/导出/菜单工具、主题和全局增强样式。
- `components/ui/`、根 `hooks/` 和根 `lib/`：shadcn/ui 基础设施；不得把管理业务重新迁回这些兼容目录。

## 页面与状态模型

- App Router 只有根页面 `/`；`app/page.tsx` 根据 `activePage` 条件渲染各业务页，不要假设 `/users`、`/files` 等独立 URL 已存在。
- `app/chat/page.tsx` 是内部聊天 `/chat`；`src/features/chat/CustomerChatPage.tsx` 是客服聊天，二者是独立业务边界。
- `/chat` 的表情、输入框、附件和链接图片预览属于内部聊天及 `/api/internal-chat/*`；客服聊天可独立修改其 `CustomerChatPage.tsx`、客服 socket 和对应接口，但两套实现不得混用业务状态或鉴权规则。
- 内部聊天附件必须通过会话参与者鉴权的接口下载或预览，禁止使用任意公开静态地址暴露物理文件。
- 处理内部聊天附件时，必须同时修改并验证前端发送与展示、后端持久化、下载与预览鉴权以及相关测试。
- 内部聊天应连接认证 WebSocket `/api/internal-chat/socket`：首页内部聊天入口显示未读总数，聊天页按会话显示未读角标，并在收到新消息时提供视觉提示和可用的声音提示；登录成功后立即发送在线状态。
- 消息列表的自动滚动必须尊重用户位置：历史滚动时禁止回弹，只有初次加载、切换会话或用户位于底部时才跟随新消息。
- 业务状态管理使用 React hooks，核心集中在 `src/features/workspace/useAdminWorkspace.ts`，当前没有 Redux、Zustand、MobX 或 React Query。
- `useAdminWorkspace` 负责会话恢复、按有效菜单加载资源、CRUD、筛选、表单、权限保存、加载状态和全局错误；页面组件通过 props 接收状态与动作。
- 当前页面写入 `sessionStorage` 的 `collector:active-page`，刷新后恢复；退出登录会清理。侧栏父菜单展开根据异步菜单树与 `activePage` 推导，修改导航时必须验证刷新恢复与父菜单展开。
- 主题使用 `localStorage` 持久化，并通过 `ADMIN_THEME_BOOTSTRAP_SCRIPT` 在首屏应用，避免水合前闪烁。新增主题时同步主题定义、CSS 变量和 Ant Design token。
- 后端 HttpOnly Cookie 是唯一认证凭据；前端不得把密码、会话 ID 或验证码持久化到 Web Storage。

## 实现约定

- 现有界面同时使用 Ant Design、Lucide、shadcn/ui、Tailwind CSS 4 和原生 CSS；修改时优先沿用相邻代码的组件和样式体系，避免为单个功能再引入一套 UI 依赖。
- 仅在需要浏览器 API、状态或事件时使用 Client Component。业务状态和 API 流程优先集中在 `useAdminWorkspace`，纯展示逻辑保留在功能组件。
- 共享业务类型、常量和请求逻辑分别放入 `src/types/`、`src/config/` 与 `src/services/`，不要在多个页面重复定义 API 结构。
- 后端地址取自 `NEXT_PUBLIC_API_BASE_URL`，默认 `http://localhost:8080`。
- 需要会话的请求复用 `requestWithSession`：固定 `credentials: 'include'`，单次超时 12 秒，仅 GET/HEAD/OPTIONS 在网络错误时按 350ms、900ms 重试；写请求禁止自动重试，以免产生重复副作用。
- API JSON 字段保持 camelCase。接口变更时同步检查后端模型、路由、权限以及 `README.md`。
- 用户有效菜单是所属部门、所属角色和个人附加菜单的并集；部门权限在部门页维护，角色权限在角色页维护，个人额外权限保留在用户页维护。
- 工作台图表使用 ECharts，数字动画使用 react-countup；图表颜色必须跟随当前主题变量。
- 保持桌面端和移动端布局可用，新增交互时检查加载、空数据、错误、禁用和提交中状态。
- 访问分析页面（`src/admin-pages/visitor-analytics/`）默认请求并展示最新 10 条记录，分页仅允许 `10`、`20`、`30`、`50`、`100`；筛选行的 Select、输入框、查询按钮和刷新按钮必须统一实际高度、垂直居中并保持独立间距，移动端自动换行或单列且无横向溢出。
- 访问分析页面的记录顺序必须是最新优先；统计数字采用平滑跳动动画，统计/图表/明细卡保持统一 hover 3D 效果，访问者列内容居中，隐私与保留策略作为固定说明展示。

## 权限与导航约定

- `src/types/admin.ts` 中的 `PageKey`、`src/config/constants.ts` 中的 `pageKeys/pageTitles`、`MainLayout.resolvePageKey` 与 `app/page.tsx` 的条件渲染必须保持一致。
- 菜单节点只有在 `code` 与去除斜杠后的 `path` 都匹配受支持页面时才映射为 `PageKey`；父级分组可以没有页面路径。
- 动作编码后端权威源是 `backend/permissions/actions.go`，前端镜像位于 `src/utils/actionPermissions.ts`。新增或重命名动作时两端与按钮显隐要同步。
- `super-admin`、`system-admin` 的判断使用 `src/utils/roleAccess.ts` 的稳定编码。不要用“超级管理员”等显示文字或用户名做安全判断。
- 所有超级管理员彼此同权，可以互相修改资料和登录权限；系统管理员及其他角色不能修改超级管理员。前端提示和按钮状态必须与后端真实结果一致，不得对初始化用户名 `MH` 设置例外。
- 页面不可只靠隐藏按钮保护操作；所有写请求必须接受并正确呈现后端 401/403/4xx 响应。

## 样式、组件与动画

- `app/globals.css` 是主样式入口，同时导入 Tailwind、shadcn 和 `article-file-enhancements.css`。修改全局选择器前先搜索同名规则和后置覆盖，避免因层叠顺序使布局失效。
- 相邻业务页主要使用 Ant Design 和原生 CSS，`components/ui/` 是 shadcn 基础组件；不要为了单个页面混入第三套新组件库。
- Ant Design 响应式布局优先使用真实 `Row/Col`，避免用宽泛的子元素 `display` 规则覆盖其 flex/grid 行为。
- 全站卡片效果由根布局中的 `TiltCardEffects` 通过事件委托和单个 `requestAnimationFrame` 自动增强；显式新卡可使用 `TiltCard/TiltCardLayer`。通过 `data-tilt-disabled="true"` 排除表单或交互密集容器。
- 3D 卡片只更新 CSS 变量和 transform；保持触摸设备与 `prefers-reduced-motion` 禁用逻辑。文件管理外层 `.file-browser-panel` 必须保持普通静态 Card，内部 `.file-card` 才启用效果。
- 弹窗和折叠权限区域要检查文字截断、Tooltip、左右留白以及 390px 移动端无横向溢出。
- Header、Dialog、Segmented 等同组工具按钮统一使用稳定高度和中心线；图标与文字必须处于同一个横向 flex 布局，`.ant-space-item`、`.ant-badge`、`.ant-btn-icon`、`.ant-segmented-item-icon` 与 SVG 使用紧凑行高。混用 Ant Design 与 Lucide 时需量测按钮、图标和文字中心线，禁止仅对某个图标添加 `top`、margin 或 `translateY` 补偿。
- 所有新增或修改的折叠面板、侧栏（sidebar）及展开/收起交互都必须为宽度、位移或内容显隐提供平滑过渡，禁止无动画瞬间跳变；动画不得造成遮挡或横向溢出，并必须在 `prefers-reduced-motion: reduce` 下关闭或显著弱化。
- 内部聊天 `/chat` 与客服聊天页面的主内容区应优先填满可用宽度，避免桌面端两侧产生不必要的大留白；调整一侧页面时保持另一侧的 API、状态和鉴权边界独立即可。
- 使用 Ant Design Modal 时遵循当前版本 API，禁止新增已弃用的 `maskClosable`，应使用 `mask={{ closable: ... }}`。

## 文章与文件能力

- 后端 `/api/articles/export` 是文章集合 CSV/PDF；前端 `articleExport.ts` 是单篇文章 CSV、打印/PDF、Word、PNG、Markdown、SEO HTML，两者不是同一实现。
- Markdown 目录与锚点逻辑位于 `articleMarkdown.ts`：正文有标题时生成目录、显式锚点和重复标题唯一后缀；无标题时不生成目录。
- 文章导出会处理跨源媒体、图片内联、分页画布和可见内容检查；修改时必须实际导出至少一个含标题和图片的样本验证文件内容。
- 前端和后端上传限制均为 32 MiB。文件读取、元数据、文本内容和永久删除复用 `src/services/fileApi.ts`，业务编排仍位于 `useAdminWorkspace` 与 `FilesPage`。
- 文件删除默认移入回收站；永久删除按钮和请求只能在用户明确确认后触发，测试不能用正式业务 API 清理样本。
- 文件管理页面必须保留刷新按钮和“设为登录背景”操作；聊天数据等新增分类只能扩展现有文件管理能力，不得覆盖或隐藏这些操作。

## 服务器监控与 SSH 工作区

- 预览台通过 `src/services/serverApi.ts` 每 5 秒读取 `/api/server/metrics`，维护最近 5 分钟的网络与磁盘吞吐趋势；停止访问或组件卸载时必须清理采样计时器。
- 工作台下的 `business-resources` 独立页面展示用户、菜单和文章的总量、有效量、构成与可用率；这些业务数据不得重新放回预览台长页面底部。
- 预览台“活动连接”指标可点击，并通过 `/api/server/connections` 按需打开连接明细；明细页需保留状态筛选、当前结果搜索、刷新以及平台不支持时的结构化警告。
- 监控值代表后端进程所在主机或容器可见资源。缺失温度、磁盘 I/O 等平台数据时展示 `collectionWarnings`，不能伪造为零或宿主机完整指标。
- SSH 是 Header 全局功能，任意有效登录用户都可打开，不依赖当前管理页面或管理员角色。弹窗关闭行为是最小化：切换页面或再次打开时连接、多终端标签、目录和未关闭预览继续保留。
- 只有 `roleCode=super-admin` 的用户可看到“部署机直连”选项，并连接 `/api/server/host-terminal`；系统管理员和普通用户仍只使用现有 SSH。前端隐藏只是体验边界，后端 403 才是安全边界。
- 部署机直连必须显示代理上报的 `系统账号@主机名`，代理离线时显示明确错误；不得退回后端容器 shell，也不得把代理共享令牌发送给浏览器。
- 只有退出登录、会话失效或浏览器页面卸载时才销毁全部 SSH 连接和敏感状态；密码、私钥、编辑内容不得写入 `localStorage`、`sessionStorage` 或日志。
- `SshTerminalModal.tsx` 管理多会话与弹窗生命周期，`SshTerminalSession.tsx` 管理单连接终端、SFTP 步进目录、当前目录搜索、文件编辑/保存和图片/PDF预览。终端 OSC 7 路径变化必须同步刷新左侧当前目录。
- 主机地址使用普通可编辑 Input，默认值为 `lolicon.beer`；用户名默认 `root`。文件保存成功必须同时显示 message、清除待保存状态并保留明确的已保存反馈。
- 使用 Ant Design 对话框确认时必须消费 `App` 上下文提供的 modal/message 实例，禁止调用会触发动态主题警告的 `Modal.confirm`、`message.success` 等静态函数。

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
