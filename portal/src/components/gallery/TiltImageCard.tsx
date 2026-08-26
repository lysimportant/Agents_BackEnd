'use client';

import { Eye, UserRound } from 'lucide-react';
import { useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ProgressiveImage } from '@/components/common/ProgressiveImage';
import type { GalleryImage } from './types';

/**
 * TiltImageCard 是瀑布流中的单张图片卡片，显示公开标签、上传人和浏览量。
 */
export function TiltImageCard({
  image,
  index,
  onOpen,
}: {
  image: GalleryImage;
  index: number;
  onOpen: (index: number, trigger: HTMLElement) => void;
}) {
  const t = useTranslations('a11y');
  const cardRef = useRef<HTMLButtonElement>(null);
  const [transform, setTransform] = useState('');
  // 读取用户是否偏好减少动态效果；仅浏览器环境可读，SSR 时回退 false。
  const [reducedMotion] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  );

  /** 根据光标在图片内的位置计算轻微 3D 摆动角度。 */
  const handlePointerMove = (event: React.PointerEvent) => {
    if (reducedMotion) {
      return;
    }
    const card = cardRef.current;
    if (!card) {
      return;
    }
    const rect = card.getBoundingClientRect();
    const x = (event.clientX - rect.left) / rect.width;
    const y = (event.clientY - rect.top) / rect.height;
    const rotateY = (x - 0.5) * 8;
    const rotateX = (0.5 - y) * 8;
    setTransform(
      'perspective(800px) rotateX(' +
        rotateX.toFixed(2) +
        'deg) rotateY(' +
        rotateY.toFixed(2) +
        'deg)',
    );
  };

  return (
    <button
      ref={cardRef}
      type="button"
      onClick={(event) => onOpen(index, event.currentTarget)}
      onPointerMove={handlePointerMove}
      onPointerLeave={() => setTransform('')}
      aria-label={t('openImage') + ': ' + image.alt}
      className="masonry-item tilt-image-card group relative block w-full overflow-hidden rounded-lg border border-border bg-surface text-left shadow-sm"
      style={{
        transform,
      }}
    >
      <ProgressiveImage
        key={image.displaySrc ?? image.thumbnailSrc ?? image.src}
        src={image.displaySrc ?? image.thumbnailSrc ?? image.src}
        placeholderSrc={image.thumbnailSrc}
        alt={image.alt}
        width={image.width}
        height={image.height}
        sizes="(min-width: 1280px) 25vw, (min-width: 768px) 33vw, (min-width: 480px) 50vw, 100vw"
        priority={index === 0}
        imageClassName="transition-transform duration-300 group-hover:scale-[1.03]"
      />
      <div className="space-y-2 border-t border-border px-3 py-3">
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span className="flex min-w-0 items-center gap-1 truncate" title={image.ownerName || undefined}>
            <UserRound className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate">{image.ownerName || '—'}</span>
          </span>
          <span className="flex shrink-0 items-center gap-1" title={t('views')}>
            <Eye className="h-3.5 w-3.5" aria-hidden="true" />
            {image.views}
          </span>
        </div>
        {image.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1" aria-label={t('tags')}>
            {image.tags.slice(0, 8).map((tag) => (
              <span key={tag} className="max-w-full truncate rounded-md bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">
                #{tag}
              </span>
            ))}
          </div>
        ) : null}
      </div>
    </button>
  );
}
