type ViewTransitionDocument = Document & {
  startViewTransition?: (update: () => void) => {
    finished: Promise<void>;
    ready?: Promise<void>;
    updateCallbackDone?: Promise<void>;
  };
};

/** 浏览器支持 View Transitions 时，为小范围 DOM 更新添加过渡动画。 */
export function runViewTransition(update: () => void) {
  if (typeof document === 'undefined') {
    update();
    return;
  }
  /** transitionDocument 保存变量 transitionDocument。 */
  const transitionDocument = document as ViewTransitionDocument;
  if (typeof transitionDocument.startViewTransition !== 'function') {
    update();
    return;
  }
  try {
    /** transition 保存变量 transition。 */
    const transition = transitionDocument.startViewTransition(update);
    // 被跳过的过渡会拒绝生命周期 Promise，这是正常的动画中断。
    void Promise.resolve(transition.finished).catch(() => undefined);
    void Promise.resolve(transition.ready).catch(() => undefined);
    void Promise.resolve(transition.updateCallbackDone).catch(() => undefined);
  } catch {
    update();
  }
}
