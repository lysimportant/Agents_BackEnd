# SQLite 数据字典

权威 DDL 位于 `backend/repository/sqlite_store.go`。本文说明各表的归属、用途和敏感数据边界；迁移变更必须同时更新代码注释和本文。

| 数据表 | 用途 | 重要关联与访问规则 |
| --- | --- | --- |
| `data_points` | 工作台来源/指标采样数据。 | 读写由工作台菜单和动作权限控制。 |
| `departments` | 层级组织部门。 | `parent_id` 引用 `departments.id`。 |
| `roles` | 稳定角色编码和可编辑展示信息。 | 安全判断使用 `code`，不得使用展示名称 `name`。 |
| `role_migrations` | 角色权限种子和迁移版本记录。 | 以迁移键保证幂等执行，仅由服务端启动迁移流程写入。 |
| `users` | 登录账户和员工资料。 | `role_id`、`department_id` 引用对应所属表；`password_hash` 永不序列化。 |
| `menus` | 导航节点和菜单鉴权数据。 | `parent_id` 构成菜单树。 |
| `user_menus` | 用户个人附加菜单权限。 | 用户与菜单的复合关联。 |
| `department_menus` | 部门继承的菜单权限。 | 部门与菜单的复合关联。 |
| `role_menus` | 角色继承的菜单权限。 | 角色与菜单的复合关联。 |
| `user_action_permissions` | 用户个人附加的稳定 `resource.action` 权限。 | 有效动作是角色默认动作与个人附加动作的并集。 |
| `sessions` | 不透明 HttpOnly 会话 ID 和过期时间。 | 前端存储中不暴露访问令牌。 |
| `articles` | 知识库文章。 | `owner_id`、`is_private` 决定可见性和修改边界。 |
| `files` | 文件管理上传和软删除元数据。 | `owner_id`、`is_private`、`deleted_at` 控制访问和回收站状态。 |
| `socket_conversations` | 客服聊天会话摘要。 | 与内部员工聊天严格区分。 |
| `socket_messages` | 客服消息和附件元数据。 | 附件通过受保护的客服聊天接口读取。 |
| `internal_chat_messages` | 内部员工私聊和群发消息。 | `recipient_id` 为空时表示群发消息。 |
| `internal_chat_attachments` | 内部聊天上传文件。 | `owner_id` 拥有待发送附件，`message_id` 绑定已发送消息；仅参与者或管理员可读取。 |
| `visitor_access_logs` | 保留的 IP、User-Agent、路径、耗时和代理地理信息。 | 仅系统管理员和超级管理员可见，并按配置的保留天数清理。 |

所有时间戳都使用 repository 统一的文本格式保存。物理存储字段 `storage_name`、`stored_name` 仅供服务端使用，不得作为公开 URL 返回。
