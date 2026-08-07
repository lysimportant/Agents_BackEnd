/**
 * 语言切换器，切换后写入 portal-locale Cookie 并重定向到对应语言路由。
 */
'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { usePathname, useRouter } from '@/navigation';
import { SUPPORTED_LOCALES } from '@/i18n/routing';
import { Languages } from 'lucide-react';
import { writeLocaleCookie } from '@/utils/cookies';

/** LOCALE_NAMES 保存各语言在切换菜单中的显示名称。 */
const LOCALE_NAMES: Record<string, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
  'ja-JP': '日本語',
};

/** LanguageSwitcher 渲染语言切换按钮与下拉菜单。 */
export function LanguageSwitcher() {
  const t = useTranslations('common');
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false);

  function switchLocale(locale: string) {
    writeLocaleCookie(locale);
    router.replace(pathname, { locale });
    setOpen(false);
  }

  return (
    <div className="relative">
      <button
        type="button"
        aria-label={t('language')}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="tap-target flex items-center justify-center rounded-md p-2 text-ink-muted hover:bg-surface hover:text-ink"
      >
        <Languages className="h-5 w-5" aria-hidden="true" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} aria-hidden="true" />
          <div className="absolute right-0 z-20 mt-2 w-40 rounded-lg border border-line bg-surface p-2 shadow-card">
            {SUPPORTED_LOCALES.map((loc) => (
              <button
                key={loc}
                type="button"
                onClick={() => switchLocale(loc)}
                className="flex w-full items-center rounded-md px-3 py-2 text-sm text-ink-muted hover:bg-canvas hover:text-ink"
              >
                {LOCALE_NAMES[loc]}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}