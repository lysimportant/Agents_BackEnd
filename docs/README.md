# 项目文档索引

根 `docs/` 保存人工维护的业务、部署和接口说明；`portal/docs/` 与 `.workspace-temp/p29-docs/frontend/` 是 TypeDoc 生成结果，不应手工编辑。API、数据库、部署和门户规则发生变化时，先更新这里列出的手写文档，再按需重新生成代码参考文档。

## 手写文档

- [平台操作说明](./operations-guide.md)：本地启动、登录、B 端多文件上传/去重/标签、C 端访问和常见问题。
- [Linux 生产部署手册](./deployment-guide.md)：Docker Compose、Nginx、HTTPS、备份和 root 宿主机代理。
- [前后端页面/API 开发说明](./frontend-backend-page-api-guide.md)：B 端页面、后端接口和 C 端公开功能的开发流程。
- [C 端门户当前实现说明](./portal-requirements.md)：门户边界、路由、公开条件、图片加载、SEO 和已知限制。
- [OpenAPI 3.1 契约](./api/openapi.yaml)：后端 HTTP 与 WebSocket 入口。
- [SQLite 数据字典](./database/schema.md)：表用途、关键字段、索引和数据保护边界。

## 同步约定

- OpenAPI 的路径、请求体和响应字段必须与后端 handler、前端 service 及本文档中的业务限制一致。
- `files` 的内容哈希、标签规范、缩略图缓存和软删除语义以数据库字典为准；聊天附件限制不继承文件管理的无限制上传规则。
- C 端列表的分页只属于内部批次协议，页面不显示分页按钮；自动加载、图片队列和预览互动以门户说明为准。
- 生产部署更新优先使用部署手册中的动态 `systemd-run` 单元，断线后通过 `journalctl` 和 `docker compose ps` 验证，不把前端终端是否保持连接当作部署结果。

## 代码参考文档

后端使用中文 GoDoc。在 `backend/` 检查包文档：

```powershell
Get-ChildItem -Directory | Where-Object { Test-Path "$($_.FullName)\doc.go" } | ForEach-Object { go doc ".\$($_.Name)" }
```

B 端使用中文 JSDoc/TypeDoc，配置位于 `frontend/typedoc.json`：

```powershell
cd frontend
npm run docs
```

输出目录为 `.workspace-temp/p29-docs/frontend`，不纳入 Git。

C 端使用中文 JSDoc/TypeDoc，配置位于 `portal/typedoc.json`：

```powershell
cd portal
npm run docs
```

输出目录为 `portal/docs/`。它是生成内容，源码契约变化后重新生成，不在其中手工维护业务规则。
