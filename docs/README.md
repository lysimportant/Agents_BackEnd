# 生成式文档说明

第一次启动和使用平台，请先阅读[平台操作说明](./operations-guide.md)。

新增业务页面和后端创建接口，请阅读[前后端页面/API 开发说明](./frontend-backend-page-api-guide.md)。

本仓库按技术栈使用各自原生的文档格式：

- 后端代码：使用中文 GoDoc 注释；在 `backend/` 执行 `Get-ChildItem -Directory | Where-Object { Test-Path "$($_.FullName)\doc.go" } | ForEach-Object { go doc ".\$($_.Name)" }` 检查。
- HTTP API：使用 `docs/api/openapi.yaml` 中的 OpenAPI 3.1 契约。
- 前端代码：使用中文 JSDoc 注释，并由 TypeDoc 按 `frontend/typedoc.json` 生成文档。
- 数据库：使用 `docs/database/schema.md` 记录表用途和权限边界，同时在迁移 SQL 旁保留对应中文注释。

安装前端依赖后生成前端参考文档：

```powershell
cd frontend
npm run docs
```

生成的 HTML 位于 `.workspace-temp/p29-docs/frontend`，不纳入 Git 提交。

业务源码中的注释统一使用简体中文。文档与解释性注释覆盖类型、函数、方法、组件、服务、模块级常量、共享状态、页面内部函数、回调和局部变量，并说明公共契约、领域规则、变量保存内容和不直观的副作用；页面或文件外层已有注释不能代替内部声明的注释。
