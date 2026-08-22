# SQLite 数据字典

权威 DDL 和增量迁移位于 `backend/repository/sqlite_store.go`。本文记录表用途、关键字段、索引和访问边界；禁止通过删除数据库处理迁移问题。

## 业务表

| 数据表 | 用途 | 重要关联与访问规则 |
| --- | --- | --- |
| `data_points` | 工作台来源/指标采样数据。 | 读写由工作台菜单和动作权限控制。 |
| `departments` | 层级组织部门。 | `parent_id` 引用 `departments.id`，禁止循环。 |
| `roles` | 稳定角色编码和可编辑展示信息。 | 安全判断使用 `code`，不得使用展示名称。 |
| `role_migrations` | 角色权限种子和迁移版本记录。 | 由迁移逻辑创建和写入，以迁移键保证幂等。 |
| `users` | 登录账户和员工资料。 | `role_id`、`department_id` 引用所属表；`password_hash` 永不序列化。 |
| `menus` | 导航节点和菜单鉴权数据。 | `parent_id` 构成菜单树。 |
| `user_menus` | 用户个人附加菜单权限。 | 用户与菜单的复合关联。 |
| `department_menus` | 部门继承的菜单权限。 | 部门与菜单的复合关联。 |
| `role_menus` | 角色继承的菜单权限。 | 角色与菜单的复合关联。 |
| `user_action_permissions` | 用户个人附加动作权限。 | 编码格式为稳定的 `resource.action`。 |
| `sessions` | 不透明 HttpOnly 会话 ID 和过期时间。 | 前端不得持久化会话 ID。 |
| `articles` | 知识库文章和 C 端文章来源。 | 所有者、私密状态、发布状态和 18R 分级共同决定访问。 |
| `files` | 文件管理上传、回收站和 C 端媒体来源。 | 所有者、私密状态、软删除和 18R 分级共同决定访问；管理上传不设置应用层单文件大小上限。 |
| `public_file_likes` | 登录用户对公开图片的点赞关系。 | `(file_id,user_id)` 复合主键保证每个账号每张图片最多一个点赞。 |
| `public_file_comments` | 登录用户发送的公开图片纯文本评论。 | 关联文件和用户；文件或用户删除时级联清理对应互动记录。 |
| `socket_conversations` | Socket 客服会话摘要。 | 与内部员工聊天严格区分。 |
| `socket_messages` | Socket 客服消息和附件元数据。 | 附件通过客服鉴权接口读取。 |
| `internal_chat_messages` | 内部员工私聊和群发消息。 | `recipient_id` 为空表示群发。 |
| `internal_chat_attachments` | 内部聊天上传文件。 | 仅消息参与者或管理员可读取。 |
| `visitor_access_logs` | IP、User-Agent、路径、耗时和可信代理地理信息。 | 仅管理员可见，并按配置的保留天数清理。 |

## 公开内容字段

文章公开条件是 `is_private=0 AND status='已发布'`。匿名请求还要求 `is_18r=0`；有效后台会话同时携带 `portal-r18=1` Cookie 时可包含 18R 文章。

- `articles.portal_published_at`：保留的发布时间元数据；为空时公开响应回退到 `updated_at`。它不是公开开关。
- `articles.content_locale`：正文实际语言，默认 `zh-CN`。
- `articles.is_18r`：18R 分级标记，默认 `0`。

文件公开条件是 `is_private=0 AND deleted_at IS NULL`。匿名请求还要求 `is_18r=0`；有效会话和 18R Cookie 同时存在时可包含 18R 文件。

- `files.content_sha256`：服务端按“同一所有者 + 有效文件”过滤重复内容，不向客户端返回。
- `files.tags`：JSON 数组形式的文件标签，最多 12 个、每个最多 24 个 Unicode 字符；用于 B 端管理、C 端展示和关键词搜索。
- 标签写入前去首尾空白、移除前导 `#`，再按不区分大小写去重；空标签丢弃，超过数量或长度时由请求校验拒绝。
- `files.image_width`、`files.image_height`：图片原始尺寸，用于预留瀑布流空间和减少布局跳动。
- `files.is_18r`：18R 分级标记，默认 `0`。
- `files.deleted_at`：非空表示进入回收站；永久删除才移除记录和物理文件。

当前数据库没有独立的门户可见或门户精选字段。C 端是否可见完全由上述公开条件判断。

## 关键索引

- `idx_articles_public(is_private,status,id)`：公开文章筛选和倒序列表。
- `idx_files_public(is_private,deleted_at,id)`：公开文件筛选和倒序列表。
- `idx_public_file_likes_file_id(file_id,user_id)`：按图片统计点赞和读取当前用户状态。
- `idx_public_file_comments_file_id(file_id,id)`：按图片读取最近评论。
- `idx_files_owner_content_sha256(owner_id,content_sha256)`：只覆盖未删除且哈希非空的文件，保证同一所有者有效内容唯一。
- `.thumbnail-cache` 不属于 SQLite 表；它是由受保护缩略图接口按需生成的可再生派生缓存，丢失时可重新生成，不能替代 `files` 记录或上传原文件。
- 聊天、访问日志、用户部门角色等索引以 `sqlite_store.go` 为权威来源，变更时同步本文。

## 互动与删除语义

`public_file_likes` 对 `(file_id,user_id)` 使用复合主键，点赞接口因此是幂等切换；`public_file_comments.content` 保存去首尾空白的纯文本，最长 500 个 Unicode 字符，读取按文件 ID 返回最新 100 条。文件永久删除或用户删除时，外键级联清理互动记录；软删除期间原记录和互动数据仍可恢复。

所有时间戳使用 repository 统一文本格式保存。`storage_name`、`stored_name`、密码哈希、会话 ID 和物理路径仅供服务端使用，不得作为公开 URL 或公开响应字段返回。
