'use client';

import { useLayoutEffect } from 'react';
import { DEFAULT_TILT_CARD_SELECTOR } from './TiltCardEffects';

/** DEFAULT_VIEWPORT_REVEAL_SELECTOR 覆盖页面区块、动态列表项和现有业务卡片。 */
export const DEFAULT_VIEWPORT_REVEAL_SELECTOR = [
  DEFAULT_TILT_CARD_SELECTOR,
  '.antd-content-view > :is(.page-stack, .dashboard-page) > *',
  '.auth-shell > *',
  '.antd-content-view tbody tr',
  '.antd-content-view .socket-conversation-item',
  '.antd-content-view .socket-message-row',
].join(',');

type ViewportRevealEffectsProps = {
  /** selector 指定需要在进入视野时显现的元素。 */
  selector?: string;
};

/** ViewportRevealEffects 统一观察当前与后续新增元素，并让每个元素只播放一次进入动画。 */
export function ViewportRevealEffects({ selector = DEFAULT_VIEWPORT_REVEAL_SELECTOR }: ViewportRevealEffectsProps) {
  useLayoutEffect(() => {
    /** reducedMotion 保存用户是否要求减少动态效果。 */
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    /** trackedElements 保存已注册元素，防止重复观察和重复播放。 */
    const trackedElements = new Set<HTMLElement>();
    /** pendingFrames 保存等待显现的动画帧，卸载时统一取消。 */
    const pendingFrames = new Set<number>();
    /** revealObserver 在元素接近视野时触发一次性显现。 */
    let revealObserver: IntersectionObserver | null = null;
    /** revealIndex 为同批元素提供有上限的错峰延迟。 */
    let revealIndex = 0;

    /** revealElement 将元素切换为可见状态并停止继续观察。 */
    const revealElement = (element: HTMLElement) => {
      revealObserver?.unobserve(element);
      element.dataset.viewportRevealState = 'visible';
    };

    /** scheduleReveal 确保浏览器先提交初始状态，再播放显现动画。 */
    const scheduleReveal = (element: HTMLElement) => {
      const frameId = window.requestAnimationFrame(() => {
        pendingFrames.delete(frameId);
        revealElement(element);
      });
      pendingFrames.add(frameId);
    };

    /** createRevealObserver 根据当前动态效果偏好创建视野观察器。 */
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

    /** observeElement 注册单个元素，并按用户偏好决定动画或直接显示。 */
    const observeElement = (element: HTMLElement) => {
      if (element.dataset.viewportRevealDisabled === 'true' || trackedElements.has(element)) return;
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

    /** observeWithin 注册根节点自身及其内部所有匹配元素。 */
    const observeWithin = (root: ParentNode) => {
      if (root instanceof HTMLElement && root.matches(selector)) observeElement(root);
      root.querySelectorAll<HTMLElement>(selector).forEach(observeElement);
    };

    /** removeWithin 释放已经从页面移除的元素引用。 */
    const removeWithin = (root: Element) => {
      trackedElements.forEach((element) => {
        if (element === root || root.contains(element)) {
          revealObserver?.unobserve(element);
          trackedElements.delete(element);
        }
      });
    };

    /** syncMotionPreference 在偏好变化时立即显示待动画元素。 */
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

    /** mutationObserver 让异步加载、翻页和新增列表项自动获得相同动画。 */
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
  }, [selector]);

  return null;
}
