'use client';

import { useLocale } from 'next-intl';
import { useEffect } from 'react';
import { LOCALE_COOKIE_NAME } from '@/config/constants';

/**
 * LocaleCookieSync 在客户端把当前 URL 语言同步到 portal-locale Cookie，
 * 使后续访问根路径 / 时能跳转到用户最近使用的语言。
 */
export function LocaleCookieSync() {
  const locale = useLocale();

  useEffect(() => {
    document.cookie =
      LOCALE_COOKIE_NAME + '=' + locale + '; path=/; max-age=31536000; SameSite=Lax';
  }, [locale]);

  return null;
}
