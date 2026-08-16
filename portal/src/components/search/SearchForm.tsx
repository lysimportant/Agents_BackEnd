'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Search } from 'lucide-react';
import { useRouter } from '@/i18n/navigation';

/** SearchForm 提交关键词并导航到带关键词查询参数的搜索 URL。 */
export function SearchForm({ initialKeyword }: { initialKeyword: string }) {
  const t = useTranslations('search');
  const router = useRouter();
  const [keyword, setKeyword] = useState(initialKeyword);

  return (
    <form
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        const trimmed = keyword.trim();
        router.replace('/search' + (trimmed ? '?keyword=' + encodeURIComponent(trimmed) : ''));
      }}
      className="flex w-full max-w-xl items-center gap-2"
    >
      <label className="relative min-w-0 flex-1">
        <span className="sr-only">{t('placeholder')}</span>
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <input
          type="search"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t('placeholder')}
          className="h-12 w-full rounded-lg border border-border bg-surface pl-10 pr-3 text-base outline-none transition-colors focus:border-primary"
        />
      </label>
      <button
        type="submit"
        className="inline-flex h-12 shrink-0 items-center gap-1.5 rounded-lg bg-primary px-5 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
      >
        {t('submit')}
      </button>
    </form>
  );
}
