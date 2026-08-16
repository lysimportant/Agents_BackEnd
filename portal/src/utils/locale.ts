import { cookies, headers } from 'next/headers';
import {
  DEFAULT_LOCALE,
  LOCALE_COOKIE_NAME,
  isSupportedLocale,
  type SupportedLocale,
} from '@/config/constants';

/** 从 Accept-Language 解析首个支持的语言，无法匹配时回退默认语言。 */
function resolveFromAcceptLanguage(acceptLanguage: string): SupportedLocale {
  for (const part of acceptLanguage.split(',')) {
    const lang = part.split(';')[0].trim().toLowerCase();
    if (lang.startsWith('zh')) {
      return 'zh-CN';
    }
    if (lang.startsWith('en')) {
      return 'en-US';
    }
    if (lang.startsWith('ja')) {
      return 'ja-JP';
    }
  }
  return DEFAULT_LOCALE;
}

/**
 * 解析访问者语言偏好：优先读取已校验的 portal-locale Cookie，其次 Accept-Language，最后回退默认语言。
 * 仅用于根路径 / 的服务端跳转，URL 语言段仍是页面语言的唯一事实来源。
 */
export async function resolveRequestLocale(): Promise<SupportedLocale> {
  const cookieStore = await cookies();
  const stored = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
  if (stored && isSupportedLocale(stored)) {
    return stored;
  }

  const headersList = await headers();
  const acceptLanguage = headersList.get('accept-language') ?? '';
  return resolveFromAcceptLanguage(acceptLanguage);
}
