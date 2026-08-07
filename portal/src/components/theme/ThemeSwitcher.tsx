/**
 * 主题切换器，提供浅色、深色、海洋与跟随系统四种主题选择。
 */
'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Sun, Moon, Waves, Monitor, Palette } from 'lucide-react';
import { useTheme } from '@/components/theme/ThemeProvider';
import type { PortalTheme } from '@/theme';

/** OPTIONS 保存主题切换菜单的可选项与对应图标。 */
const OPTIONS: { value: PortalTheme; icon: typeof Sun }[] = [
  { value: 'light', icon: Sun },
  { value: 'dark', icon: Moon },
  { value: 'ocean', icon: Waves },
  { value: 'system', icon: Monitor },
];

/** ThemeSwitcher 渲染主题切换按钮与下拉菜单。 */
export function ThemeSwitcher() {
  const t = useTranslations('theme');
  const { preference, setPreference } = useTheme();
  const [open, setOpen] = useState(false);

  return (
    <div className="relative">
      <button
        type="button"
        aria-label={t('themeMenu')}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="tap-target flex items-center justify-center rounded-md p-2 text-ink-muted hover:bg-surface hover:text-ink"
      >
        <Palette className="h-5 w-5" aria-hidden="true" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} aria-hidden="true" />
          <div className="absolute right-0 z-20 mt-2 w-44 rounded-lg border border-line bg-surface p-2 shadow-card">
            {OPTIONS.map((opt) => {
              const Icon = opt.icon;
              const active = preference === opt.value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => { setPreference(opt.value); setOpen(false); }}
                  aria-pressed={active}
                  className={'flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm ' + (active ? 'bg-accent-soft text-accent' : 'text-ink-muted hover:bg-canvas')}
                >
                  <Icon className="h-4 w-4" aria-hidden="true" />
                  {t(opt.value)}
                </button>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}