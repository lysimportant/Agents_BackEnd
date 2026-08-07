/**
 * 根布局：仅透传 next-intl 配置后的 children。
 * 实际 <html>/<body> 结构由 [locale]/layout 负责启用语言路由。
 */
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return children;
}