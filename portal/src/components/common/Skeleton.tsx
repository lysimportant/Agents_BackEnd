import { cn } from '@/utils/cn';

/** Skeleton 渲染稳定尺寸的骨架占位，避免加载状态引发布局跳动。 */
export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('skeleton', className)} aria-hidden="true" />;
}
