import { cookies } from 'next/headers';
import { defaultRevalidate } from './publicApi';
import type { PublicFetchOptions } from './publicApi';

/**
 * 构造服务端公开请求选项，转发当前访客的后端认证 Cookie。
 * 带登录态的请求禁止共享缓存，避免 18R 结果泄露给匿名访客。
 */
export async function serverPublicFetchOptions(): Promise<PublicFetchOptions> {
  const cookieStore = await cookies();
  const sessionID = cookieStore.get('sessionId')?.value;
  const portalR18 = cookieStore.get('portal-r18')?.value;
  const cookieParts = [
    sessionID ? 'sessionId=' + sessionID : '',
    portalR18 === '1' ? 'portal-r18=1' : '',
  ].filter(Boolean);
  const cookieHeader = cookieParts.join('; ');
  return {
    credentials: 'include',
    headers: cookieHeader ? { cookie: cookieHeader } : undefined,
    revalidate: cookieHeader ? undefined : defaultRevalidate(),
  };
}
