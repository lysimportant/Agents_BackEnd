'use client';

import { useEffect } from 'react';
import { DEFAULT_TILT_CARD_OPTIONS } from './TiltCard';

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
  '.antd-content-view .file-card',
  '.antd-content-view .recycle-file-card',
  '.antd-content-view .article-library-card',
  '.antd-content-view .ant-card:not(.file-browser-panel):not([data-tilt-disabled="true"])',
  '.antd-content-view [data-slot="card"]',
].join(',');

type TiltCardEffectsProps = {
  selector?: string;
};

type MotionValues = {
  rotateX: number;
  rotateY: number;
  scale: number;
  lift: number;
  glareX: number;
  glareY: number;
  glareOpacity: number;
  holographicOpacity: number;
  parallaxX: number;
  parallaxY: number;
};

type RuntimeOptions = {
  maxTilt: number;
  scale: number;
  lift: number;
  smoothing: number;
  glareStrength: number;
  parallax: number;
  perspective: number;
};

type CardMotion = {
  card: HTMLElement;
  current: MotionValues;
  target: MotionValues;
  options: RuntimeOptions;
  rect: DOMRect | null;
  active: boolean;
};

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

const MOTION_EPSILON = 0.01;

/**
 * Enhances current and future business cards through event delegation. A single
 * animation loop serves the whole application and pointermove only writes
 * targets; DOM geometry is cached until the active card or viewport changes.
 */
export function TiltCardEffects({ selector = DEFAULT_TILT_CARD_SELECTOR }: TiltCardEffectsProps) {
  useEffect(() => {
    const finePointer = window.matchMedia('(hover: hover) and (pointer: fine)');
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const trackedCards = new Set<HTMLElement>();
    const autoEnhancedCards = new Set<HTMLElement>();
    const motions = new Map<HTMLElement, CardMotion>();

    let enabled = finePointer.matches && !reducedMotion.matches;
    let activeCard: HTMLElement | null = null;
    let frameId = 0;
    let lastFrameTime = performance.now();

    const enhanceCard = (card: HTMLElement) => {
      if (card.dataset.tiltDisabled === 'true' || trackedCards.has(card)) return;
      trackedCards.add(card);
      card.dataset.tiltReady = 'true';
      if (!card.classList.contains('tilt-card-surface')) {
        autoEnhancedCards.add(card);
        card.classList.add('tilt-card-surface');
      }
    };

    const enhanceWithin = (root: ParentNode) => {
      if (root instanceof HTMLElement && root.matches(selector)) enhanceCard(root);
      root.querySelectorAll<HTMLElement>(selector).forEach(enhanceCard);
    };

    const resetCard = (card: HTMLElement) => {
      applyValues(card, RESTING_VALUES);
      delete card.dataset.tiltActive;
      card.style.removeProperty('--tilt-perspective');
    };

    const stopMotion = () => {
      if (frameId) cancelAnimationFrame(frameId);
      frameId = 0;
      activeCard = null;
      motions.forEach(({ card }) => resetCard(card));
      motions.clear();
    };

    const syncCapability = () => {
      enabled = finePointer.matches && !reducedMotion.matches;
      if (enabled) enhanceWithin(document);
      else stopMotion();
    };

    const getMotion = (card: HTMLElement) => {
      const existing = motions.get(card);
      if (existing) return existing;

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

    const settle = (card: HTMLElement | null) => {
      if (!card) return;
      const motion = getMotion(card);
      motion.active = false;
      motion.target = { ...RESTING_VALUES };
      scheduleFrame();
    };

    const activate = (card: HTMLElement) => {
      if (activeCard !== card) {
        settle(activeCard);
        activeCard = card;
      }

      const motion = getMotion(card);
      motion.active = true;
      motion.options = readOptions(card);
      motion.rect = card.getBoundingClientRect();
      card.dataset.tiltActive = 'true';
      card.style.setProperty('--tilt-perspective', `${motion.options.perspective}px`);
      return motion;
    };

    const releaseActiveCard = () => {
      settle(activeCard);
      activeCard = null;
    };

    const findCard = (target: EventTarget | null) => {
      if (!(target instanceof Element)) return null;
      const card = target.closest<HTMLElement>(selector);
      if (!card || card.dataset.tiltDisabled === 'true') return null;

      const disabledRegion = target.closest<HTMLElement>('[data-tilt-disabled="true"]');
      if (disabledRegion && card.contains(disabledRegion)) return null;
      return card;
    };

    const onPointerMove = (event: PointerEvent) => {
      if (!enabled || event.pointerType === 'touch') return;
      const card = findCard(event.target);
      if (!card) {
        releaseActiveCard();
        return;
      }

      const motion = activeCard === card ? getMotion(card) : activate(card);
      const rect = motion.rect ?? card.getBoundingClientRect();
      motion.rect = rect;
      if (rect.width <= 0 || rect.height <= 0) return;

      const normalizedX = clamp(((event.clientX - rect.left) / rect.width) * 2 - 1, -1, 1);
      const normalizedY = clamp(((event.clientY - rect.top) / rect.height) * 2 - 1, -1, 1);
      const largeSurfaceFactor = rect.width > 760 || rect.height > 520 ? 0.56 : 1;
      const scaleFactor = rect.width > 760 || rect.height > 520 ? 0.44 : 1;
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

    const onPointerOut = (event: PointerEvent) => {
      if (event.relatedTarget === null) releaseActiveCard();
    };

    const invalidateGeometry = () => {
      if (activeCard) getMotion(activeCard).rect = null;
    };

    const onVisibilityChange = () => {
      if (document.hidden) releaseActiveCard();
    };

    const observer = new MutationObserver((records) => {
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
      const elapsedFrames = Math.min(2, Math.max(0.5, (now - lastFrameTime) / 16.667));
      lastFrameTime = now;
      let needsAnotherFrame = false;

      motions.forEach((motion, card) => {
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

function readNumber(value: string | undefined, fallback: number, minimum: number, maximum: number) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? clamp(parsed, minimum, maximum) : fallback;
}

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

function isSettled(current: MotionValues, target: MotionValues) {
  return (Object.keys(current) as Array<keyof MotionValues>).every(
    (key) => Math.abs(current[key] - target[key]) <= MOTION_EPSILON,
  );
}

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

function lerp(from: number, to: number, amount: number) {
  return from + (to - from) * amount;
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}
