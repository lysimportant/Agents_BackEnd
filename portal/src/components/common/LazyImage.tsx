'use client';

import { useEffect, useRef, useState } from 'react';
import { ImageOff } from 'lucide-react';
import { cn } from '@/utils/cn';

/**
 * LazyImage 提供模糊占位到清晰图片的渐进加载，并根据宽高比预留空间防止布局跳动。
 * 额外处理缓存图片 onLoad 不触发导致的“图片不显示”问题，并提供破图降级占位。
 */
export function LazyImage({
  src,
  alt,
  width,
  height,
  sizes,
  priority = false,
  className,
  imageClassName,
}: {
  src: string;
  alt: string;
  width?: number;
  height?: number;
  sizes?: string;
  priority?: boolean;
  className?: string;
  imageClassName?: string;
}) {
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const imageRef = useRef<HTMLImageElement>(null);

  // 缓存图片可能在 React 挂载 onLoad 之前就已加载完成，这里补一次 complete 判断。
  useEffect(() => {
    if (imageRef.current?.complete && imageRef.current.naturalWidth > 0) {
      setLoaded(true);
    }
  }, [src]);

  const aspectRatio = width && height ? width + ' / ' + height : '4 / 3';

  return (
    <div
      className={cn('relative overflow-hidden bg-muted', className)}
      style={{ aspectRatio }}
    >
      {!loaded && !failed ? (
        <div className="absolute inset-0 bg-muted" aria-hidden="true" />
      ) : null}
      {failed ? (
        <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
          <ImageOff className="h-6 w-6" aria-hidden="true" />
        </div>
      ) : (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          ref={imageRef}
          src={src}
          alt={alt}
          width={width}
          height={height}
          sizes={sizes}
          loading={priority ? 'eager' : 'lazy'}
          decoding="async"
          fetchPriority={priority ? 'high' : 'auto'}
          onLoad={() => setLoaded(true)}
          onError={() => setFailed(true)}
          className={cn(
            'h-full w-full object-cover transition-opacity duration-500',
            loaded ? 'opacity-100' : 'opacity-0',
            imageClassName,
          )}
        />
      )}
    </div>
  );
}
