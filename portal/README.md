# 门户（C 端）

面向普通访问者的公开内容门户，通过后端 <code>/api/public/*</code> 展示经 B 端明确发布的文章、图片与文件资源，并支持登录后图片点赞和评论。

- 技术栈：Next.js 16 App Router + React + strict TypeScript + Tailwind CSS 4 + next-intl + Lucide React。
- 默认地址：<code>http://localhost:3001</code>，后端默认 <code>http://localhost:8080</code>。
- 支持语言：<code>zh-CN</code>（默认）、<code>en-US</code>、<code>ja-JP</code>。
- 主题：浅色、深色、海洋品牌、跟随系统。
- 边界：只调用 <code>/api/public/*</code>，不携带后台 Cookie，不提供任何写能力。

## 安装

    cd portal
    npm install

## 环境变量

复制 <code>.env.example</code> 为 <code>.env.local</code> 并按需调整：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| <code>NEXT_PUBLIC_API_BASE_URL</code> | <code>http://localhost:8080</code> | 浏览器可访问的后端地址 |
| <code>BACKEND_INTERNAL_URL</code> | <code>http://localhost:8080</code> | 服务端渲染访问后端的内部地址，Compose 中使用后端服务名 |
| <code>NEXT_PUBLIC_SITE_URL</code> | <code>http://localhost:3001</code> | C 端绝对站点地址，生产必须是最终 HTTPS 域名 |
| <code>NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT</code> | <code>false</code> | 是否启用客服入口 |
| <code>PORTAL_REVALIDATE_SECONDS</code> | <code>60</code> | 服务端内容缓存秒数，校验为合理正整数 |

<code>NEXT_PUBLIC_*</code> 中禁止放密钥、数据库地址、后台 Cookie 或其他服务端秘密。

## 常用脚本

    npm run dev        # 开发服务器，端口 3001
    npm run build      # 生产构建
    npm run start      # 生产启动，端口 3001
    npm run typecheck  # 严格类型检查
    npm run lint       # ESLint 检查
    npm run docs       # 生成 TypeDoc 文档到 docs/

## Docker 部署

仓库根 <code>docker-compose.yml</code> 已包含门户服务，可在仓库根目录执行：

    Copy-Item .env.example .env
    docker compose up --build

根 <code>.env</code> 中的 <code>PORTAL_PORT</code> 控制宿主机访问端口，<code>PORTAL_INTERNAL_PORT</code> 控制容器内监听端口，默认均为 <code>3001</code>。如修改 <code>PORTAL_PORT</code>，还需同步修改 <code>PORTAL_SITE_URL</code> 与 <code>CORS_ALLOWED_ORIGINS</code>；如修改后端对外端口，还需同步修改 <code>PORTAL_API_BASE_URL</code>。这些公开地址在镜像构建时写入客户端产物，修改后需要重新执行 <code>docker compose up --build</code>。

## 路由

| 路径 | 说明 |
| --- | --- |
| <code>/</code> | 根据已验证语言偏好跳转到对应语言首页 |
| <code>/{locale}</code> | 首页：图片瀑布流、数据概览、最新文章、热门分类 |
| <code>/{locale}/articles</code> | 文章列表 |
| <code>/{locale}/articles/[id]/[slug]</code> | 文章详情 |
| <code>/{locale}/images</code> | 图片瀑布流 |
| <code>/{locale}/resources</code> | 资源列表 |
| <code>/{locale}/categories</code> | 分类总览 |
| <code>/{locale}/categories/[category]</code> | 分类详情 |
| <code>/{locale}/search</code> | 搜索 |
| <code>/{locale}/about</code> | 关于 |
| <code>/{locale}/feed.xml</code> | RSS 订阅 |
| <code>/sitemap.xml</code>、<code>/robots.txt</code> | SEO 元数据路由 |

## 公开 API（后端提供）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | <code>/api/public/articles</code> | 公开文章列表 |
| GET | <code>/api/public/articles/:id</code> | 公开文章详情 |
| GET | <code>/api/public/images</code> | 公开图片列表 |
| GET | <code>/api/public/resources</code> | 公开资源列表 |
| GET | <code>/api/public/files/:id/preview</code> | 文件内联预览 |
| GET | <code>/api/public/files/:id/thumbnail</code> | 图片缩略图 |
| GET | <code>/api/public/files/:id/medium</code> | 图片屏幕适配中图 |
| GET | <code>/api/public/files/:id/download</code> | 文件下载 |
| GET | <code>/api/public/files/:id/interactions</code> | 图片点赞数量和评论 |
| POST | <code>/api/public/files/:id/like</code> | 登录用户点赞或取消点赞 |
| POST | <code>/api/public/files/:id/comments</code> | 登录用户发送纯文本评论 |
| GET | <code>/api/public/categories</code> | 公开分类聚合 |
| GET | <code>/api/public/site-summary</code> | 站点聚合概览 |
| GET | <code>/api/public/search?keyword=</code> | 聚合搜索 |

## 三应用联调

1. 启动后端：<code>cd backend && go run .</code>（默认 <code>http://localhost:8080</code>）。
2. 启动管理前端：<code>cd frontend && npm run dev</code>（默认 <code>http://localhost:3000</code>）。
3. 启动门户：<code>cd portal && npm run dev</code>（默认 <code>http://localhost:3001</code>）。

在 B 端文章/文件管理中开启“发布到门户”后，满足发布条件的内容才会出现在门户。

## 目录结构

<code>app/</code> 只承载 App Router 路由入口、布局、metadata 与全局样式；业务源码位于 <code>src/</code>，按 <code>components/</code>、<code>services/</code>、<code>types/</code>、<code>config/</code>、<code>i18n/</code>、<code>content/</code>、<code>theme/</code>、<code>utils/</code> 等职责拆分。详细约定见 <code>AGENTS.md</code>。

## 安全与缓存说明

- 正文在服务端按白名单二次清洗后才渲染，本地媒体重写为公开地址。
- 列表与详情使用 Next.js revalidation，默认 TTL 60 秒；取消发布后在缓存窗口内失效。
- 生产部署前需确认：最终 HTTPS <code>NEXT_PUBLIC_SITE_URL</code>、CORS 白名单、CSP、默认分享图与客服开关。
