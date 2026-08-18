'use client';

import { useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ProgressiveImage } from '@/components/common/ProgressiveImage';
import type { GalleryImage } from './types';

/**
 * TiltImageCard 是瀑布流中的单张裸图瓦片：
 * 不使用卡片边框与说明条，仅保留圆角、缩略图懒加载与悬停轻微 3D 摆动和缩放。
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
      className="masonry-item tilt-image-card group relative block w-full overflow-hidden rounded-lg text-left"
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
    </button>
  );
}
