/**
 * 图片预览浮层，支持前后切换、缩放、拖拽，并按 Esc 关闭。
 */
'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { X, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, RotateCcw } from 'lucide-react';
import { publicPreviewUrl } from '@/services/publicApi';
import type { PublicFileListItem } from '@/types/publicContent';

/** ImagePreviewModal 渲染图片查看浮层。 */
export function ImagePreviewModal({
  images,
  initial,
  onClose,
}: {
  images: PublicFileListItem[];
  initial: PublicFileListItem;
  onClose: () => void;
}) {
  const t = useTranslations('image');
  const [index, setIndex] = useState(() => Math.max(0, images.findIndex((img) => img.id === initial.id)));
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const dragRef = useRef<{ startX: number; startY: number; ox: number; oy: number } | null>(null);
  const current = images[index];

  // 监听键盘事件：Esc 关闭、方向键切换、+/- 缩放。
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
      if (e.key === 'ArrowLeft') setIndex((i) => Math.max(0, i - 1));
      if (e.key === 'ArrowRight') setIndex((i) => Math.min(images.length - 1, i + 1));
      if (e.key === '+' || e.key === '=') setScale((s) => Math.min(3, s + 0.25));
      if (e.key === '-') setScale((s) => Math.max(1, s - 0.25));
    }
    window.addEventListener('keydown', onKey);
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  }, [onClose, images.length]);

  // 切换到指定图片并重置缩放与偏移。
  const go = useCallback((next: number) => {
    setIndex(next);
    setScale(1);
    setOffset({ x: 0, y: 0 });
  }, []);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('viewLarge')}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-lg"
      onClick={onClose}
    >
      <button
        type="button"
        aria-label={t('previewClose')}
        onClick={onClose}
        className="absolute right-4 top-4 tap-target flex items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
      >
        <X className="h-6 w-6" aria-hidden="true" />
      </button>

      {index > 0 && (
        <button
          type="button"
          aria-label={t('previewPrev')}
          onClick={(e) => { e.stopPropagation(); go(index - 1); }}
          className="absolute left-3 top-1/2 -translate-y-1/2 tap-target flex items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
        >
          <ChevronLeft className="h-6 w-6" aria-hidden="true" />
        </button>
      )}
      {index < images.length - 1 && (
        <button
          type="button"
          aria-label={t('previewNext')}
          onClick={(e) => { e.stopPropagation(); go(index + 1); }}
          className="absolute right-3 top-1/2 -translate-y-1/2 tap-target flex items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
        >
          <ChevronRight className="h-6 w-6" aria-hidden="true" />
        </button>
      )}

      <div className="flex max-h-[90vh] max-w-[90vw] flex-col items-center gap-3" onClick={(e) => e.stopPropagation()}>
        <div
          className="cursor-grab touch-none select-none overflow-hidden rounded-lg"
          onPointerDown={(e) => {
            dragRef.current = { startX: e.clientX, startY: e.clientY, ox: offset.x, oy: offset.y };
            (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
          }}
          onPointerMove={(e) => {
            if (!dragRef.current) return;
            setOffset({
              x: dragRef.current.ox + (e.clientX - dragRef.current.startX),
              y: dragRef.current.oy + (e.clientY - dragRef.current.startY),
            });
          }}
          onPointerUp={() => { dragRef.current = null; }}
          style={{ transform: 'translate(' + offset.x + 'px,' + offset.y + 'px) scale(' + scale + ')', transition: 'transform 200ms ease-out' }}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={publicPreviewUrl(current.id)} alt={current.altText || current.displayName} className="max-h-[80vh] max-w-[85vw] object-contain" />
        </div>
        <div className="flex items-center gap-2 text-white">
          <button type="button" aria-label={t('previewZoomOut')} onClick={() => setScale((s) => Math.max(1, s - 0.25))} className="tap-target flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20">
            <ZoomOut className="h-5 w-5" aria-hidden="true" />
          </button>
          <button type="button" aria-label={t('previewReset')} onClick={() => { setScale(1); setOffset({ x: 0, y: 0 }); }} className="tap-target flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20">
            <RotateCcw className="h-5 w-5" aria-hidden="true" />
          </button>
          <button type="button" aria-label={t('previewZoomIn')} onClick={() => setScale((s) => Math.min(3, s + 0.25))} className="tap-target flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20">
            <ZoomIn className="h-5 w-5" aria-hidden="true" />
          </button>
          <span className="ml-2 text-sm">{index + 1} / {images.length}</span>
        </div>
      </div>
    </div>
  );
}