'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Monitor, Moon, Sun, Waves } from 'lucide-react';
import { THEME_KEYS, type ThemeKey } from '@/config/constants';
import { useTheme } from '@/theme/ThemeProvider';
import { cn } from '@/utils/cn';

/** 各主题对应的图标，颜色不能作为表达状态的唯一方式，均配文字标签。 */
const THEME_ICONS: Record<ThemeKey, typeof Sun> = {
  light: Sun,
  dark: Moon,
  ocean: Waves,
  system: Monitor,
};

/** ThemeToggle 提供浅色、深色、海洋品牌与跟随系统四种主题的选择。 */
export function ThemeToggle() {
  const t = useTranslations('theme');
  const { preference, setPreference } = useTheme();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // 点击外部或按 Esc 关闭菜单。
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

  const ActiveIcon = THEME_ICONS[preference];

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t('label')}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-11 w-11 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <ActiveIcon className="h-5 w-5" aria-hidden="true" />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute right-0 top-full z-50 mt-2 w-44 overflow-hidden rounded-lg border border-border bg-surface p-1 shadow-lg"
        >
          {THEME_KEYS.map((key) => {
            const Icon = THEME_ICONS[key];
            const selected = preference === key;
            return (
              <button
                key={key}
                type="button"
                role="menuitemradio"
                aria-checked={selected}
                onClick={() => {
                  setPreference(key);
                  setOpen(false);
                }}
                className={cn(
                  'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-muted',
                  selected && 'text-primary',
                )}
              >
                <Icon className="h-4 w-4" aria-hidden="true" />
                {t(key)}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
