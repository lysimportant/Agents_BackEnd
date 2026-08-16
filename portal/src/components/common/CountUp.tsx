'use client';

import { useEffect, useRef, useState } from 'react';

/** CountUp 在元素进入视口后播放数字滚动增长动画，支持减少动态效果时直接显示最终值。 */
export function CountUp({
  value,
  locale,
  duration = 1200,
}: {
  value: number;
  locale: string;
  duration?: number;
}) {
  const [display, setDisplay] = useState(0);
  const ref = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    const node = ref.current;
    if (!node || typeof IntersectionObserver === 'undefined') {
      setDisplay(value);
      return;
    }
    const prefersReduced =
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) {
      // 减少动态效果时直接显示最终值，属于与系统偏好的一次性同步，不需要动画。
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setDisplay(value);
      return;
    }

    let frame = 0;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0].isIntersecting) {
          return;
        }
        const start = performance.now();
        const tick = (now: number) => {
          const ratio = Math.min(1, (now - start) / duration);
          const eased = 1 - Math.pow(1 - ratio, 3);
          setDisplay(Math.round(eased * value));
          if (ratio < 1) {
            frame = window.requestAnimationFrame(tick);
          }
        };
        frame = window.requestAnimationFrame(tick);
        observer.disconnect();
      },
      { threshold: 0.3 },
    );
    observer.observe(node);
    return () => {
      observer.disconnect();
      window.cancelAnimationFrame(frame);
    };
  }, [value, duration]);

  return <span ref={ref}>{new Intl.NumberFormat(locale).format(display)}</span>;
}
