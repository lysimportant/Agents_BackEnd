'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Eye, Lock, Plus, UserRound } from 'lucide-react';
import { useAuth } from '@/features/auth/AuthProvider';
import { createDaily, listDailies } from '@/services/publicApi';
import type { PublicDailyItem } from '@/types/publicContent';

/** 日常页面负责加载公开内容、显示个人内容并控制发布确认弹窗。 */
export function DailyPageClient() {
  const t = useTranslations('daily');
  const common = useTranslations('common');
  const { isLoggedIn } = useAuth();
  const [dailies, setDailies] = useState<PublicDailyItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [content, setContent] = useState('');
  const [isPrivate, setIsPrivate] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const publishButtonRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const dialogInitialFocusRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    listDailies({ credentials: 'include' })
      .then((response) => { if (!cancelled) setDailies(response.items); })
      .catch(() => { if (!cancelled) setError(t('loadFailed')); })
      .finally(() => { if (!cancelled) setIsLoading(false); });
    return () => { cancelled = true; };
  }, [isLoggedIn, t]);

  // 发布确认框打开时把焦点移入对话框，并支持 Escape 关闭；关闭后还原到触发按钮。
  useEffect(() => {
    if (!dialogOpen) return;
    const publishButton = publishButtonRef.current;
    dialogInitialFocusRef.current?.focus();
    const handleDialogKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isSubmitting) setDialogOpen(false);
      if (event.key !== 'Tab') return;
      const focusableElements = dialogRef.current?.querySelectorAll<HTMLElement>(
        'button, input, textarea, select, a[href], [tabindex]:not([tabindex="-1"])',
      );
      if (!focusableElements?.length) return;
      const firstFocusableElement = focusableElements[0];
      const lastFocusableElement = focusableElements[focusableElements.length - 1];
      if (event.shiftKey && document.activeElement === firstFocusableElement) {
        event.preventDefault();
        lastFocusableElement.focus();
      } else if (!event.shiftKey && document.activeElement === lastFocusableElement) {
        event.preventDefault();
        firstFocusableElement.focus();
      }
    };
    document.addEventListener('keydown', handleDialogKeyDown);
    return () => {
      document.removeEventListener('keydown', handleDialogKeyDown);
      publishButton?.focus();
    };
  }, [dialogOpen, isSubmitting]);

  const openPublishDialog = () => {
    setError('');
    if (!isLoggedIn) {
      setError(t('loginRequired'));
      return;
    }
    if (!content.trim()) {
      setError(t('contentRequired'));
      return;
    }
    setDialogOpen(true);
  };

  const submitDaily = async () => {
    setIsSubmitting(true);
    setError('');
    try {
      const createdDaily = await createDaily(content.trim(), isPrivate);
      setDailies((currentDailies) => [createdDaily, ...currentDailies]);
      setContent('');
      setDialogOpen(false);
    } catch {
      setError(t('publishFailed'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mt-8 space-y-6">
      <section className="border-b border-border pb-6">
        <label className="block">
          <span className="mb-2 block text-sm font-medium">{t('contentLabel')}</span>
          <textarea value={content} onChange={(event) => setContent(event.target.value)} maxLength={2000} rows={4} placeholder={t('placeholder')} className="w-full resize-y rounded-lg border border-border bg-background p-3 text-sm outline-none focus:border-primary" />
        </label>
        {error ? <p className="mt-2 text-sm text-red-500">{error}</p> : null}
        <button ref={publishButtonRef} type="button" onClick={openPublishDialog} className="mt-3 inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground hover:opacity-90">
          <Plus className="h-4 w-4" aria-hidden="true" />
          {t('publish')}
        </button>
      </section>

      {isLoading ? <p className="text-sm text-muted-foreground">{common('loading')}</p> : null}
      {!isLoading && dailies.length === 0 ? <p className="text-sm text-muted-foreground">{t('empty')}</p> : null}
      <div className="space-y-4">
        {dailies.map((daily) => (
          <article key={daily.id} className="rounded-lg border border-border bg-surface p-4">
            <p className="whitespace-pre-wrap break-words text-sm leading-6">{daily.content}</p>
            <div className="mt-4 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1"><UserRound className="h-3.5 w-3.5" aria-hidden="true" />{daily.authorName}</span>
              <span className="inline-flex items-center gap-1"><Eye className="h-3.5 w-3.5" aria-hidden="true" />{daily.views}</span>
              {daily.isPrivate ? <span className="inline-flex items-center gap-1"><Lock className="h-3.5 w-3.5" aria-hidden="true" />{t('private')}</span> : null}
              <time dateTime={daily.createdAt}>{new Date(daily.createdAt).toLocaleString()}</time>
            </div>
          </article>
        ))}
      </div>

      {dialogOpen ? (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-overlay p-4">
          <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="daily-confirm-title" aria-describedby="daily-confirm-description" className="w-full max-w-sm rounded-lg border border-border bg-surface p-5 shadow-xl">
            <h2 id="daily-confirm-title" className="text-lg font-semibold">{t('confirmTitle')}</h2>
            <p id="daily-confirm-description" className="mt-2 text-sm text-muted-foreground">{t('confirmDescription')}</p>
            <div className="mt-4 space-y-2">
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 text-sm">
                <input ref={dialogInitialFocusRef} name="daily-visibility" type="radio" checked={!isPrivate} onChange={() => setIsPrivate(false)} className="mt-0.5" />
                <span><strong className="block">{t('publicOption')}</strong><span className="text-muted-foreground">{t('publicHint')}</span></span>
              </label>
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 text-sm">
                <input name="daily-visibility" type="radio" checked={isPrivate} onChange={() => setIsPrivate(true)} className="mt-0.5" />
                <span><strong className="block">{t('privateOption')}</strong><span className="text-muted-foreground">{t('privateHint')}</span></span>
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button type="button" onClick={() => setDialogOpen(false)} className="min-h-11 rounded-lg border border-border px-4 text-sm hover:bg-muted">{common('cancel')}</button>
              <button type="button" disabled={isSubmitting} onClick={() => void submitDaily()} className="min-h-11 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground disabled:opacity-50">{isSubmitting ? common('loading') : t('confirmPublish')}</button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
