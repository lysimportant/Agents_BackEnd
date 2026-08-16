'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { FileText, Folder, Images, Info, LayoutGrid, Menu, Search, X } from 'lucide-react';
import { Link, usePathname } from '@/i18n/navigation';
import { ThemeToggle } from '@/components/theme/ThemeToggle';
import { LanguageSwitcher } from '@/components/language/LanguageSwitcher';
import { AuthControls } from '@/components/auth/AuthControls';
import { cn } from '@/utils/cn';

/** 顶部导航项，图片作为首页，故不单独列首页；每个项带图标美化。 */
const NAV_ITEMS: {
  key: 'images' | 'articles' | 'resources' | 'categories' | 'about';
  href: string;
  icon: typeof Images;
}[] = [
  { key: 'images', href: '/images', icon: Images },
  { key: 'articles', href: '/articles', icon: FileText },
  { key: 'resources', href: '/resources', icon: Folder },
  { key: 'categories', href: '/categories', icon: LayoutGrid },
  { key: 'about', href: '/about', icon: Info },
];

/** 判断导航项是否处于激活状态。 */
function isNavActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + '/');
}

/**
 * SiteHeader 是悬浮居中、胶囊造型、带高斯模糊的顶部导航：
 * 左侧放导航、右侧放搜索/主题/语言，中间留空。
 */
export function SiteHeader() {
  const t = useTranslations('navigation');
  const pathname = usePathname();
  const [scrolled, setScrolled] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // 使用 IntersectionObserver 监听顶部哨兵，判断是否已滚动。
  useEffect(() => {
    const sentinel = document.getElementById('header-sentinel');
    if (!sentinel || typeof IntersectionObserver === 'undefined') {
      const handleScroll = () => setScrolled(window.scrollY > 8);
      window.addEventListener('scroll', handleScroll, { passive: true });
      return () => window.removeEventListener('scroll', handleScroll);
    }
    const observer = new IntersectionObserver(
      (entries) => setScrolled(!entries[0].isIntersecting),
      { threshold: 0 },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, []);

  // 抽屉打开时锁定底层滚动，并在 Esc 时关闭。
  useEffect(() => {
    if (!drawerOpen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setDrawerOpen(false);
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [drawerOpen]);

  return (
    <header className="pointer-events-none fixed inset-x-0 top-4 z-40 px-4">
      <nav
        className={cn(
          'site-header pointer-events-auto mx-auto flex h-14 w-full max-w-7xl items-center justify-between gap-2 rounded-full border px-3 shadow-sm transition-all duration-300',
          scrolled ? 'shadow-lg' : 'shadow-sm',
        )}
      >
        {/* 左侧：导航 */}
        <div className="flex min-w-0 items-center gap-1">
          <ul className="hidden items-center gap-1 md:flex">
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              const active = isNavActive(pathname, item.href);
              return (
                <li key={item.key}>
                  <Link
                    href={item.href}
                    aria-current={active ? 'page' : undefined}
                    className={cn(
                      'flex items-center gap-1.5 rounded-full px-3 py-2 text-sm font-medium transition-colors',
                      active
                        ? 'bg-muted text-foreground'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                    )}
                  >
                    <Icon className="h-4 w-4" aria-hidden="true" />
                    {t(item.key)}
                  </Link>
                </li>
              );
            })}
          </ul>

          <button
            type="button"
            onClick={() => setDrawerOpen(true)}
            aria-label={t('openMenu')}
            className="flex h-11 w-11 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:hidden"
          >
            <Menu className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* 右侧：搜索 + 主题 + 语言 */}
        <div className="flex shrink-0 items-center gap-1">
          <Link
            href="/search"
            aria-label={t('searchToggle')}
            className="flex h-11 w-11 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Search className="h-5 w-5" aria-hidden="true" />
          </Link>
          <ThemeToggle />
          <LanguageSwitcher />
          <AuthControls />
        </div>
      </nav>

      {/* 移动端抽屉菜单 */}
      <div
        className={cn(
          'fixed inset-0 z-50 md:hidden',
          drawerOpen ? 'pointer-events-auto' : 'pointer-events-none',
        )}
        aria-hidden={!drawerOpen}
      >
        <div
          onClick={() => setDrawerOpen(false)}
          className={cn(
            'absolute inset-0 bg-overlay backdrop-blur-sm transition-opacity duration-300',
            drawerOpen ? 'opacity-100' : 'opacity-0',
          )}
        />
        <aside
          className={cn(
            'absolute right-0 top-0 flex h-full w-72 max-w-[82vw] flex-col border-l border-border bg-surface p-4 transition-transform duration-300',
            drawerOpen ? 'translate-x-0' : 'translate-x-full',
          )}
        >
          <div className="flex items-center justify-between">
            <span className="font-semibold">{t('images')}</span>
            <button
              type="button"
              onClick={() => setDrawerOpen(false)}
              aria-label={t('closeMenu')}
              className="flex h-11 w-11 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted"
            >
              <X className="h-5 w-5" aria-hidden="true" />
            </button>
          </div>
          <ul className="mt-4 flex flex-col gap-1">
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <li key={item.key}>
                  <Link
                    href={item.href}
                    onClick={() => setDrawerOpen(false)}
                    className={cn(
                      'flex items-center gap-2 rounded-lg px-3 py-3 text-base font-medium transition-colors hover:bg-muted',
                      isNavActive(pathname, item.href) ? 'text-primary' : 'text-foreground',
                    )}
                  >
                    <Icon className="h-5 w-5" aria-hidden="true" />
                    {t(item.key)}
                  </Link>
                </li>
              );
            })}
          </ul>
        </aside>
      </div>
    </header>
  );
}
