/**
 * JSON-LD 组件，将结构化数据注入页面 <script type="application/ld+json">。
 * 通过 ref 与 effect 保持数据变化时更新，避免重复注入。
 */
'use client';

import { useEffect, useId, useRef } from 'react';

/** JsonLd 渲染并维护 JSON-LD 结构化数据脚本。 */
export function JsonLd({ data }: { data: Record<string, unknown> }) {
  const id = useId();
  const lastJson = useRef<string>('');
  const json = JSON.stringify(data);
  useEffect(() => {
    if (lastJson.current === json) return;
    lastJson.current = json;
    const existing = document.getElementById(id);
    if (existing) existing.remove();
    const script = document.createElement('script');
    script.id = id;
    script.type = 'application/ld+json';
    script.textContent = json;
    document.head.appendChild(script);
    return () => { script.remove(); };
  }, [id, json]);
  return null;
}