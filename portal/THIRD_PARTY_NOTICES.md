# 第三方组件与许可证说明（THIRD_PARTY_NOTICES）

本文件记录 C 端门户 <code>portal/</code> 直接引入的开源依赖及其许可证、来源、使用范围与修改说明。完整的依赖闭包以 <code>package-lock.json</code> 解析结果为准，各依赖的完整许可证文本见其 npm 包内 LICENSE 文件。

## 直接运行时依赖

| 项目 | 许可证 | 来源 | 使用范围 |
| --- | --- | --- | --- |
| Next.js | MIT | https://github.com/vercel/next.js | App Router 框架、服务端渲染、路由与 metadata |
| React / React DOM | MIT | https://github.com/facebook/react | UI 组件与客户端交互 |
| next-intl | MIT | https://github.com/amannn/next-intl | 三语言路由、消息资源与服务端格式化 |
| Tailwind CSS | MIT | https://github.com/tailwindlabs/tailwindcss | 原子化样式与主题变量 |
| Lucide React | ISC | https://github.com/lucide-icons/lucide | 图标组件 |
| sanitize-html | MIT | https://github.com/apostrophecms/sanitize-html | 正文 HTML 白名单清洗 |
| cheerio | MIT | https://github.com/cheeriojs/cheerio | 服务端提取标题、生成目录锚点 |
| clsx | MIT | https://github.com/lukeed/clsx | 条件 className 合并 |
| tailwind-merge | MIT | https://github.com/dcastil/tailwind-merge | 合并并消除冲突的 Tailwind 类名 |

## 直接开发依赖

| 项目 | 许可证 | 来源 | 使用范围 |
| --- | --- | --- | --- |
| TypeScript | Apache-2.0 | https://github.com/microsoft/TypeScript | 严格类型检查 |
| ESLint | MIT | https://github.com/eslint/eslint | 代码质量检查 |
| eslint-config-next | MIT | https://github.com/vercel/next.js | Next.js 官方 ESLint 规则集 |
| TypeDoc | Apache-2.0 | https://github.com/TypeStrong/typedoc | 公共 API 文档生成 |
| PostCSS / @tailwindcss/postcss | MIT | https://github.com/postcss/postcss | 样式构建管线 |

## 修改说明

以上依赖均按原样引入，未对第三方源码做任何改动。文章正文的清洗白名单与目录锚点逻辑为 <code>src/content/sanitizeArticle.ts</code> 中的自主实现，仅调用 sanitize-html 与 cheerio 的公开 API。

## 许可证风险说明

本项目未引入 GPL/AGPL 传播型许可证依赖。Lucide 为 ISC 许可证；其余均为 MIT 或 Apache-2.0，允许商业使用与再分发。
