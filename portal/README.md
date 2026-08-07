# C 端内容门户（portal）

面向普通访问者的公开内容门户，展示经 B 端明确发布到门户的文章、图片与文件资源，支持简体中文 / English / 日本語、三套主题、移动端适配与 SEO。

## 技术栈
- Next.js 16 App Router + React 19 + strict TypeScript
- Tailwind CSS 4 + Lucide React
- next-intl 4 管理多语言路由与消息资源
- 开发端口固定为 `3001`，避免与 B 端 `3000` 冲突

## 环境变量
复制 `.env.example` 为 `.env`，并按部署环境调整取值。

- `NEXT_PUBLIC_API_BASE_URL`：后端 API 基础地址，默认 `http://localhost:8080`
- `NEXT_PUBLIC_SITE_URL`：门户站点地址，默认 `http://localhost:3001`，生产环境必须使用 HTTPS
- `NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT`：是否启用客服聊天入口，默认 `false`
- `PORTAL_REVALIDATE_SECONDS`：公开数据缓存秒数，默认 `60`

## 运行命令
```bash
npm install
npm run dev     # 开发模式：http://localhost:3001
npm run build   # 生产构建
npm run start   # 生产预览：http://localhost:3001
npm run typecheck  # 严格类型检查
npm run lint       # ESLint
npm run docs       # TypeDoc
```

## 数据边界
- 页面只通过后端 `/api/public/*` 只读接口获取数据，不携带后台登录 Cookie，也不提供任何写能力。
- 文章只有状态为 `已发布` 且 `portalVisible` 开启、非 `isPrivate` 时才会展示；文件同理需满足 `portalVisible` 开启、非私密且未删除。
- 门户不直连 SQLite，所有内容均以公开 API 为准，避免绕过发布权限。
- 图片与资源预览、下载统一走后端公开媒体接口，不暴露物理存储路径。

## 隔离环境
- 联调与验收使用 `.workspace-temp/p<n>-portal/` 下的独立 SQLite 数据库与上传目录。
- 禁止修改、重置或清空 `backend/data/` 与 `backend/uploads/` 中的业务数据。

## SEO
- 首页、文章详情、分类页、图片页均生成 canonical 与 hreflang 多语言标注。
- 文章详情注入 `Article` JSON-LD，首页注入 `WebSite` JSON-LD。
- 提供 `/sitemap.xml`、`/robots.txt`、`/{locale}/feed.xml` 订阅源。
- 搜索结果页声明 `noindex,follow`，避免低价值页面被收录。