'use client';

import { useEffect, useMemo, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import Image from 'next/image';
import { Eye, Heart, Lock, UserRound } from 'lucide-react';
import { Link } from '@/i18n/navigation';
import { useAuth } from '@/features/auth/AuthProvider';
import { listDailies } from '@/services/publicApi';
import { sanitizeArticle } from '@/content/sanitizeArticle';
import type { PublicDailyItem } from '@/types/publicContent';
import { resolveMediaUrl } from '@/config/constants';
import { formatCount, formatDateTime } from '@/utils/format';

/** 日常页面只负责加载公开内容和个人内容，发布入口通过按钮进入独立编辑页。 */
export function DailyPageClient() {
  const t = useTranslations('daily');
  const common = useTranslations('common');
  const locale = useLocale();
  const { isLoggedIn } = useAuth();
  const [dailies, setDailies] = useState<PublicDailyItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    listDailies({}, { credentials: 'include' })
      .then((response) => {
        if (!cancelled) {
          setDailies(response.items);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError(t('loadFailed'));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [isLoggedIn, t]);

  /** 对后端白名单清洗后的日常正文再次清洗，防止旧数据或代理响应绕过前端安全边界。 */
  const renderedDailies = useMemo(
    () => dailies.map((daily) => ({ daily, html: sanitizeArticle(daily.content).html })),
    [dailies],
  );

  return (
    <div className="mt-8 space-y-6">
      <section className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-6">
        <p className="text-sm text-muted-foreground">{t('listHint')}</p>
      </section>

      {isLoading ? <p className="text-sm text-muted-foreground">{common('loading')}</p> : null}
      {error ? <p className="text-sm text-red-500" role="alert">{error}</p> : null}
      {!isLoading && !error && dailies.length === 0 ? <p className="text-sm text-muted-foreground">{t('empty')}</p> : null}
      <div className="space-y-4">
        {renderedDailies.map(({ daily, html }) => (
          <Link key={daily.id} href={`/daily/${daily.id}`} className="block rounded-lg border border-border bg-surface transition-colors hover:border-primary/60">
            {daily.coverImage ? (
              <Image
                src={resolveMediaUrl(daily.coverImage)}
                alt={daily.coverAlt || t('coverAlt')}
                width={daily.coverWidth || 1280}
                height={daily.coverHeight || 720}
                unoptimized
                className="h-48 w-full rounded-t-lg object-cover sm:h-56"
              />
            ) : null}
            <article className="p-4">
              <div
                className="article-content line-clamp-5 max-w-none text-sm"
                dangerouslySetInnerHTML={{ __html: html || '<p></p>' }}
              />
              <div className="mt-4 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                <span className="inline-flex items-center gap-1">
                  <UserRound className="h-3.5 w-3.5" aria-hidden="true" />
                  {daily.authorName}
                </span>
                <span className="inline-flex items-center gap-1" title={t('views')}>
                  <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                  {formatCount(daily.views, locale)}
                </span>
                <span className="inline-flex items-center gap-1" title={t('likes')}>
                  <Heart className="h-3.5 w-3.5" aria-hidden="true" />
                  {formatCount(daily.likeCount ?? 0, locale)}
                </span>
              {daily.isPrivate ? (
                <span className="inline-flex items-center gap-1">
                  <Lock className="h-3.5 w-3.5" aria-hidden="true" />
                  {t('private')}
                </span>
              ) : null}
                <time dateTime={daily.createdAt}>{formatDateTime(daily.createdAt, locale)}</time>
              </div>
            </article>
          </Link>
        ))}
      </div>
    </div>
  );
}
