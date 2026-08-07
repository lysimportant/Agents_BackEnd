/**
 * 门户站点页头，包含站点名、主导航、搜索、语言与主题切换。
 * 移动端以汉堡菜单收纳导航，并保持与桌面端一致的交互语义。
 */
'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { usePathname } from '@/navigation';
import { Search, X, Menu } from 'lucide-react';
import { ThemeSwitcher } from '@/components/theme/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/site/LanguageSwitcher';

/** SiteHeader 渲染门户顶部导航。 */
export function SiteHeader() {
  const t = useTranslations();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);

  const navItems = [
    { href: '/', label: t('nav.home'), match: pathname === '/' },
    { href: '/articles', label: t('nav.articles'), match: pathname.startsWith('/articles') },
    { href: '/images', label: t('nav.images'), match: pathname.startsWith('/images') },
    { href: '/resources', label: t('nav.resources'), match: pathname.startsWith('/resources') },
    { href: '/categories', label: t('nav.categories'), match: pathname.startsWith('/categories') },
    { href: '/about', label: t('nav.about'), match: pathname.startsWith('/about') },
  ];

  return (
    <header className="glass-nav sticky top-0 z-40">
      <nav className="mx-auto flex h-16 max-w-7xl items-center gap-3 px-4 sm:px-6" aria-label={t('common.menu')}>
        <Link
          href="/"
          className="tap-target flex shrink-0 items-center gap-2 rounded-md px-1 font-semibold tracking-tight"
        >
          <span className="text-lg">{t('common.siteName')}</span>
        </Link>

        <div className="ml-4 hidden items-center gap-1 lg:flex">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={item.match ? 'page' : undefined}
              className={'tap-target flex items-center rounded-md px-3 text-sm font-medium transition-colors ' + (item.match ? 'bg-accent-soft text-accent' : 'text-ink-muted hover:bg-surface hover:text-ink')}
            >
              {item.label}
            </Link>
          ))}
        </div>

        <div className="ml-auto flex items-center gap-1">
          <Link
            href="/search"
            aria-label={t('nav.search')}
            className="tap-target flex items-center justify-center rounded-md p-2 text-ink-muted hover:bg-surface hover:text-ink"
          >
            <Search className="h-5 w-5" aria-hidden="true" />
          </Link>
          <div className="hidden sm:block"><LanguageSwitcher /></div>
          <div className="hidden sm:block"><ThemeSwitcher /></div>
          <button
            type="button"
            aria-label={t('common.menu')}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((open) => !open)}
            className="tap-target flex items-center justify-center rounded-md p-2 text-ink-muted hover:bg-surface hover:text-ink lg:hidden"
          >
            {menuOpen ? <X className="h-5 w-5" aria-hidden="true" /> : <Menu className="h-5 w-5" aria-hidden="true" />}
          </button>
        </div>
      </nav>

      {menuOpen && (
        <div className="border-t border-line bg-canvas px-4 py-3 lg:hidden">
          <ul className="flex flex-col gap-1">
            {navItems.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={() => setMenuOpen(false)}
                  aria-current={item.match ? 'page' : undefined}
                  className={'tap-target flex items-center rounded-md px-3 text-sm font-medium ' + (item.match ? 'bg-accent-soft text-accent' : 'text-ink-muted')}
                >
                  {item.label}
                </Link>
              </li>
            ))}
            <li className="flex gap-2 border-t border-line pt-2 sm:hidden">
              <div className="flex-1"><LanguageSwitcher /></div>
              <div className="flex-1"><ThemeSwitcher /></div>
            </li>
          </ul>
        </div>
      )}
    </header>
  );
}