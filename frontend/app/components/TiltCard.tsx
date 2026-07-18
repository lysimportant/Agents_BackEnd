'use client';

import {
  forwardRef,
  type CSSProperties,
  type HTMLAttributes,
} from 'react';

export type TiltCardOptions = {
  /** Maximum rotation around each axis, in degrees. */
  maxTilt?: number;
  /** Scale applied while the pointer is over the card. */
  scale?: number;
  /** Upward movement while active, in pixels. */
  lift?: number;
  /** Lerp strength used by the requestAnimationFrame loop (0-1). */
  smoothing?: number;
  /** Peak opacity of the pointer-following radial glare (0-1). */
  glareStrength?: number;
  /** Maximum movement of the strongest parallax layer, in pixels. */
  parallax?: number;
  /** Perspective depth, in pixels. */
  perspective?: number;
  /** Adds a subtle rainbow holographic overlay. */
  holographic?: boolean;
  /** Keeps the card static while preserving its normal appearance. */
  disabled?: boolean;
};

export type TiltCardProps = HTMLAttributes<HTMLDivElement> & TiltCardOptions;

export const DEFAULT_TILT_CARD_OPTIONS = {
  maxTilt: 4.5,
  scale: 1.01,
  lift: 4,
  smoothing: 0.14,
  glareStrength: 0.16,
  parallax: 6,
  perspective: 1100,
} as const;

/**
 * Reusable 3D card surface. TiltCardEffects is mounted once in the root layout,
 * so this component only declares per-card options and stays inexpensive.
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

export type TiltCardLayerProps = HTMLAttributes<HTMLDivElement> & {
  depth?: 'subtle' | 'medium' | 'strong';
};

/** A content layer that moves independently inside a TiltCard. */
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

function joinClassNames(...classNames: Array<string | false | null | undefined>) {
  return classNames.filter(Boolean).join(' ');
}
