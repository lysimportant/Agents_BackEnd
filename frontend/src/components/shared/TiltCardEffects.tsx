'use client';

import { useEffect } from 'react';
import { DEFAULT_TILT_CARD_OPTIONS } from './TiltCard';

/** DEFAULT_TILT_CARD_SELECTOR 保存模块使用的固定配置或共享状态。 */
export const DEFAULT_TILT_CARD_SELECTOR = [
  '[data-tilt-card="true"]:not([data-tilt-disabled="true"])',
  '.antd-content-view .section-header-card',
  '.antd-content-view .welcome-card',
  '.antd-content-view .panel-card',
  '.antd-content-view .stat-card',
  '.antd-content-view .dashboard-stat-card',
  '.antd-content-view .dashboard-panel',
  '.antd-content-view .menu-summary-card',
  '.antd-content-view .file-kind-card',
  '.antd-content-view .article-library-card',
  '.antd-content-view .ant-card:not(.file-browser-panel):not([data-tilt-disabled="true"])',
  '.antd-content-view [data-slot="card"]',
].join(',');

type TiltCardEffectsProps = {
  /** selector 表示变量 selector。 */
  selector?: string;
};

type MotionValues = {
  /** rotateX 表示变量 rotateX。 */
  rotateX: number;
  /** rotateY 表示变量 rotateY。 */
  rotateY: number;
  /** scale 表示缩放比例。 */
  scale: number;
  /** lift 表示变量 lift。 */
  lift: number;
  /** glareX 表示变量 glareX。 */
  glareX: number;
  /** glareY 表示变量 glareY。 */
  glareY: number;
  /** glareOpacity 表示变量 glareOpacity。 */
  glareOpacity: number;
  /** holographicOpacity 表示变量 holographicOpacity。 */
  holographicOpacity: number;
  /** parallaxX 表示变量 parallaxX。 */
  parallaxX: number;
  /** parallaxY 表示变量 parallaxY。 */
  parallaxY: number;
};

type RuntimeOptions = {
  /** maxTilt 表示变量 maxTilt。 */
  maxTilt: number;
  /** scale 表示缩放比例。 */
  scale: number;
  /** lift 表示变量 lift。 */
  lift: number;
  /** smoothing 表示变量 smoothing。 */
  smoothing: number;
  /** glareStrength 表示变量 glareStrength。 */
  glareStrength: number;
  /** parallax 表示变量 parallax。 */
  parallax: number;
  /** perspective 表示变量 perspective。 */
  perspective: number;
};

type CardMotion = {
  /** card 表示卡片元素。 */
  card: HTMLElement;
  /** current 表示当前。 */
  current: MotionValues;
  /** target 表示目标。 */
  target: MotionValues;
  /** options 表示选项。 */
  options: RuntimeOptions;
  /** rect 表示元素边界。 */
  rect: DOMRect | null;
  /** active 表示当前激活。 */
  active: boolean;
};

/** RESTING_VALUES 保存模块使用的固定配置或共享状态。 */
const RESTING_VALUES: MotionValues = {
  rotateX: 0,
  rotateY: 0,
  scale: 1,
  lift: 0,
  glareX: 50,
  glareY: 50,
  glareOpacity: 0,
  holographicOpacity: 0,
  parallaxX: 0,
  parallaxY: 0,
};

/** MOTION_EPSILON 保存模块使用的固定配置或共享状态。 */
const MOTION_EPSILON = 0.01;

/**
 * 通过事件委托增强当前和后续新增的业务卡片。整个应用共用一个动画循环，
 * pointermove 仅写入目标值；当前卡片或视口变化前持续复用 DOM 几何缓存。
 */
export function TiltCardEffects({ selector = DEFAULT_TILT_CARD_SELECTOR }: TiltCardEffectsProps) {
  useEffect(() => {
    /** finePointer 保存变量 finePointer。 */
    const finePointer = window.matchMedia('(hover: hover) and (pointer: fine)');
    /** reducedMotion 保存动画状态。 */
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    /** trackedCards 保存卡片元素。 */
    const trackedCards = new Set<HTMLElement>();
    /** autoEnhancedCards 保存卡片元素。 */
    const autoEnhancedCards = new Set<HTMLElement>();
    /** motions 保存动画状态。 */
    const motions = new Map<HTMLElement, CardMotion>();

    /** enabled 保存已启用。 */
    let enabled = finePointer.matches && !reducedMotion.matches;
    /** activeCard、HTMLElement、null 保存当前激活、变量 HTMLElement、空值标记。 */
    let activeCard: HTMLElement | null = null;
    /** frameId 保存标识。 */
    let frameId = 0;
    /** lastFrameTime 保存时间。 */
    let lastFrameTime = performance.now();

    /** enhanceCard 负责计算或维护卡片元素。 */
    const enhanceCard = (card: HTMLElement) => {
      if (card.dataset.tiltDisabled === 'true' || trackedCards.has(card)) return;
      trackedCards.add(card);
      card.dataset.tiltReady = 'true';
      if (!card.classList.contains('tilt-card-surface')) {
        autoEnhancedCards.add(card);
        card.classList.add('tilt-card-surface');
      }
    };

    /** enhanceWithin 负责计算或维护变量 enhanceWithin。 */
    const enhanceWithin = (root: ParentNode) => {
      if (root instanceof HTMLElement && root.matches(selector)) enhanceCard(root);
      root.querySelectorAll<HTMLElement>(selector).forEach(enhanceCard);
    };

    /** resetCard 负责计算或维护卡片元素。 */
    const resetCard = (card: HTMLElement) => {
      applyValues(card, RESTING_VALUES);
      delete card.dataset.tiltActive;
      card.style.removeProperty('--tilt-perspective');
    };

    /** stopMotion 负责计算或维护动画状态。 */
    const stopMotion = () => {
      if (frameId) cancelAnimationFrame(frameId);
      frameId = 0;
      activeCard = null;
      motions.forEach(({ card }) => resetCard(card));
      motions.clear();
    };

    /** syncCapability 负责更新并保存对应业务状态。 */
    const syncCapability = () => {
      enabled = finePointer.matches && !reducedMotion.matches;
      if (enabled) enhanceWithin(document);
      else stopMotion();
    };

    /** getMotion 负责读取并返回对应业务数据。 */
    const getMotion = (card: HTMLElement) => {
      /** existing 保存已有记录。 */
      const existing = motions.get(card);
      if (existing) return existing;

      /** motion、CardMotion 保存动画状态、卡片元素动画状态。 */
      const motion: CardMotion = {
        card,
        current: { ...RESTING_VALUES },
        target: { ...RESTING_VALUES },
        options: readOptions(card),
        rect: null,
        active: false,
      };
      motions.set(card, motion);
      applyValues(card, motion.current);
      return motion;
    };

    /** settle 负责更新并保存对应业务状态。 */
    const settle = (card: HTMLElement | null) => {
      if (!card) return;
      /** motion 保存动画状态。 */
      const motion = getMotion(card);
      motion.active = false;
      motion.target = { ...RESTING_VALUES };
      scheduleFrame();
    };

    /** activate 负责计算或维护变量 activate。 */
    const activate = (card: HTMLElement) => {
      if (activeCard !== card) {
        settle(activeCard);
        activeCard = card;
      }

      /** motion 保存动画状态。 */
      const motion = getMotion(card);
      motion.active = true;
      motion.options = readOptions(card);
      motion.rect = card.getBoundingClientRect();
      card.dataset.tiltActive = 'true';
      card.style.setProperty('--tilt-perspective', `${motion.options.perspective}px`);
      return motion;
    };

    /** releaseActiveCard 负责计算或维护当前激活。 */
    const releaseActiveCard = () => {
      settle(activeCard);
      activeCard = null;
    };

    /** findCard 负责读取并返回对应业务数据。 */
    const findCard = (target: EventTarget | null) => {
      if (!(target instanceof Element)) return null;
      /** card 保存卡片元素。 */
      const card = target.closest<HTMLElement>(selector);
      if (!card || card.dataset.tiltDisabled === 'true') return null;

      /** disabledRegion 保存变量 disabledRegion。 */
      const disabledRegion = target.closest<HTMLElement>('[data-tilt-disabled="true"]');
      if (disabledRegion && card.contains(disabledRegion)) return null;
      return card;
    };

    /** onPointerMove 负责处理对应的界面事件和状态变化。 */
    const onPointerMove = (event: PointerEvent) => {
      if (!enabled || event.pointerType === 'touch') return;
      /** card 保存卡片元素。 */
      const card = findCard(event.target);
      if (!card) {
        releaseActiveCard();
        return;
      }

      /** motion 保存动画状态。 */
      const motion = activeCard === card ? getMotion(card) : activate(card);
      /** rect 保存元素边界。 */
      const rect = motion.rect ?? card.getBoundingClientRect();
      motion.rect = rect;
      if (rect.width <= 0 || rect.height <= 0) return;

      /** normalizedX 保存变量 normalizedX。 */
      const normalizedX = clamp(((event.clientX - rect.left) / rect.width) * 2 - 1, -1, 1);
      /** normalizedY 保存变量 normalizedY。 */
      const normalizedY = clamp(((event.clientY - rect.top) / rect.height) * 2 - 1, -1, 1);
      /** largeSurfaceFactor 保存变量 largeSurfaceFactor。 */
      const largeSurfaceFactor = rect.width > 760 || rect.height > 520 ? 0.56 : 1;
      /** scaleFactor 保存缩放比例。 */
      const scaleFactor = rect.width > 760 || rect.height > 520 ? 0.44 : 1;
      /** parallaxFactor 保存变量 parallaxFactor。 */
      const parallaxFactor = rect.width > 760 || rect.height > 520 ? 0.62 : 1;

      motion.target = {
        rotateX: -normalizedY * motion.options.maxTilt * largeSurfaceFactor,
        rotateY: normalizedX * motion.options.maxTilt * largeSurfaceFactor,
        scale: 1 + (motion.options.scale - 1) * scaleFactor,
        lift: -motion.options.lift * (largeSurfaceFactor === 1 ? 1 : 0.7),
        glareX: (normalizedX + 1) * 50,
        glareY: (normalizedY + 1) * 50,
        glareOpacity: motion.options.glareStrength,
        holographicOpacity: motion.options.glareStrength * 0.82,
        parallaxX: normalizedX * motion.options.parallax * parallaxFactor,
        parallaxY: normalizedY * motion.options.parallax * parallaxFactor,
      };
      scheduleFrame();
    };

    /** onPointerOut 负责处理对应的界面事件和状态变化。 */
    const onPointerOut = (event: PointerEvent) => {
      if (event.relatedTarget === null) releaseActiveCard();
    };

    /** invalidateGeometry 负责计算或维护变量 invalidateGeometry。 */
    const invalidateGeometry = () => {
      if (activeCard) getMotion(activeCard).rect = null;
    };

    /** onVisibilityChange 负责处理对应的界面事件和状态变化。 */
    const onVisibilityChange = () => {
      if (document.hidden) releaseActiveCard();
    };

    /** observer 负责计算或维护变量 observer。 */
    const observer = new MutationObserver((records) => {
      /** record 表示当前循环中的索引、键或业务元素。 */
      for (const record of records) {
        record.addedNodes.forEach((node) => {
          if (node instanceof Element) enhanceWithin(node);
        });
        record.removedNodes.forEach((node) => {
          if (!(node instanceof Element)) return;
          trackedCards.forEach((card) => {
            if (card === node || node.contains(card)) {
              motions.delete(card);
              trackedCards.delete(card);
              autoEnhancedCards.delete(card);
              if (activeCard === card) activeCard = null;
            }
          });
        });
      }
    });

    function scheduleFrame() {
      if (frameId) return;
      lastFrameTime = performance.now();
      frameId = requestAnimationFrame(animate);
    }

    function animate(now: number) {
      frameId = 0;
      /** elapsedFrames 保存变量 elapsedFrames。 */
      const elapsedFrames = Math.min(2, Math.max(0.5, (now - lastFrameTime) / 16.667));
      lastFrameTime = now;
      /** needsAnotherFrame 保存变量 needsAnotherFrame。 */
      let needsAnotherFrame = false;

      motions.forEach((motion, card) => {
        /** alpha 保存变量 alpha。 */
        const alpha = 1 - Math.pow(1 - motion.options.smoothing, elapsedFrames);
        motion.current = interpolateValues(motion.current, motion.target, alpha);
        applyValues(card, motion.current);

        if (!isSettled(motion.current, motion.target)) {
          needsAnotherFrame = true;
          return;
        }

        motion.current = { ...motion.target };
        applyValues(card, motion.current);
        if (!motion.active) {
          delete card.dataset.tiltActive;
          motions.delete(card);
        }
      });

      if (needsAnotherFrame) frameId = requestAnimationFrame(animate);
    }

    enhanceWithin(document);
    observer.observe(document.body, { childList: true, subtree: true });
    document.addEventListener('pointermove', onPointerMove, { passive: true });
    document.addEventListener('pointerout', onPointerOut, { passive: true });
    document.addEventListener('visibilitychange', onVisibilityChange);
    window.addEventListener('blur', releaseActiveCard);
    window.addEventListener('resize', invalidateGeometry, { passive: true });
    window.addEventListener('scroll', invalidateGeometry, { passive: true, capture: true });
    finePointer.addEventListener('change', syncCapability);
    reducedMotion.addEventListener('change', syncCapability);
    syncCapability();

    return () => {
      observer.disconnect();
      stopMotion();
      document.removeEventListener('pointermove', onPointerMove);
      document.removeEventListener('pointerout', onPointerOut);
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.removeEventListener('blur', releaseActiveCard);
      window.removeEventListener('resize', invalidateGeometry);
      window.removeEventListener('scroll', invalidateGeometry, true);
      finePointer.removeEventListener('change', syncCapability);
      reducedMotion.removeEventListener('change', syncCapability);
      trackedCards.forEach((card) => delete card.dataset.tiltReady);
      autoEnhancedCards.forEach((card) => card.classList.remove('tilt-card-surface'));
    };
  }, [selector]);

  return null;
}

/** readOptions 加载对应业务数据。 */
function readOptions(card: HTMLElement): RuntimeOptions {
  return {
    maxTilt: readNumber(card.dataset.tiltMax, DEFAULT_TILT_CARD_OPTIONS.maxTilt, 0, 18),
    scale: readNumber(card.dataset.tiltScale, DEFAULT_TILT_CARD_OPTIONS.scale, 1, 1.08),
    lift: readNumber(card.dataset.tiltLift, DEFAULT_TILT_CARD_OPTIONS.lift, 0, 20),
    smoothing: readNumber(card.dataset.tiltSmoothing, DEFAULT_TILT_CARD_OPTIONS.smoothing, 0.04, 0.4),
    glareStrength: readNumber(card.dataset.tiltGlare, DEFAULT_TILT_CARD_OPTIONS.glareStrength, 0, 0.75),
    parallax: readNumber(card.dataset.tiltParallax, DEFAULT_TILT_CARD_OPTIONS.parallax, 0, 24),
    perspective: readNumber(card.dataset.tiltPerspective, DEFAULT_TILT_CARD_OPTIONS.perspective, 500, 2200),
  };
}

/** readNumber 加载对应业务数据。 */
function readNumber(value: string | undefined, fallback: number, minimum: number, maximum: number) {
  /** parsed 保存解析结果。 */
  const parsed = Number(value);
  return Number.isFinite(parsed) ? clamp(parsed, minimum, maximum) : fallback;
}

/** interpolateValues 实现对应业务逻辑。 */
function interpolateValues(current: MotionValues, target: MotionValues, alpha: number): MotionValues {
  return {
    rotateX: lerp(current.rotateX, target.rotateX, alpha),
    rotateY: lerp(current.rotateY, target.rotateY, alpha),
    scale: lerp(current.scale, target.scale, alpha),
    lift: lerp(current.lift, target.lift, alpha),
    glareX: lerp(current.glareX, target.glareX, alpha),
    glareY: lerp(current.glareY, target.glareY, alpha),
    glareOpacity: lerp(current.glareOpacity, target.glareOpacity, alpha),
    holographicOpacity: lerp(current.holographicOpacity, target.holographicOpacity, alpha),
    parallaxX: lerp(current.parallaxX, target.parallaxX, alpha),
    parallaxY: lerp(current.parallaxY, target.parallaxY, alpha),
  };
}

/** isSettled 校验对应业务条件。 */
function isSettled(current: MotionValues, target: MotionValues) {
  return (Object.keys(current) as Array<keyof MotionValues>).every(
    (key) => Math.abs(current[key] - target[key]) <= MOTION_EPSILON,
  );
}

/** applyValues 执行对应业务流程。 */
function applyValues(card: HTMLElement, values: MotionValues) {
  card.style.setProperty('--tilt-rotate-x', `${values.rotateX.toFixed(3)}deg`);
  card.style.setProperty('--tilt-rotate-y', `${values.rotateY.toFixed(3)}deg`);
  card.style.setProperty('--tilt-scale', values.scale.toFixed(4));
  card.style.setProperty('--tilt-lift', `${values.lift.toFixed(3)}px`);
  card.style.setProperty('--tilt-glare-x', `${values.glareX.toFixed(2)}%`);
  card.style.setProperty('--tilt-glare-y', `${values.glareY.toFixed(2)}%`);
  card.style.setProperty('--tilt-glare-opacity', values.glareOpacity.toFixed(3));
  card.style.setProperty('--tilt-holographic-opacity', values.holographicOpacity.toFixed(3));
  card.style.setProperty('--tilt-layer-subtle-x', `${(values.parallaxX * 0.34).toFixed(3)}px`);
  card.style.setProperty('--tilt-layer-subtle-y', `${(values.parallaxY * 0.34).toFixed(3)}px`);
  card.style.setProperty('--tilt-layer-medium-x', `${(values.parallaxX * 0.62).toFixed(3)}px`);
  card.style.setProperty('--tilt-layer-medium-y', `${(values.parallaxY * 0.62).toFixed(3)}px`);
  card.style.setProperty('--tilt-layer-strong-x', `${values.parallaxX.toFixed(3)}px`);
  card.style.setProperty('--tilt-layer-strong-y', `${values.parallaxY.toFixed(3)}px`);
}

/** lerp 实现对应业务逻辑。 */
function lerp(from: number, to: number, amount: number) {
  return from + (to - from) * amount;
}

/** clamp 实现对应业务逻辑。 */
function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}
