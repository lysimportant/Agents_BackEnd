/**
 * 瀑布流图片卡片，按已知宽高比预留空间，加载完成后淡入图片。
 */
'use client';

import { useState } from 'react';
import { publicThumbnailUrl } from '@/services/publicApi';
import type { PublicFileListItem } from '@/types/publicContent';

/** MasonryImageCard 渲染瀑布流中的单张图片卡片。 */
export function MasonryImageCard({ image, onPreview }: { image: PublicFileListItem; onPreview?: (image: PublicFileListItem) => void }) {
  const [loaded, setLoaded] = useState(false);
  const ratio = image.imageWidth && image.imageHeight ? image.imageWidth / image.imageHeight : 4 / 3;
  return (
    <figure className="content-card fade-up break-inside-avoid overflow-hidden">
      <button
        type="button"
        className="relative block w-full"
        onClick={() => onPreview?.(image)}
        aria-label={image.altText || image.displayName}
      >
        <div style={{ aspectRatio: String(ratio) }} className="w-full overflow-hidden bg-line/30">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={publicThumbnailUrl(image.id)}
            alt={image.altText || image.displayName}
            loading="lazy"
            onLoad={() => setLoaded(true)}
            className={'h-full w-full object-cover transition-opacity duration-300 ' + (loaded ? 'opacity-100' : 'opacity-0')}
          />
        </div>
      </button>
      <figcaption className="px-3 py-2 text-sm text-ink-muted">{image.displayName}</figcaption>
    </figure>
  );
}