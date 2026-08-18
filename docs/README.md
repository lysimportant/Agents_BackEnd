# 项目文档索引

根 `docs/` 保存人工维护的业务、部署和接口说明；`portal/docs/` 是 TypeDoc 生成结果，不应手工编辑。

## 手写文档

- [平台操作说明](./operations-guide.md)：本地启动、登录、B 端操作、C 端访问和常见问题。
- [Linux 生产部署手册](./deployment-guide.md)：Docker Compose、Nginx、HTTPS、备份和 root 宿主机代理。
- [前后端页面/API 开发说明](./frontend-backend-page-api-guide.md)：B 端页面、后端接口和 C 端公开功能的开发流程。
- [C 端门户当前实现说明](./portal-requirements.md)：门户边界、路由、公开条件、图片加载、SEO 和已知限制。
- [OpenAPI 3.1 契约](./api/openapi.yaml)：后端 HTTP 与 WebSocket 入口。
- [SQLite 数据字典](./database/schema.md)：表用途、关键字段、索引和数据保护边界。

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
