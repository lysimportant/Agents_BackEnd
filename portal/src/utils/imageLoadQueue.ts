/** 图片加载任务开始时收到的完成回调，用于释放一个并发槽位。 */
export type CompleteImageLoad = () => void;

/** 图片加载任务在获得并发槽位后执行的启动函数。 */
export type StartImageLoad = (completeImageLoad: CompleteImageLoad) => void;

/** ImageLoadTicket 允许组件在任务排队或加载期间取消当前任务。 */
export interface ImageLoadTicket {
  /** cancel 取消排队任务；已开始的任务则立即释放并发槽位。 */
  cancel: () => void;
}

/** PendingImageLoad 保存队列内部单个图片任务的生命周期。 */
interface PendingImageLoad {
  /** startImageLoad 在任务获得并发槽位后启动实际图片请求。 */
  startImageLoad: StartImageLoad;
  /** isActive 表示任务是否已经占用并发槽位。 */
  isActive: boolean;
  /** isCancelled 表示任务是否已被调用方取消。 */
  isCancelled: boolean;
}

/**
 * ImageLoadQueue 按先进先出顺序限制图片请求并发数。
 * 漫画阅读器等未来场景可以创建独立实例并提供自己的并发策略。
 */
export class ImageLoadQueue {
  /** pendingImageLoads 保存尚未获得并发槽位的图片任务。 */
  private pendingImageLoads: PendingImageLoad[] = [];
  /** activeImageLoadCount 保存当前占用的并发槽位数量。 */
  private activeImageLoadCount = 0;
  /** isDrainScheduled 避免同一轮渲染重复安排队列启动。 */
  private isDrainScheduled = false;

  /**
   * @param resolveConcurrencyLimit 根据当前设备状态返回允许的并发图片数。
   */
  constructor(private readonly resolveConcurrencyLimit: () => number) {}

  /** enqueue 将图片任务加入队列，并返回可取消的任务票据。 */
  enqueue(startImageLoad: StartImageLoad): ImageLoadTicket {
    /** pendingImageLoad 保存本次加入队列的图片任务。 */
    const pendingImageLoad: PendingImageLoad = {
      startImageLoad,
      isActive: false,
      isCancelled: false,
    };
    this.pendingImageLoads.push(pendingImageLoad);
    this.scheduleDrain();

    return {
      cancel: () => {
        if (pendingImageLoad.isCancelled) {
          return;
        }
        pendingImageLoad.isCancelled = true;
        if (pendingImageLoad.isActive) {
          this.complete(pendingImageLoad);
        } else {
          this.pendingImageLoads = this.pendingImageLoads.filter(
            (queuedImageLoad) => queuedImageLoad !== pendingImageLoad,
          );
        }
      },
    };
  }

  /** complete 释放已完成或取消任务占用的并发槽位。 */
  private complete(pendingImageLoad: PendingImageLoad): void {
    if (!pendingImageLoad.isActive) {
      return;
    }
    pendingImageLoad.isActive = false;
    this.activeImageLoadCount = Math.max(0, this.activeImageLoadCount - 1);
    this.scheduleDrain();
  }

  /** scheduleDrain 延后一微任务启动请求，让 Strict Mode 清理先取消试运行任务。 */
  private scheduleDrain(): void {
    if (this.isDrainScheduled) {
      return;
    }
    this.isDrainScheduled = true;
    queueMicrotask(() => {
      this.isDrainScheduled = false;
      this.drain();
    });
  }

  /** drain 在并发上限内依次启动等待中的图片任务。 */
  private drain(): void {
    /** concurrencyLimit 保存当前设备允许的最小为 1 的并发数。 */
    const concurrencyLimit = Math.max(1, this.resolveConcurrencyLimit());
    while (
      this.activeImageLoadCount < concurrencyLimit &&
      this.pendingImageLoads.length > 0
    ) {
      /** pendingImageLoad 保存队首等待任务。 */
      const pendingImageLoad = this.pendingImageLoads.shift();
      if (!pendingImageLoad || pendingImageLoad.isCancelled) {
        continue;
      }
      pendingImageLoad.isActive = true;
      this.activeImageLoadCount += 1;
      try {
        pendingImageLoad.startImageLoad(() => this.complete(pendingImageLoad));
      } catch {
        this.complete(pendingImageLoad);
      }
    }
  }
}

/** resolveGalleryConcurrencyLimit 为移动端限制 2 张、较宽视口限制 4 张并发中图。 */
function resolveGalleryConcurrencyLimit(): number {
  if (typeof window === 'undefined') {
    return 2;
  }
  return window.matchMedia('(max-width: 767px)').matches ? 2 : 4;
}

/** galleryImageLoadQueue 是瀑布流共享的屏幕适配图片加载队列。 */
export const galleryImageLoadQueue = new ImageLoadQueue(resolveGalleryConcurrencyLimit);
