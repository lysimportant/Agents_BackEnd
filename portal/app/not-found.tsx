import Link from 'next/link';

/** 全局 404 页面，用于无语言段匹配的多段未知路径，按默认语言渲染基础 html/body。 */
export default function GlobalNotFound() {
  return (
    <html lang="zh-CN">
      <body className="bg-background text-foreground antialiased">
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
          <p className="text-5xl font-bold text-muted-foreground">404</p>
          <h1 className="text-xl font-semibold">页面不存在 / Page not found</h1>
          <p className="max-w-md text-sm text-muted-foreground">
            该内容不存在、已取消发布或无权访问。
          </p>
          <Link
            href="/"
            className="mt-2 inline-flex items-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground"
          >
            返回首页 / Back to home
          </Link>
        </div>
      </body>
    </html>
  );
}
