/**
 * 门户中间件，负责根据 portal-locale Cookie 重定向根路径并处理语言路由。
 * 读取 portal-locale Cookie，并同步回写 NEXT_LOCALE 供 next-intl 使用。
 */
import { NextRequest, NextResponse } from 'next/server';
import createIntlMiddleware from 'next-intl/middleware';
import { routing, isSupportedLocale } from '@/i18n/routing';

/** readLocaleCookie 从请求中读取并校验 portal-locale Cookie。 */
function readLocaleCookie(request: NextRequest): string | undefined {
  const raw = request.cookies.get('portal-locale')?.value;
  return isSupportedLocale(raw) ? (raw as string) : undefined;
}

/** intlMiddleware 创建 next-intl 的语言路由中间件。 */
const intlMiddleware = createIntlMiddleware(routing);

/** middleware 根据 Cookie 处理根路径重定向，其余请求交由 next-intl。 */
export default function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (pathname === '/') {
    const cookieLocale = readLocaleCookie(request);
    if (cookieLocale) {
      const url = request.nextUrl.clone();
      url.pathname = '/' + cookieLocale;
      const response = NextResponse.redirect(url);
      response.cookies.set('NEXT_LOCALE', cookieLocale, {
        path: '/',
        maxAge: 60 * 60 * 24 * 365,
        sameSite: 'lax',
      });
      return response;
    }
  }

  return intlMiddleware(request);
}

/** config 声明中间件匹配路径，排除静态资源与 API。 */
export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico|robots.txt|sitemap.xml|feed.xml).*)'],
};