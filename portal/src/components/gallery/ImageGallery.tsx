'use client';

import { useRef, useState } from 'react';
import { EmptyState } from '@/components/common/EmptyState';
import { TiltImageCard } from './TiltImageCard';
import { ImagePreviewOverlay } from './ImagePreviewOverlay';
import type { GalleryImage } from './types';

/**
 * ImageGallery 渲染响应式图片瀑布流，卡片用缩略图、预览用原图，点击打开预览浮层。
 */
export function ImageGallery({
  images,
  emptyTitle,
  emptyDescription,
}: {
  images: GalleryImage[];
  emptyTitle: string;
  emptyDescription: string;
}) {
  const [previewIndex, setPreviewIndex] = useState<number | null>(null);
  const triggerRef = useRef<HTMLElement | null>(null);

  if (images.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <>
      <div className="masonry">
        {images.map((image, index) => (
          <TiltImageCard
            key={image.id}
            image={image}
            index={index}
            onOpen={(nextIndex, trigger) => {
              triggerRef.current = trigger;
              setPreviewIndex(nextIndex);
            }}
          />
        ))}
      </div>

      {previewIndex !== null ? (
        <ImagePreviewOverlay
          images={images}
          index={previewIndex}
          onIndexChange={setPreviewIndex}
          onClose={() => setPreviewIndex(null)}
          returnFocusRef={triggerRef}
        />
      ) : null}
    </>
  );
}
