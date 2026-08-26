'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Check, ChevronLeft, ChevronRight, Copy, Download, Heart, LoaderCircle, Maximize2, Minimize2, Plus, Send, X, ZoomIn, ZoomOut } from 'lucide-react';
import { useAuth } from '@/features/auth/AuthProvider';
import { addImageTag, createImageComment, getImageInteraction, toggleImageLike } from '@/services/publicApi';
import type { PublicFileInteraction } from '@/types/publicContent';
import type { GalleryImage } from './types';

/** 图片预览浮层，支持左右切换、缩放、拖动、Esc 关闭与移动端手势。 */
export function ImagePreviewOverlay({
  images,
  index,
  onIndexChange,
  onTagsChange,
  onLikeCountChange,
  onClose,
  returnFocusRef,
}: {
  images: GalleryImage[];
  index: number;
  onIndexChange: (index: number) => void;
  onTagsChange: (imageID: number, tags: string[]) => void;
  onLikeCountChange: (imageID: number, likeCount: number) => void;
  onClose: () => void;
  returnFocusRef: React.RefObject<HTMLElement | null>;
}) {
  const t = useTranslations('images');
  const commonT = useTranslations('common');
  const locale = useLocale();
  const { isLoggedIn } = useAuth();
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const copiedResetTimerRef = useRef<number | null>(null);
  // messageResetTimerRef 保存临时操作提示的关闭计时器，连续提示时用于重置时长。
  const messageResetTimerRef = useRef<number | null>(null);
  const activeImageIDRef = useRef(images[index]?.id ?? 0);
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [isCopied, setIsCopied] = useState(false);
  const [interaction, setInteraction] = useState<PublicFileInteraction | null>(null);
  const [isInteractionLoading, setIsInteractionLoading] = useState(true);
  const [interactionError, setInteractionError] = useState('');
  const [commentContent, setCommentContent] = useState('');
  // tagContent 保存当前图片待提交的单个标签输入。
  const [tagContent, setTagContent] = useState('');
  // portalMessage 保存匿名登录提示或标签写入结果的短暂可见文案。
  const [portalMessage, setPortalMessage] = useState('');
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
  // isSubmittingTag 防止同一个标签在请求完成前被重复提交。
  const [isSubmittingTag, setIsSubmittingTag] = useState(false);
  const [isTogglingLike, setIsTogglingLike] = useState(false);
  const dragStart = useRef<{ x: number; y: number; startX: number; startY: number; scaled: boolean } | null>(null);

  const current = images[index];
  const currentID = current?.id ?? 0;
  const hasPrevious = index > 0;
  const hasNext = index < images.length - 1;

  /** 切换图片前重置仅属于上一张图片的交互状态。 */
  const changePreviewIndex = useCallback((nextIndex: number) => {
    if (copiedResetTimerRef.current !== null) {
      window.clearTimeout(copiedResetTimerRef.current);
      copiedResetTimerRef.current = null;
    }
    setIsCopied(false);
    setInteraction(null);
    setIsInteractionLoading(true);
    setInteractionError('');
    setCommentContent('');
    setTagContent('');
    setPortalMessage('');
    setIsSubmittingComment(false);
    setIsSubmittingTag(false);
    setIsTogglingLike(false);
    activeImageIDRef.current = images[nextIndex]?.id ?? 0;
    onIndexChange(nextIndex);
  }, [images, onIndexChange]);

  const goPrevious = useCallback(() => {
    if (hasPrevious) {
      changePreviewIndex(index - 1);
    }
  }, [changePreviewIndex, hasPrevious, index]);

  const goNext = useCallback(() => {
    if (hasNext) {
      changePreviewIndex(index + 1);
    }
  }, [changePreviewIndex, hasNext, index]);

  const resetTransform = useCallback(() => {
    setScale(1);
    setOffset({ x: 0, y: 0 });
  }, []);

  /** 显示短暂的门户操作 message，并让连续提示刷新停留时间。 */
  const showPortalMessage = useCallback((message: string) => {
    setPortalMessage(message);
    if (messageResetTimerRef.current !== null) {
      window.clearTimeout(messageResetTimerRef.current);
    }
    messageResetTimerRef.current = window.setTimeout(() => {
      setPortalMessage('');
      messageResetTimerRef.current = null;
    }, 2200);
  }, []);

  // 切换图片时在渲染阶段重置缩放与位移（React 官方“根据上次渲染状态调整”模式）。
  const [previousIndex, setPreviousIndex] = useState(index);
  if (previousIndex !== index) {
    setPreviousIndex(index);
    setScale(1);
    setOffset({ x: 0, y: 0 });
  }

  // 打开时锁定滚动、聚焦关闭按钮，关闭时归还焦点。
  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    closeButtonRef.current?.focus();
    const returnFocus = returnFocusRef.current;
    return () => {
      document.body.style.overflow = previousOverflow;
      returnFocus?.focus();
    };
  }, [returnFocusRef]);

  // 预览关闭或组件卸载时清理复制状态计时器，避免悬挂的状态更新。
  useEffect(() => {
    return () => {
      if (copiedResetTimerRef.current !== null) {
        window.clearTimeout(copiedResetTimerRef.current);
      }
      if (messageResetTimerRef.current !== null) {
        window.clearTimeout(messageResetTimerRef.current);
      }
    };
  }, []);

  // 当前图片变化时读取点赞和评论；关闭或切换时取消旧请求。
  useEffect(() => {
    if (currentID <= 0) {
      return;
    }
    const requestController = new AbortController();
    void getImageInteraction(currentID, { signal: requestController.signal })
      .then((nextInteraction) => {
        setInteraction(nextInteraction);
        onLikeCountChange(currentID, nextInteraction.likeCount);
      })
      .catch(() => {
        if (!requestController.signal.aborted) {
          setInteractionError(t('interactionLoadFailed'));
        }
      })
      .finally(() => {
        if (!requestController.signal.aborted) {
          setIsInteractionLoading(false);
        }
      });
    return () => requestController.abort();
  }, [currentID, onLikeCountChange, t]);

  // 键盘交互：左右切换、缩放、Esc 关闭。
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      } else if (event.key === 'ArrowLeft') {
        goPrevious();
      } else if (event.key === 'ArrowRight') {
        goNext();
      } else if (event.key === '+' || event.key === '=') {
        setScale((value) => Math.min(3, value + 0.5));
      } else if (event.key === '-') {
        setScale((value) => Math.max(1, value - 0.5));
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [goNext, goPrevious, onClose]);

  if (!current) {
    return null;
  }

  const handleZoomIn = () => setScale((value) => Math.min(3, value + 0.5));
  const handleZoomOut = () => {
    setScale((value) => {
      const next = Math.max(1, value - 0.5);
      if (next === 1) {
        setOffset({ x: 0, y: 0 });
      }
      return next;
    });
  };

  /** 将当前公开原图地址复制到剪贴板，并兼容不支持 Clipboard API 的浏览器。 */
  const copyCurrentImageUrl = async () => {
    let copied = false;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(current.src);
        copied = true;
      }
    } catch {
      copied = false;
    }

    if (!copied) {
      let temporaryInput: HTMLTextAreaElement | null = null;
      try {
        temporaryInput = document.createElement('textarea');
        temporaryInput.value = current.src;
        temporaryInput.setAttribute('readonly', '');
        temporaryInput.style.position = 'fixed';
        temporaryInput.style.opacity = '0';
        document.body.appendChild(temporaryInput);
        temporaryInput.select();
        copied = document.execCommand('copy');
      } catch {
        copied = false;
      } finally {
        temporaryInput?.remove();
      }
    }

    if (!copied) {
      return;
    }
    setIsCopied(true);
    if (copiedResetTimerRef.current !== null) {
      window.clearTimeout(copiedResetTimerRef.current);
    }
    copiedResetTimerRef.current = window.setTimeout(() => {
      setIsCopied(false);
      copiedResetTimerRef.current = null;
    }, 1600);
  };

  /** 切换当前用户点赞状态，并使用后端返回值刷新权威数量。 */
  const toggleCurrentLike = async () => {
    if (!isLoggedIn || isTogglingLike) {
      return;
    }
    setIsTogglingLike(true);
    setInteractionError('');
    // requestFileID 标识本次点赞所属图片，避免切图后旧响应覆盖新图片。
    const requestFileID = current.id;
    try {
      const nextInteraction = await toggleImageLike(requestFileID);
      if (activeImageIDRef.current === requestFileID) {
        setInteraction(nextInteraction);
        onLikeCountChange(requestFileID, nextInteraction.likeCount);
      }
    } catch {
      if (activeImageIDRef.current === requestFileID) {
        setInteractionError(t('interactionUpdateFailed'));
      }
    } finally {
      if (activeImageIDRef.current === requestFileID) {
        setIsTogglingLike(false);
      }
    }
  };

  /** 发送当前评论，并将后端返回的新评论追加到已加载列表。 */
  const submitCurrentComment = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!isLoggedIn) {
      showPortalMessage(t('loginToComment'));
      return;
    }
    const normalizedContent = commentContent.trim();
    if (!normalizedContent || isSubmittingComment) {
      return;
    }
    setIsSubmittingComment(true);
    setInteractionError('');
    // requestFileID 标识本次评论所属图片，避免切图后旧响应串入新图片。
    const requestFileID = current.id;
    try {
      const createdComment = await createImageComment(requestFileID, normalizedContent);
      if (activeImageIDRef.current === requestFileID) {
        setInteraction((currentInteraction) => ({
          likeCount: currentInteraction?.likeCount ?? current.likeCount,
          likedByCurrentUser: currentInteraction?.likedByCurrentUser ?? false,
          comments: [...(currentInteraction?.comments ?? []), createdComment],
        }));
        setCommentContent('');
      }
    } catch {
      if (activeImageIDRef.current === requestFileID) {
        setInteractionError(t('commentSendFailed'));
      }
    } finally {
      if (activeImageIDRef.current === requestFileID) {
        setIsSubmittingComment(false);
      }
    }
  };

  /** 追加当前图片标签，并用后端返回值刷新本次浏览中的权威标签。 */
  const submitCurrentTag = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!isLoggedIn) {
      showPortalMessage(t('loginToAddTag'));
      return;
    }
    // normalizedTag 去除用户习惯输入的井号前缀，并按 Unicode 字符限制提交内容。
    const normalizedTag = Array.from(tagContent.trim().replace(/^#+/, '').trim()).slice(0, 24).join('');
    if (!normalizedTag || isSubmittingTag) {
      return;
    }
    setIsSubmittingTag(true);
    // requestFileID 标识本次标签所属图片，避免切图后旧响应覆盖新图片。
    const requestFileID = current.id;
    try {
      // tagResponse 保存后端合并后的权威标签与实际新增状态。
      const tagResponse = await addImageTag(requestFileID, normalizedTag);
      if (activeImageIDRef.current === requestFileID) {
        onTagsChange(requestFileID, tagResponse.tags);
        setTagContent('');
        // resultMessage 区分成功、重复和达到统一数量上限三种结果。
        const resultMessage = tagResponse.added
          ? t('tagAdded')
          : tagResponse.tags.length >= 12
            ? t('tagLimitReached')
            : t('tagAlreadyExists');
        showPortalMessage(resultMessage);
      }
    } catch {
      if (activeImageIDRef.current === requestFileID) {
        showPortalMessage(t('tagAddFailed'));
      }
    } finally {
      if (activeImageIDRef.current === requestFileID) {
        setIsSubmittingTag(false);
      }
    }
  };
  const handleDoubleClick = () => {
    if (scale > 1) {
      resetTransform();
    } else {
      setScale(2);
    }
  };

  /** 鼠标滚轮缩放：向上放大、向下缩小，范围 1-3，回到 1 时复位位移。 */
  const handleWheel = (event: React.WheelEvent) => {
    const step = event.deltaY < 0 ? 0.2 : -0.2;
    setScale((value) => {
      const next = Math.min(3, Math.max(1, value + step));
      if (next === 1) {
        setOffset({ x: 0, y: 0 });
      }
      return next;
    });
  };

  const handlePointerDown = (event: React.PointerEvent) => {
    dragStart.current = {
      x: event.clientX,
      y: event.clientY,
      startX: offset.x,
      startY: offset.y,
      scaled: scale > 1,
    };
  };
  const handlePointerMove = (event: React.PointerEvent) => {
    if (!dragStart.current) {
      return;
    }
    const deltaX = event.clientX - dragStart.current.x;
    const deltaY = event.clientY - dragStart.current.y;
    if (dragStart.current.scaled) {
      setOffset({ x: dragStart.current.startX + deltaX, y: dragStart.current.startY + deltaY });
    }
  };
  const handlePointerUp = (event: React.PointerEvent) => {
    if (!dragStart.current) {
      return;
    }
    const deltaX = event.clientX - dragStart.current.x;
    if (!dragStart.current.scaled) {
      // 未缩放时，横向滑动超过阈值触发切换。
      if (Math.abs(deltaX) > 48) {
        if (deltaX < 0) {
          goNext();
        } else {
          goPrevious();
        }
      }
    }
    dragStart.current = null;
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-overlay backdrop-blur-md"
      role="dialog"
      aria-modal="true"
      aria-label={current.alt}
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      {portalMessage ? (
        <div className="pointer-events-none fixed left-1/2 top-5 z-[70] max-w-[calc(100vw-2rem)] -translate-x-1/2 rounded-lg bg-white px-4 py-2 text-center text-sm font-medium text-black shadow-lg" role="status" aria-live="polite">
          {portalMessage}
        </div>
      ) : null}
      <div className="flex items-center justify-between gap-2 px-4 py-3">
        <p className="truncate text-sm text-white/90">
          {index + 1} / {images.length}
        </p>
        <button
          ref={closeButtonRef}
          type="button"
          onClick={onClose}
          aria-label={t('closePreview')}
          className="flex h-11 w-11 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10"
        >
          <X className="h-5 w-5" aria-hidden="true" />
        </button>
      </div>

      <div className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_minmax(15rem,47vh)] gap-3 px-4 pb-4 lg:grid-cols-[minmax(0,1fr)_22rem] lg:grid-rows-1">
        <div className="relative flex min-h-0 items-center justify-center overflow-hidden">
          {hasPrevious ? (
            <button
              type="button"
              onClick={goPrevious}
              aria-label={t('previousImage')}
              className="absolute left-2 top-1/2 z-10 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-black/30 text-white transition-colors hover:bg-black/50"
            >
              <ChevronLeft className="h-6 w-6" aria-hidden="true" />
            </button>
          ) : null}

          <div
            className="flex h-full w-full touch-none items-center justify-center overflow-hidden"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={() => {
              dragStart.current = null;
            }}
            onDoubleClick={handleDoubleClick}
            onWheel={handleWheel}
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={current.src}
              alt={current.alt}
              className="max-h-full max-w-full select-none object-contain transition-transform duration-150"
              style={{
                transform: 'translate(' + offset.x + 'px, ' + offset.y + 'px) scale(' + scale + ')',
                cursor: scale > 1 ? 'grab' : 'zoom-in',
              }}
              draggable={false}
            />
          </div>

          {hasNext ? (
            <button
              type="button"
              onClick={goNext}
              aria-label={t('nextImage')}
              className="absolute right-2 top-1/2 z-10 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-black/30 text-white transition-colors hover:bg-black/50"
            >
              <ChevronRight className="h-6 w-6" aria-hidden="true" />
            </button>
          ) : null}
        </div>

        <aside className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-white/15 bg-black/35 text-white">
          <div className="border-b border-white/15 px-4 py-3">
            <p className="break-words text-sm font-medium">{current.displayName}</p>
            {current.category ? <p className="mt-1 text-xs text-white/70">{current.category}</p> : null}
            {current.tags.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {current.tags.map((tag) => (
                  <span key={tag} className="rounded border border-white/20 px-2 py-0.5 text-xs text-white/80">#{tag}</span>
                ))}
              </div>
            ) : null}
            <form className="mt-2 flex items-center gap-2" onSubmit={(event) => void submitCurrentTag(event)}>
              <label className="sr-only" htmlFor={'image-tag-' + current.id}>{t('tagPlaceholder')}</label>
              <input
                id={'image-tag-' + current.id}
                value={tagContent}
                onChange={(event) => setTagContent(Array.from(event.target.value).slice(0, 24).join(''))}
                onFocus={() => {
                  if (!isLoggedIn) {
                    showPortalMessage(t('loginToAddTag'));
                  }
                }}
                readOnly={!isLoggedIn}
                disabled={isSubmittingTag}
                maxLength={24}
                placeholder={isLoggedIn ? t('tagPlaceholder') : t('loginToAddTag')}
                className="min-h-11 min-w-0 flex-1 rounded-lg border border-white/20 bg-black/20 px-3 py-2 text-sm text-white outline-none placeholder:text-white/45 focus:border-white/50 disabled:opacity-60"
              />
              <button
                type="submit"
                disabled={isSubmittingTag || (isLoggedIn && !tagContent.trim())}
                aria-label={t('addTag')}
                title={t('addTag')}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border border-white/20 text-white transition-colors hover:bg-white/10 disabled:opacity-50"
              >
                {isSubmittingTag ? <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Plus className="h-4 w-4" aria-hidden="true" />}
              </button>
            </form>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
            <h2 className="text-sm font-semibold">{t('comments')}</h2>
            {isInteractionLoading ? (
              <div className="flex items-center gap-2 py-6 text-sm text-white/70"><LoaderCircle className="h-4 w-4 animate-spin" aria-hidden="true" />{commonT('loading')}</div>
            ) : interaction?.comments.length ? (
              <div className="mt-3 space-y-3">
                {interaction.comments.map((comment) => (
                  <article key={comment.id} className="border-b border-white/10 pb-3 last:border-0">
                    <div className="flex items-baseline justify-between gap-3">
                      <strong className="break-words text-sm">{comment.userName}</strong>
                      <time className="shrink-0 text-xs text-white/55" dateTime={comment.createdAt}>{new Intl.DateTimeFormat(locale, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(comment.createdAt))}</time>
                    </div>
                    <p className="mt-1 whitespace-pre-wrap break-words text-sm text-white/80">{comment.content}</p>
                  </article>
                ))}
              </div>
            ) : (
              <p className="py-3 text-sm text-white/60">{t('noComments')}</p>
            )}
          </div>
          <form className="border-t border-white/15 p-3" onSubmit={(event) => void submitCurrentComment(event)}>
            <label className="sr-only" htmlFor={'image-comment-' + current.id}>{t('commentPlaceholder')}</label>
            <textarea
              id={'image-comment-' + current.id}
              value={commentContent}
              onChange={(event) => setCommentContent(Array.from(event.target.value).slice(0, 500).join(''))}
              onFocus={() => {
                if (!isLoggedIn) {
                  showPortalMessage(t('loginToComment'));
                }
              }}
              rows={2}
              readOnly={!isLoggedIn}
              disabled={isSubmittingComment}
              placeholder={isLoggedIn ? t('commentPlaceholder') : t('loginToComment')}
              className="w-full resize-none rounded-lg border border-white/20 bg-black/20 px-3 py-2 text-sm text-white outline-none placeholder:text-white/45 focus:border-white/50 disabled:opacity-60"
            />
            <div className="mt-2 flex items-center justify-between gap-3">
              <span className="text-xs text-white/55">{Array.from(commentContent).length} / 500</span>
              <button type="submit" disabled={isSubmittingComment || (isLoggedIn && !commentContent.trim())} className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm font-medium text-black transition-opacity disabled:opacity-50">
                {isSubmittingComment ? <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
                {t('sendComment')}
              </button>
            </div>
            {interactionError ? <p className="mt-2 text-sm text-red-300" role="alert">{interactionError}</p> : null}
          </form>
        </aside>
      </div>

      <div className="flex flex-col items-center gap-2 px-4 pb-4">
        <div className="flex flex-wrap items-center justify-center gap-2">
          <button
            type="button"
            onClick={handleZoomOut}
            disabled={scale <= 1}
            aria-label={t('zoomOut')}
            className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10 disabled:opacity-40"
          >
            <ZoomOut className="h-5 w-5" aria-hidden="true" />
          </button>
          {scale > 1 ? (
            <button
              type="button"
              onClick={resetTransform}
              className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10"
            >
              <Minimize2 className="h-5 w-5" aria-hidden="true" />
            </button>
          ) : (
            <button
              type="button"
              onClick={() => setScale(2)}
              className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10"
            >
              <Maximize2 className="h-5 w-5" aria-hidden="true" />
            </button>
          )}
          <button
            type="button"
            onClick={handleZoomIn}
            disabled={scale >= 3}
            aria-label={t('zoomIn')}
            className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10 disabled:opacity-40"
          >
            <ZoomIn className="h-5 w-5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={() => void copyCurrentImageUrl()}
            aria-label={isCopied ? commonT('copied') : commonT('copyLink')}
            title={isCopied ? commonT('copied') : commonT('copyLink')}
            className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10"
          >
            {isCopied ? (
              <Check className="h-5 w-5" aria-hidden="true" />
            ) : (
              <Copy className="h-5 w-5" aria-hidden="true" />
            )}
          </button>
          <a
            href={current.downloadSrc}
            download
            aria-label={t('downloadImage')}
            title={t('downloadImage')}
            className="flex h-10 w-10 items-center justify-center rounded-lg text-white/90 transition-colors hover:bg-white/10"
          >
            <Download className="h-5 w-5" aria-hidden="true" />
          </a>
          <button
            type="button"
            onClick={() => void toggleCurrentLike()}
            disabled={!isLoggedIn || isTogglingLike || isInteractionLoading}
            aria-label={interaction?.likedByCurrentUser ? t('unlike') : t('like')}
            title={isLoggedIn ? interaction?.likedByCurrentUser ? t('unlike') : t('like') : t('loginToInteract')}
            className="inline-flex h-10 min-w-10 items-center justify-center gap-1 rounded-lg px-2 text-white/90 transition-colors hover:bg-white/10 disabled:opacity-50"
          >
            <Heart className={'h-5 w-5 ' + (interaction?.likedByCurrentUser ? 'fill-current text-red-400' : '')} aria-hidden="true" />
            <span className="text-xs">{interaction?.likeCount ?? current.likeCount}</span>
          </button>
        </div>
        <span className="sr-only" role="status" aria-live="polite">
          {isCopied ? commonT('copied') : ''}
        </span>
      </div>
    </div>
  );
}
