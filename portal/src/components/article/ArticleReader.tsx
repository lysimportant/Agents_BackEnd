'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import type { PublicTocEntry } from '@/types/publicContent';
import { cn } from '@/utils/cn';

/**
 * ArticleReader 渲染已清洗的文章正文、目录锚点与阅读进度。
 * 正文在服务端完成白名单清洗后传入，这里仅通过 dangerouslySetInnerHTML 输出受控内容。
 */
export function ArticleReader({
  html,
  tableOfContents,
  contentLocale,
}: {
  html: string;
  tableOfContents: PublicTocEntry[];
  contentLocale: string;
}) {
  const t = useTranslations('a11y');
  const articleRef = useRef<HTMLElement>(null);
  const [progress, setProgress] = useState(0);

  // 监听滚动，计算文章阅读进度，使用 rAF 节流避免高频重排。
  useEffect(() => {
    let frame = 0;
    const update = () => {
      const article = articleRef.current;
      if (!article) {
        return;
      }
      const rect = article.getBoundingClientRect();
      const total = rect.height - window.innerHeight;
      const scrolled = -rect.top;
      const ratio = total > 0 ? Math.min(1, Math.max(0, scrolled / total)) : 1;
      setProgress(ratio);
    };
    const onScroll = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(update);
    };
    update();
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll, { passive: true });
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener('scroll', onScroll);
      window.removeEventListener('resize', onScroll);
    };
  }, []);

  return (
    <div className="relative">
      {/* 阅读进度条 */}
      <div
        className="fixed left-0 top-0 z-50 h-1 origin-left bg-primary transition-transform duration-100"
        style={{ transform: 'scaleX(' + progress + ')' }}
        role="progressbar"
        aria-label={t('readingProgress')}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(progress * 100)}
      />

      <div className="lg:grid lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
        {tableOfContents.length > 0 ? (
          <aside className="mb-8 lg:mb-0">
            <nav aria-label={t('articleToc')} className="lg:sticky lg:top-24">
              <p className="mb-3 text-sm font-semibold">{t('articleToc')}</p>
              <ul className="space-y-1 border-l border-border">
                {tableOfContents.map((entry) => (
                  <li key={entry.id}>
                    <a
                      href={'#' + entry.id}
                      className={cn(
                        'block rounded-r-md py-1 text-sm text-muted-foreground transition-colors hover:text-foreground',
                        entry.level === 2 && 'pl-3',
                        entry.level === 3 && 'pl-6 text-xs',
                        entry.level >= 4 && 'pl-9 text-xs',
                      )}
                    >
                      {entry.text}
                    </a>
                  </li>
                ))}
              </ul>
            </nav>
          </aside>
        ) : null}

        <div className="min-w-0">
          <article
            ref={articleRef}
            lang={contentLocale}
            className="article-content max-w-[76ch]"
            dangerouslySetInnerHTML={{ __html: html }}
          />
        </div>
      </div>
    </div>
  );
}
