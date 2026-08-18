'use client';

import { useEffect } from 'react';

/** PORTAL_VIEWPORT_REVEAL_SELECTOR 覆盖门户中的瀑布流、文章、资源和分类卡片。 */
const PORTAL_VIEWPORT_REVEAL_SELECTOR = [
  '#main-content .masonry-item:not(.skeleton)',
  '#main-content article.rounded-xl',
  '#main-content a.rounded-xl.border',
  '#main-content div.rounded-xl.border',
].join(',');

/** ViewportRevealEffects 让当前和后续新增的门户卡片在首次进入视野时依次显现。 */
export function ViewportRevealEffects() {
  useEffect(() => {
    /** reducedMotion 保存用户是否要求减少动态效果。 */
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    /** trackedElements 保存已经注册的卡片，避免滚动返回时重复播放。 */
    const trackedElements = new Set<HTMLElement>();
    /** pendingFrames 保存等待执行的显现帧，组件卸载时统一取消。 */
    const pendingFrames = new Set<number>();
    /** revealObserver 在卡片接近视野时触发一次性显现。 */
    let revealObserver: IntersectionObserver | null = null;
    /** revealIndex 为同批新增卡片生成有上限的错峰延迟。 */
    let revealIndex = 0;

    /** revealElement 将卡片切换为可见状态并停止继续观察。 */
    const revealElement = (element: HTMLElement) => {
      revealObserver?.unobserve(element);
      element.dataset.viewportRevealState = 'visible';
    };

    /** scheduleReveal 让初始隐藏状态先提交一帧，再开始显现。 */
    const scheduleReveal = (element: HTMLElement) => {
      /** frameId 标识当前卡片等待执行的显现帧。 */
      const frameId = window.requestAnimationFrame(() => {
        pendingFrames.delete(frameId);
        revealElement(element);
      });
      pendingFrames.add(frameId);
    };

    /** createRevealObserver 根据动态效果偏好创建视野观察器。 */
    const createRevealObserver = () => {
      if (reducedMotion.matches || typeof IntersectionObserver === 'undefined') return null;
      return new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) scheduleReveal(entry.target as HTMLElement);
          });
        },
        { rootMargin: '0px 0px -6% 0px', threshold: 0.04 },
      );
    };

    /** observeElement 注册单张卡片，并按用户偏好决定动画或直接显示。 */
    const observeElement = (element: HTMLElement) => {
      if (trackedElements.has(element)) return;
      trackedElements.add(element);
      element.style.setProperty('--viewport-reveal-delay', `${Math.min(revealIndex % 6, 5) * 35}ms`);
      revealIndex += 1;

      if (reducedMotion.matches || !revealObserver) {
        revealElement(element);
        return;
      }
      element.dataset.viewportRevealState = 'pending';
      revealObserver.observe(element);
    };

    /** observeWithin 注册根节点自身及其内部所有匹配卡片。 */
    const observeWithin = (root: ParentNode) => {
      if (root instanceof HTMLElement && root.matches(PORTAL_VIEWPORT_REVEAL_SELECTOR)) observeElement(root);
      root.querySelectorAll<HTMLElement>(PORTAL_VIEWPORT_REVEAL_SELECTOR).forEach(observeElement);
    };

    /** removeWithin 释放已经从页面移除的卡片引用。 */
    const removeWithin = (root: Element) => {
      trackedElements.forEach((element) => {
        if (element === root || root.contains(element)) {
          revealObserver?.unobserve(element);
          trackedElements.delete(element);
        }
      });
    };

    /** syncMotionPreference 在偏好变化时立即显示所有待动画卡片。 */
    const syncMotionPreference = () => {
      if (reducedMotion.matches) {
        revealObserver?.disconnect();
        revealObserver = null;
        trackedElements.forEach(revealElement);
        return;
      }
      if (!revealObserver) revealObserver = createRevealObserver();
    };

    revealObserver = createRevealObserver();
    observeWithin(document);

    /** mutationObserver 让无限滚动追加的图片自动进入同一观察流程。 */
    const mutationObserver = new MutationObserver((records) => {
      records.forEach((record) => {
        record.addedNodes.forEach((node) => {
          if (node instanceof Element) observeWithin(node);
        });
        record.removedNodes.forEach((node) => {
          if (node instanceof Element) removeWithin(node);
        });
      });
    });
    mutationObserver.observe(document.body, { childList: true, subtree: true });
    reducedMotion.addEventListener('change', syncMotionPreference);

    return () => {
      mutationObserver.disconnect();
      revealObserver?.disconnect();
      reducedMotion.removeEventListener('change', syncMotionPreference);
      pendingFrames.forEach((frameId) => window.cancelAnimationFrame(frameId));
      trackedElements.forEach((element) => {
        delete element.dataset.viewportRevealState;
        element.style.removeProperty('--viewport-reveal-delay');
      });
    };
  }, []);

  return null;
}
