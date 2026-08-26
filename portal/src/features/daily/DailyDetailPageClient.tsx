'use client';

import { useEffect, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import Image from 'next/image';
import { Eye, Heart, Send, UserRound } from 'lucide-react';
import { Link } from '@/i18n/navigation';
import { useAuth } from '@/features/auth/AuthProvider';
import { createDailyComment, getDailyInteraction, toggleDailyLike } from '@/services/publicApi';
import { resolveMediaUrl } from '@/config/constants';
import { formatCount, formatDateTime } from '@/utils/format';
import type { PublicDailyInteraction, PublicDailyItem, PublicTocEntry } from '@/types/publicContent';

/** 日常详情页渲染正文目录，并负责点赞与评论互动。 */
export function DailyDetailPageClient({
  daily,
  html,
  tableOfContents,
}: {
  daily: PublicDailyItem;
  html: string;
  tableOfContents: PublicTocEntry[];
}) {
  const t = useTranslations('daily');
  const common = useTranslations('common');
  const locale = useLocale();
  const { isLoggedIn } = useAuth();
  const [interaction, setInteraction] = useState<PublicDailyInteraction | null>(null);
  const [commentContent, setCommentContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isTogglingLike, setIsTogglingLike] = useState(false);
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    void getDailyInteraction(daily.id, { signal: controller.signal })
      .then(setInteraction)
      .catch(() => {
        if (!controller.signal.aborted) {
          setError(t('interactionLoadFailed'));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false);
        }
      });
    return () => controller.abort();
  }, [daily.id, t]);

  const toggleLike = async () => {
    if (!isLoggedIn) {
      setMessage(t('loginToLike'));
      return;
    }
    if (isTogglingLike) {
      return;
    }
    setIsTogglingLike(true);
    setError('');
    try {
      setInteraction(await toggleDailyLike(daily.id));
    } catch {
      setError(t('interactionUpdateFailed'));
    } finally {
      setIsTogglingLike(false);
    }
  };

  const submitComment = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!isLoggedIn) {
      setMessage(t('loginToComment'));
      return;
    }
    const normalized = commentContent.trim();
    if (!normalized || isSubmittingComment) {
      return;
    }
    setIsSubmittingComment(true);
    setError('');
    try {
      const comment = await createDailyComment(daily.id, normalized);
      setInteraction((current) => ({
        likeCount: current?.likeCount ?? daily.likeCount,
        likedByCurrentUser: current?.likedByCurrentUser ?? false,
        comments: [...(current?.comments ?? []), comment].slice(-100),
      }));
      setCommentContent('');
    } catch {
      setError(t('commentSendFailed'));
    } finally {
      setIsSubmittingComment(false);
    }
  };

  const likeCount = interaction?.likeCount ?? daily.likeCount ?? 0;

  return (
    <div className="mt-8 space-y-8">
      <Link href="/daily" className="inline-flex min-h-11 items-center rounded-lg px-3 text-sm text-muted-foreground hover:bg-muted hover:text-foreground">
        {t('backToDaily')}
      </Link>

      <header className="overflow-hidden rounded-lg border border-border bg-surface">
        {daily.coverImage ? <Image src={resolveMediaUrl(daily.coverImage)} alt={daily.coverAlt || t('coverAlt')} width={daily.coverWidth || 1280} height={daily.coverHeight || 720} unoptimized priority className="max-h-[28rem] w-full object-cover" /> : null}
        <div className="p-5 sm:p-7">
          <h1 className="text-2xl font-bold">{t('detailTitle')}</h1>
          <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <span className="inline-flex items-center gap-1"><UserRound className="h-4 w-4" aria-hidden="true" />{daily.authorName}</span>
            <time dateTime={daily.createdAt}>{formatDateTime(daily.createdAt, locale)}</time>
            <span className="inline-flex items-center gap-1"><Eye className="h-4 w-4" aria-hidden="true" />{formatCount(daily.views, locale)}</span>
            <span className="inline-flex items-center gap-1"><Heart className="h-4 w-4" aria-hidden="true" />{formatCount(likeCount, locale)}</span>
          </div>
        </div>
      </header>

      <div className="lg:grid lg:grid-cols-[minmax(0,1fr)_220px] lg:gap-10">
        <article className="daily-detail-content article-content min-w-0 max-w-[76ch]" dangerouslySetInnerHTML={{ __html: html || '<p></p>' }} />
        {tableOfContents.length > 0 ? <aside className="mt-8 lg:order-last lg:mt-0"><nav aria-label={t('toc')} className="lg:sticky lg:top-24"><p className="mb-3 text-sm font-semibold">{t('toc')}</p><ul className="space-y-1 border-l border-border">{tableOfContents.map((entry) => <li key={entry.id}><a href={'#' + entry.id} className={`block rounded-r-md py-1 text-sm text-muted-foreground hover:text-foreground ${entry.level === 2 ? 'pl-3' : entry.level >= 3 ? 'pl-6 text-xs' : ''}`}>{entry.text}</a></li>)}</ul></nav></aside> : null}
      </div>

      <section className="border-t border-border pt-6" aria-labelledby="daily-comments-title">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id="daily-comments-title" className="text-xl font-semibold">{t('comments')}</h2>
          <button type="button" onClick={() => void toggleLike()} aria-pressed={interaction?.likedByCurrentUser ?? false} disabled={isTogglingLike} className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-border px-4 text-sm transition-colors hover:bg-muted disabled:opacity-50"><Heart className="h-4 w-4" aria-hidden="true" />{interaction?.likedByCurrentUser ? t('unlike') : t('like')}<span>{formatCount(likeCount, locale)}</span></button>
        </div>
        {message ? <p className="mt-3 text-sm text-muted-foreground" role="status">{message}</p> : null}
        {error ? <p className="mt-3 text-sm text-red-500" role="alert">{error}</p> : null}
        {isLoading ? <p className="mt-4 text-sm text-muted-foreground">{common('loading')}</p> : null}
        {!isLoading && interaction?.comments.length === 0 ? <p className="mt-4 text-sm text-muted-foreground">{t('noComments')}</p> : null}
        <div className="mt-4 space-y-3">{interaction?.comments.map((comment) => <article key={comment.id} className="border-b border-border pb-3"><div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground"><span>{comment.userName}</span><time dateTime={comment.createdAt}>{formatDateTime(comment.createdAt, locale)}</time></div><p className="mt-1 whitespace-pre-wrap text-sm">{comment.content}</p></article>)}</div>
        <form onSubmit={submitComment} className="mt-5 flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="min-w-0 flex-1"><span className="sr-only">{t('commentPlaceholder')}</span><textarea value={commentContent} onChange={(event) => setCommentContent(event.target.value)} maxLength={500} rows={3} placeholder={t('commentPlaceholder')} className="w-full resize-y rounded-lg border border-border bg-background p-3 text-sm outline-none focus:border-primary" /></label>
          <button type="submit" disabled={isSubmittingComment || !commentContent.trim()} className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground disabled:opacity-50"><Send className="h-4 w-4" aria-hidden="true" />{isSubmittingComment ? common('loading') : t('sendComment')}</button>
        </form>
      </section>
    </div>
  );
}
