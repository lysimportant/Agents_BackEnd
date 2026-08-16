'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight, Maximize2, Minimize2, X, ZoomIn, ZoomOut } from 'lucide-react';
import type { GalleryImage } from './types';

/** 图片预览浮层，支持左右切换、缩放、拖动、Esc 关闭与移动端手势。 */
export function ImagePreviewOverlay({
  images,
  index,
  onIndexChange,
  onClose,
  returnFocusRef,
}: {
  images: GalleryImage[];
  index: number;
  onIndexChange: (index: number) => void;
  onClose: () => void;
  returnFocusRef: React.RefObject<HTMLElement | null>;
}) {
  const t = useTranslations('images');
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const dragStart = useRef<{ x: number; y: number; startX: number; startY: number; scaled: boolean } | null>(null);

  const current = images[index];
  const hasPrevious = index > 0;
  const hasNext = index < images.length - 1;

  const goPrevious = useCallback(() => {
    if (hasPrevious) {
      onIndexChange(index - 1);
    }
  }, [hasPrevious, index, onIndexChange]);

  const goNext = useCallback(() => {
    if (hasNext) {
      onIndexChange(index + 1);
    }
  }, [hasNext, index, onIndexChange]);

  const resetTransform = useCallback(() => {
    setScale(1);
    setOffset({ x: 0, y: 0 });
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

      <div className="relative flex min-h-0 flex-1 items-center justify-center px-4 pb-4">
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

      <div className="flex flex-col items-center gap-3 px-4 pb-6">
        <div className="max-w-2xl text-center">
          <p className="truncate text-sm font-medium text-white">{current.displayName}</p>
          {current.category ? (
            <p className="mt-1 text-xs text-white/70">{current.category}</p>
          ) : null}
        </div>
        <div className="flex items-center gap-2">
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
        </div>
      </div>
    </div>
  );
}
