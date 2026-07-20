'use client';

import { App } from 'antd';
import type { ReactNode } from 'react';

export function GlobalFeedbackProvider({ children }: { children: ReactNode }) {
  return (
    <App component="div" message={{ duration: 2.4, maxCount: 3, top: 22 }} notification={{ placement: 'bottomRight', maxCount: 5 }}>
      {children}
    </App>
  );
}
