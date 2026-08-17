'use client';

import { useEffect, useRef, useState } from 'react';
import { ImageOff } from 'lucide-react';
import {
  galleryImageLoadQueue,
  type ImageLoadQueue,
  type ImageLoadTicket,
} from '@/utils/imageLoadQueue';
import { cn } from '@/utils/cn';

/** ProgressiveImageProps 定义渐进式图片的媒体地址、尺寸和加载策略。 */
export interface ProgressiveImageProps {
  /** src 是进入预加载范围后请求的屏幕适配图片地址。 */
  src: string;
  /** placeholderSrc 是立即用于低清占位的缩略图地址。 */
  placeholderSrc?: string;
  /** alt 是图片无法显示及辅助技术读取时使用的替代文本。 */
  alt: string;
  /** width 是原始图片宽度，用于渲染前预留比例。 */
  width?: number;
  /** height 是原始图片高度，用于渲染前预留比例。 */
  height?: number;
  /** sizes 描述图片在响应式布局中的预计展示宽度。 */
  sizes?: string;
  /** priority 让首个关键图片立即进入加载队列并提高请求优先级。 */
  priority?: boolean;
  /** className 扩展稳定比例容器的样式。 */
  className?: string;
  /** imageClassName 扩展缩略图与屏幕适配图片共用的样式。 */
  imageClassName?: string;
  /** loadQueue 允许漫画阅读器等场景传入独立并发策略。 */
  loadQueue?: ImageLoadQueue;
  /** preloadMargin 控制图片在进入视口前多远开始排队。 */
  preloadMargin?: string;
}

/**
 * ProgressiveImage 先显示缩略图，再按视口距离与共享队列渐入屏幕适配图片。
 * 两层图片共用固定宽高比，网络状态变化不会推动瀑布流重新排版。
 */
export function ProgressiveImage({
  src,
  placeholderSrc,
  alt,
  width,
  height,
  sizes,
  priority = false,
  className,
  imageClassName,
  loadQueue = galleryImageLoadQueue,
  preloadMargin = '75% 0px 150% 0px',
}: ProgressiveImageProps) {
  /** containerRef 指向 IntersectionObserver 观察的稳定比例容器。 */
  const containerRef = useRef<HTMLDivElement>(null);
  /** activeTicketRef 保存当前排队或加载中的任务票据。 */
  const activeTicketRef = useRef<ImageLoadTicket | null>(null);
  /** targetRequestControllerRef 保存当前中图请求的取消控制器。 */
  const targetRequestControllerRef = useRef<AbortController | null>(null);
  /** hasPlaceholderLoaded 表示缩略图已可见。 */
  const [hasPlaceholderLoaded, setHasPlaceholderLoaded] = useState(false);
  /** hasPlaceholderFailed 表示缩略图请求失败。 */
  const [hasPlaceholderFailed, setHasPlaceholderFailed] = useState(false);
  /** targetObjectUrl 保存受控中图请求完成后生成的本地展示地址。 */
  const [targetObjectUrl, setTargetObjectUrl] = useState('');
  /** hasTargetLoaded 表示屏幕适配图片已完成并可覆盖缩略图。 */
  const [hasTargetLoaded, setHasTargetLoaded] = useState(false);
  /** hasTargetFailed 表示屏幕适配图片请求失败。 */
  const [hasTargetFailed, setHasTargetFailed] = useState(false);
  /** initialImageSrc 保存低清占位地址；缺省时直接使用目标图片。 */
  const initialImageSrc = placeholderSrc || src;
  /** hasSeparateTarget 表示当前图片确实需要从缩略图升级到中图。 */
  const hasSeparateTarget = Boolean(src && src !== initialImageSrc);
  /** aspectRatio 在图片响应前稳定保存原始宽高比。 */
  const aspectRatio = width && height ? width + ' / ' + height : '4 / 3';

  useEffect(() => {
    if (!hasSeparateTarget || targetObjectUrl) {
      return;
    }
    /** isEffectActive 防止卸载后的异步请求写入组件状态。 */
    let isEffectActive = true;
    /** isWithinPreloadRange 记录图片当前是否仍处于观察器预加载范围。 */
    let isWithinPreloadRange = priority;
    /** observer 保存当前容器的视口距离观察器。 */
    let observer: IntersectionObserver | null = null;
    /** container 保存当前需要观察的图片容器。 */
    const container = containerRef.current;
    if (!container) {
      return;
    }
    /** queueTargetImage 将目标中图加入受限并发队列。 */
    const queueTargetImage = () => {
      if (activeTicketRef.current || targetRequestControllerRef.current) {
        return;
      }
      /** imageLoadTicket 保存本次排队任务，用于避免旧请求清理覆盖新任务。 */
      let imageLoadTicket: ImageLoadTicket | null = null;
      imageLoadTicket = loadQueue.enqueue((completeImageLoad) => {
        if (!isEffectActive) {
          completeImageLoad();
          return;
        }
        /** requestController 允许离开预加载范围时取消正在传输的中图。 */
        const requestController = new AbortController();
        targetRequestControllerRef.current = requestController;
        /** wasAborted 记录请求是否因滚动或清理主动取消。 */
        let wasAborted = false;
        void fetch(src, {
          credentials: 'include',
          signal: requestController.signal,
        })
          .then((response) => {
            if (!response.ok) {
              throw new Error('图片响应不可用');
            }
            return response.blob();
          })
          .then((imageBlob) => {
            if (!isEffectActive) {
              return;
            }
            setTargetObjectUrl(URL.createObjectURL(imageBlob));
            setHasTargetFailed(false);
          })
          .catch((requestError: unknown) => {
            wasAborted =
              requestError instanceof DOMException && requestError.name === 'AbortError';
            if (isEffectActive && !wasAborted) {
              setHasTargetFailed(true);
            }
          })
          .finally(() => {
            if (targetRequestControllerRef.current === requestController) {
              targetRequestControllerRef.current = null;
            }
            completeImageLoad();
            if (activeTicketRef.current === imageLoadTicket) {
              activeTicketRef.current = null;
            }
            if (wasAborted && isEffectActive && isWithinPreloadRange) {
              queueTargetImage();
            }
          });
      });
      activeTicketRef.current = imageLoadTicket;
    };
    if (priority || typeof IntersectionObserver === 'undefined') {
      queueTargetImage();
    } else {
      observer = new IntersectionObserver(
        (entries) => {
          /** observedEntry 保存当前组件唯一容器的相交状态。 */
          const observedEntry = entries[0];
          isWithinPreloadRange = Boolean(observedEntry?.isIntersecting);
          if (isWithinPreloadRange) {
            queueTargetImage();
          } else if (activeTicketRef.current) {
            targetRequestControllerRef.current?.abort();
            activeTicketRef.current.cancel();
            activeTicketRef.current = null;
          }
        },
        { rootMargin: preloadMargin },
      );
      observer.observe(container);
    }
    return () => {
      isEffectActive = false;
      observer?.disconnect();
      targetRequestControllerRef.current?.abort();
      targetRequestControllerRef.current = null;
      activeTicketRef.current?.cancel();
      activeTicketRef.current = null;
    };
  }, [hasSeparateTarget, loadQueue, preloadMargin, priority, src, targetObjectUrl]);

  useEffect(
    () => () => {
      if (targetObjectUrl) {
        URL.revokeObjectURL(targetObjectUrl);
      }
    },
    [targetObjectUrl],
  );

  /** shouldShowFailure 仅在缩略图和中图均失败时显示破图状态。 */
  const shouldShowFailure =
    hasPlaceholderFailed && (!hasSeparateTarget || hasTargetFailed);
  /** shouldShowPlaceholderSkeleton 在任一图片可见前保持安静的稳定占位。 */
  const shouldShowPlaceholderSkeleton =
    !hasPlaceholderLoaded && !hasTargetLoaded && !shouldShowFailure;

  return (
    <div
      ref={containerRef}
      className={cn('relative overflow-hidden bg-muted', className)}
      style={{ aspectRatio }}
    >
      {shouldShowPlaceholderSkeleton ? (
        <div className="absolute inset-0 bg-muted" aria-hidden="true" />
      ) : null}
      {shouldShowFailure ? (
        <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
          <ImageOff className="h-6 w-6" aria-hidden="true" />
        </div>
      ) : (
        <>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={initialImageSrc}
            alt={hasSeparateTarget ? '' : alt}
            width={width}
            height={height}
            sizes={sizes}
            loading={priority ? 'eager' : 'lazy'}
            decoding="async"
            fetchPriority={priority ? 'high' : 'auto'}
            onLoad={() => setHasPlaceholderLoaded(true)}
            onError={() => setHasPlaceholderFailed(true)}
            className={cn(
              'absolute inset-0 h-full w-full object-cover transition-opacity duration-300',
              hasPlaceholderLoaded && !hasTargetLoaded ? 'opacity-100' : 'opacity-0',
              imageClassName,
            )}
          />
          {hasSeparateTarget && targetObjectUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={targetObjectUrl}
              alt={alt}
              width={width}
              height={height}
              sizes={sizes}
              loading="eager"
              decoding="async"
              fetchPriority={priority ? 'high' : 'auto'}
              onLoad={() => setHasTargetLoaded(true)}
              onError={() => setHasTargetFailed(true)}
              className={cn(
                'absolute inset-0 h-full w-full object-cover transition-opacity duration-500',
                hasTargetLoaded ? 'opacity-100' : 'opacity-0',
                imageClassName,
              )}
            />
          ) : null}
        </>
      )}
    </div>
  );
}
