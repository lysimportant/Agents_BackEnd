'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
  // tagOverrides 保存本次浏览期间后端返回的最新标签，避免关闭预览后回退到旧列表值。
  const [tagOverrides, setTagOverrides] = useState<Record<number, string[]>>({});
  // likeCountOverrides 保存预览互动返回的最新点赞数，保证卡片与浮层显示一致。
  const [likeCountOverrides, setLikeCountOverrides] = useState<Record<number, number>>({});
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

  /** displayedImages 合并本次浏览期间更新的权威标签和点赞数量，供卡片与预览保持一致。 */
  const displayedImages = useMemo(
    () => images.map((image) => ({
      ...image,
      tags: tagOverrides[image.id] ?? image.tags,
      likeCount: likeCountOverrides[image.id] ?? image.likeCount,
    })),
    [images, likeCountOverrides, tagOverrides],
  );

  const imageColumns = useMemo(
    () => distributeMasonryImages(displayedImages, columnCount),
    [columnCount, displayedImages],
  );

  /** 更新指定图片的权威标签，并保持其他图片的本地覆盖结果。 */
  const updateImageTags = useCallback((imageID: number, tags: string[]) => {
    setTagOverrides((currentOverrides) => ({ ...currentOverrides, [imageID]: tags }));
  }, []);

  /** 更新指定图片的权威点赞数量，供卡片在预览互动后即时刷新。 */
  const updateImageLikeCount = useCallback((imageID: number, likeCount: number) => {
    setLikeCountOverrides((currentOverrides) => ({ ...currentOverrides, [imageID]: likeCount }));
  }, []);

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
          images={displayedImages}
          index={previewIndex}
          onIndexChange={setPreviewIndex}
          onTagsChange={updateImageTags}
          onLikeCountChange={updateImageLikeCount}
          onClose={() => setPreviewIndex(null)}
          returnFocusRef={triggerRef}
        />
      ) : null}
    </>
  );
}
