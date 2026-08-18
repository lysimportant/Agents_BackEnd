'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { EmptyState } from '@/components/common/EmptyState';
import { TiltImageCard } from './TiltImageCard';
import { ImagePreviewOverlay } from './ImagePreviewOverlay';
import type { GalleryImage } from './types';

/** MasonryColumnImage 保存图片及其在完整预览列表中的稳定索引。 */
interface MasonryColumnImage {
  image: GalleryImage;
  index: number;
}

/** 根据现有视觉断点返回瀑布流列数。 */
function resolveMasonryColumnCount(viewportWidth: number): number {
  if (viewportWidth >= 1536) {
    return 5;
  }
  if (viewportWidth >= 1280) {
    return 4;
  }
  if (viewportWidth >= 768) {
    return 3;
  }
  if (viewportWidth >= 480) {
    return 2;
  }
  return 1;
}

/**
 * 将图片按预计高度放入当前最短列。同一图片前缀每次都会得到相同列归属，
 * 追加新图片时不会移动或重新挂载已经显示的卡片。
 */
function distributeMasonryImages(
  images: GalleryImage[],
  columnCount: number,
): MasonryColumnImage[][] {
  const imageColumns = Array.from(
    { length: columnCount },
    (): MasonryColumnImage[] => [],
  );
  const estimatedColumnHeights = Array<number>(columnCount).fill(0);

  images.forEach((image, index) => {
    const shortestColumnHeight = Math.min(...estimatedColumnHeights);
    const shortestColumnIndex = estimatedColumnHeights.indexOf(shortestColumnHeight);
    const estimatedImageHeight =
      image.width > 0 && image.height > 0 ? image.height / image.width : 0.75;
    imageColumns[shortestColumnIndex].push({ image, index });
    estimatedColumnHeights[shortestColumnIndex] += estimatedImageHeight;
  });

  return imageColumns;
}

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
  const [columnCount, setColumnCount] = useState(1);
  const triggerRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    let resizeFrame = 0;
    /** updateColumnCount 合并连续 resize 事件，并保持列数断点与样式一致。 */
    const updateColumnCount = () => {
      window.cancelAnimationFrame(resizeFrame);
      resizeFrame = window.requestAnimationFrame(() => {
        setColumnCount(resolveMasonryColumnCount(window.innerWidth));
      });
    };
    const initialTimer = window.setTimeout(updateColumnCount, 0);
    window.addEventListener('resize', updateColumnCount);
    return () => {
      window.clearTimeout(initialTimer);
      window.cancelAnimationFrame(resizeFrame);
      window.removeEventListener('resize', updateColumnCount);
    };
  }, []);

  const imageColumns = useMemo(
    () => distributeMasonryImages(images, columnCount),
    [columnCount, images],
  );

  if (images.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <>
      <div className={'masonry masonry-columns-' + columnCount}>
        {imageColumns.map((columnImages, columnIndex) => (
          <div key={columnIndex} className="masonry-column">
            {columnImages.map(({ image, index }) => (
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
