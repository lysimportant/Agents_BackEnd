'use client';

import { useEffect, useRef, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Globe } from 'lucide-react';
import { LOCALE_LABELS, SUPPORTED_LOCALES } from '@/config/constants';
import { usePathname, useRouter } from '@/i18n/navigation';
import { cn } from '@/utils/cn';

/** LanguageSwitcher 切换当前路径的等价语言 URL，并尽量保留筛选、分页与位置。 */
export function LanguageSwitcher() {
  const t = useTranslations('language');
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };
    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t('switch')}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-11 items-center gap-1.5 rounded-lg px-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <Globe className="h-5 w-5" aria-hidden="true" />
        <span className="hidden text-sm font-medium sm:inline">{locale}</span>
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute right-0 top-full z-50 mt-2 w-40 overflow-hidden rounded-lg border border-border bg-surface p-1 shadow-lg"
        >
          {SUPPORTED_LOCALES.map((item) => {
            const selected = locale === item;
            return (
              <button
                key={item}
                type="button"
                role="menuitemradio"
                aria-checked={selected}
                onClick={() => {
                  router.replace(pathname, { locale: item });
                  setOpen(false);
                }}
                className={cn(
                  'flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-muted',
                  selected && 'text-primary',
                )}
              >
                {LOCALE_LABELS[item]}
                <span className="text-xs text-muted-foreground">{item}</span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
