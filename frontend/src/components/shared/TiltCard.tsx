'use client';

import {
  forwardRef,
  type CSSProperties,
  type HTMLAttributes,
} from 'react';

/** TiltCardOptions 定义对应业务的数据结构与调用契约。 */
export type TiltCardOptions = {
  /** 每个轴允许的最大旋转角度。 */
  maxTilt?: number;
  /** 指针悬停卡片时使用的缩放比例。 */
  scale?: number;
  /** 卡片激活时向上移动的像素数。 */
  lift?: number;
  /** requestAnimationFrame 循环使用的线性插值强度，范围为 0 到 1。 */
  smoothing?: number;
  /** 跟随指针的径向高光最大不透明度，范围为 0 到 1。 */
  glareStrength?: number;
  /** 最强视差图层允许移动的最大像素数。 */
  parallax?: number;
  /** 透视深度，单位为像素。 */
  perspective?: number;
  /** 是否添加轻微的彩虹全息叠层。 */
  holographic?: boolean;
  /** 是否在保持正常外观的同时禁用卡片动态效果。 */
  disabled?: boolean;
};

/** TiltCardProps 定义对应业务的数据结构与调用契约。 */
export type TiltCardProps = HTMLAttributes<HTMLDivElement> & TiltCardOptions;

/** DEFAULT_TILT_CARD_OPTIONS 保存模块使用的固定配置或共享状态。 */
export const DEFAULT_TILT_CARD_OPTIONS = {
  maxTilt: 1.2,
  scale: 1.01,
  lift: 3.31,
  smoothing: 0.14,
  glareStrength: 0.44,
  parallax: 4.95,
  perspective: 1100,
} as const;

/**
 * 可复用的 3D 卡片表面。TiltCardEffects 仅在根布局挂载一次，
 * 因此本组件只声明单卡选项并保持较低运行开销。
 */
export const TiltCard = forwardRef<HTMLDivElement, TiltCardProps>(function TiltCard(
  {
    children,
    className,
    style,
    maxTilt = DEFAULT_TILT_CARD_OPTIONS.maxTilt,
    scale = DEFAULT_TILT_CARD_OPTIONS.scale,
    lift = DEFAULT_TILT_CARD_OPTIONS.lift,
    smoothing = DEFAULT_TILT_CARD_OPTIONS.smoothing,
    glareStrength = DEFAULT_TILT_CARD_OPTIONS.glareStrength,
    parallax = DEFAULT_TILT_CARD_OPTIONS.parallax,
    perspective = DEFAULT_TILT_CARD_OPTIONS.perspective,
    holographic = false,
    disabled = false,
    ...props
  },
  ref,
) {
  return (
    <div
      ref={ref}
      className={joinClassNames(
        'tilt-card-surface',
        holographic && 'tilt-card--holographic',
        className,
      )}
      data-tilt-card="true"
      data-tilt-disabled={disabled ? 'true' : undefined}
      data-tilt-max={maxTilt}
      data-tilt-scale={scale}
      data-tilt-lift={lift}
      data-tilt-smoothing={smoothing}
      data-tilt-glare={glareStrength}
      data-tilt-parallax={parallax}
      data-tilt-perspective={perspective}
      style={{
        ...style,
        '--tilt-perspective': `${perspective}px`,
      } as CSSProperties}
      {...props}
    >
      {children}
    </div>
  );
});

/** TiltCardLayerProps 定义对应业务的数据结构与调用契约。 */
export type TiltCardLayerProps = HTMLAttributes<HTMLDivElement> & {
  depth?: 'subtle' | 'medium' | 'strong';
};

/** 在 TiltCard 内独立移动的内容图层。 */
export const TiltCardLayer = forwardRef<HTMLDivElement, TiltCardLayerProps>(function TiltCardLayer(
  { depth = 'medium', className, children, ...props },
  ref,
) {
  return (
    <div
      ref={ref}
      className={joinClassNames('tilt-card-layer', className)}
      data-tilt-layer={depth}
      {...props}
    >
      {children}
    </div>
  );
});

/** joinClassNames 定义对应业务的数据结构与调用契约。 */
function joinClassNames(...classNames: Array<string | false | null | undefined>) {
  return classNames.filter(Boolean).join(' ');
}
