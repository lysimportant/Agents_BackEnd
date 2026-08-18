# C 端内容门户当前实现说明

本文以当前代码为准，描述 `portal/` 已实现能力、后端公开条件和后续修改边界。它不是未实施的规划书。

## 一、当前状态

`portal/` 是可独立安装、开发、构建和容器部署的 Next.js 16 应用，默认地址为 `http://localhost:3001`。仓库根 `docker-compose.yml` 已同时编排 Redis、后端、B 端和 C 端。

当前技术栈：

- Next.js 16 App Router、React、strict TypeScript。
- Tailwind CSS 4、Lucide React、`next-intl`。
- `sanitize-html` 与 Cheerio 用于公开文章内容处理。
- TypeDoc 输出到 `portal/docs/`。

`backend/` 是唯一业务数据来源。C 端不得直连 SQLite、扫描上传目录、复制后台数据或建立第二套内容存储。

## 二、应用边界

C 端内容请求只能调用 `/api/public/*`。为实现登录后 18R 可见性，允许调用以下认证接口：

- `POST /api/auth/login`
- `GET /api/auth/session`
- `POST /api/auth/logout`

C 端不得调用后台文章、文件、用户、权限或终端写接口。后端 HttpOnly Cookie 是唯一会话凭据；浏览器不得保存会话 ID、密码或访问令牌。

客服开关 `NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT` 已保留，但当前核心门户页面不依赖客服组件。后续启用时只能复用 Socket 客服边界，不得调用 `/api/internal-chat/*`。

## 三、公开条件与 18R

### 文章

公开文章的数据库条件：

```text
is_private = 0 AND status = '已发布'
```

匿名请求还会追加 `is_18r=0`。文章不存在、私密、未发布或被 18R 过滤时，详情接口统一按不可公开处理。

### 文件

公开文件的数据库条件：

```text
is_private = 0 AND deleted_at IS NULL
```

图片由 `content_type LIKE 'image/%'` 区分，其他文件进入资源列表。匿名请求同样追加 `is_18r=0`。

### 18R 会话规则

后端只有在以下两个条件同时成立时才包含 18R 内容：

1. 请求携带有效后端登录会话 Cookie。
2. 请求携带 `portal-r18=1` Cookie。

仅设置 18R Cookie、仅登录或匿名访问都不会返回 18R 内容。退出登录时 C 端同步关闭本地 18R 偏好。

### 当前没有门户发布开关

数据库和模型中没有独立的门户可见或门户精选字段。`articles.portal_published_at` 是保留的时间元数据，不是公开开关；文件公开时间当前使用 `updated_at`。

因此 B 端修改文章状态、私密状态、文件私密状态或回收站状态，会直接改变 C 端可见性。

## 四、路由

规范语言为 `zh-CN`、`en-US`、`ja-JP`。根路径 `/` 根据合法语言 Cookie 或浏览器语言跳转；没有合法偏好时回退 `zh-CN`。

已实现路由：

| 路由 | 当前行为 |
| --- | --- |
| `/` | 解析语言并跳转到 `/{locale}`。 |
| `/{locale}` | 当前跳转到 `/{locale}/images`，图片瀑布流是门户第一入口。 |
| `/{locale}/images` | 图片筛选、自动加载、渐进图片和预览。 |
| `/{locale}/articles` | 文章筛选和 URL 分页。 |
| `/{locale}/articles/{id}/{slug}` | 文章详情、清洗正文、目录、推荐和 SEO。 |
| `/{locale}/resources` | 非图片资源筛选和 URL 分页。 |
| `/{locale}/categories` | 公开分类聚合。 |
| `/{locale}/categories/{category}` | 分类下文章与图片。 |
| `/{locale}/search` | 跨文章、图片和资源搜索，页面 noindex。 |
| `/{locale}/about` | 关于页面。 |
| `/{locale}/feed.xml` | 当前语言文章 RSS。 |
| `/sitemap.xml` | 静态页面、文章和分类 sitemap。 |
| `/robots.txt` | 允许抓取内容页并排除搜索页。 |

不支持的语言段返回 404。C 端使用真实 URL 路由，不沿用 B 端 `activePage` 模型。

## 五、公开 API

当前公开路由：

```text
GET /api/public/articles
GET /api/public/articles/:id
GET /api/public/images
GET /api/public/resources
GET /api/public/files/:id/preview
GET /api/public/files/:id/thumbnail
GET /api/public/files/:id/medium
GET /api/public/files/:id/download
GET /api/public/categories
GET /api/public/site-summary
GET /api/public/search
```

列表统一返回：

```json
{
  "items": [],
  "pagination": {
    "page": 1,
    "pageSize": 24,
    "total": 0,
    "totalPages": 0
  }
}
```

- 默认 `page=1`、`pageSize=24`，最大 `pageSize=50`。
- 非法页码或分页大小回退默认值。
- `keyword` 和 `category` 用于筛选。
- `sort` 当前会解析但不会影响数据库排序；列表固定按 ID 倒序。
- 聚合搜索要求 `keyword`，超过 40 个字符时截断，每组最多返回 12 条。
- 公开响应不返回 `storageName`、物理路径、所有者 ID、后台权限和删除字段。

文章详情正文由后端白名单清洗，并把符合公开条件的本地媒体改写为 `/api/public/files/...`。相关文章最多 6 篇；封面从清洗后正文第一张图片派生；`tableOfContents` 后端字段当前为空，C 端可从正文生成展示目录。

## 六、图片加载与瀑布流

公开图片提供三种读取方式：

| 端点 | 用途 | 规格 |
| --- | --- | --- |
| `thumbnail` | 低流量占位和预加载 | 最大 `480x480`，JPEG 质量 80 |
| `medium` | 瀑布流主体 | 最大宽度 `1280`，不限制长图高度，JPEG 质量 85 |
| `preview` | 大图预览和原图读取 | 返回原文件 |

图片页不显示分页按钮，但不会一次取回全部图片。实现规则：

- 后端分页仍是内部批次协议，用于限制单次响应和恢复失败批次。
- `IntersectionObserver` 在用户接近底部前预加载下一批。
- 底部显示加载状态；失败时保留重试入口。
- 已加载卡片按稳定的独立列分配，追加下一批时不得让旧卡片跨列重排或短暂消失。
- 根据 `imageWidth/imageHeight` 预留比例空间，缩略图、中图、原图逐级加载。
- 瀑布流共享队列限制中图并发：移动端最多 2 张，较宽视口最多 4 张；离开预加载范围时会取消尚未完成的中图请求。
- 当前预览浮层一次只展示并请求当前原图。未来漫画阅读应创建独立队列，在视口附近逐张加载原图并预加载下一张，禁止无界并发。
- 新卡片进入视口时使用轻量透明度和位移动画；`prefers-reduced-motion` 下关闭或弱化。

这些能力可复用于未来的纵向漫画阅读：列表先加载低清或中图，只在视口附近请求原图，并预加载下一张。

## 七、布局、主题和动画

- Header 与页面主内容宽度统一为视口的 `4/5`，页面内容不得超过 Header 宽度。
- C 端页脚已移除，图片瀑布流可以占用剩余页面空间。
- 主题支持 `light`、`dark`、`ocean` 和 `system`，持久化键为 `portal-theme`。
- 主题在首屏脚本中应用，避免水合前闪烁。
- 公共卡片和新内容进入视口时使用统一 reveal 动画；触摸设备和减少动态偏好下保持可用。
- 图片预览支持键盘关闭、前后切换、缩放和移动端交互。
- 页面必须保持无横向溢出、长文案换行、稳定控件尺寸和可见焦点。

## 八、国际化

UI 支持：

- `zh-CN`：默认语言、源文案和最终回退语言。
- `en-US`
- `ja-JP`

URL 语言段是页面语言的唯一事实来源。`portal-locale` Cookie 只保存通过白名单验证的语言代码。用户生成的标题、正文、分类和说明保持原文，不自动翻译。

新增或修改用户可见文案时必须同时更新三份消息文件，并检查英文、日文长文本在桌面和移动端不会挤压控件。

## 九、SEO 与内容安全

- 页面 metadata、canonical、Open Graph、JSON-LD、sitemap、robots 和 RSS 使用 `NEXT_PUBLIC_SITE_URL` 生成绝对地址。
- 生产环境必须设置最终 HTTPS 站点地址，不能使用 localhost 生成 canonical。
- 搜索页使用 `noindex`；文章详情以稳定 ID 查询并使用 slug 提升可读性。
- 原始文章 HTML 不直接渲染，危险标签、事件属性和危险协议由清洗层移除。
- 本地媒体必须经过公开接口再次校验，不拼接上传目录。
- 公开文件预览和下载以数据库记录解析存储名，不接受任意磁盘路径参数。
- 测试、构建和浏览器验收不得删除或重置业务数据库和上传目录。

## 十、环境变量与部署

`portal/.env.example`：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | 浏览器访问后端的公开地址 |
| `BACKEND_INTERNAL_URL` | `http://localhost:8080` | 服务端渲染访问后端的内部地址 |
| `NEXT_PUBLIC_SITE_URL` | `http://localhost:3001` | canonical、sitemap、RSS 等站点绝对地址 |
| `NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT` | `false` | 保留的客服入口开关 |
| `PORTAL_REVALIDATE_SECONDS` | `60` | 服务端内容缓存秒数 |

Compose 中门户默认映射 `3001`，生产配置由根 `.env` 的 `PORTAL_*` 变量覆盖。后端 CORS 白名单应同时包含 B 端和 C 端完整 Origin。

## 十一、验证

C 端：

```powershell
cd portal
npm run typecheck
npm run lint
npm run docs
npm run build
```

公开 API 改动还需执行：

```powershell
cd backend
go test ./...
go vet ./...
```

涉及 B 端文章或文件字段时执行 B 端严格类型检查和生产构建。视觉或交互改动使用官方 Browser 能力检查桌面、平板、移动端、三语言、主题、控制台、横向溢出和减少动态效果。

## 十二、当前已知限制

- `sort` 参数当前不改变查询排序。
- `/api/public/*` 没有独立速率限制器；当前只有分页上限和搜索关键词长度限制。
- `FeaturedImages` 字段名称保留兼容性，但实际返回最新 8 张公开图片，不存在精选字段。
- `portal_published_at` 不是公开开关，文件的 `publishedAt` 当前等于 `updatedAt`。
- `site-summary` 的文章总数当前按全部非私密已发布文章统计，未按匿名 18R 条件缩减；文章列表本身仍会正确过滤。
- 公开原文件预览和下载当前没有统一的文件类型白名单，安全边界主要依赖公开状态、数据库存储名解析和浏览器响应行为。
- 客服环境开关存在，但当前没有作为门户核心页面的必备功能。

修改这些限制时必须同步后端、C 端调用方、OpenAPI、本文和相关测试。
