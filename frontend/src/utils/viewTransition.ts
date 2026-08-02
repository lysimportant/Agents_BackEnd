type ViewTransitionDocument = Document & {
  startViewTransition?: (update: () => void) => {
    finished: Promise<void>;
    ready?: Promise<void>;
    updateCallbackDone?: Promise<void>;
  };
};

/** Run a small DOM update transition when the browser supports View Transitions. */
export function runViewTransition(update: () => void) {
  if (typeof document === 'undefined') {
    update();
    return;
  }
  const transitionDocument = document as ViewTransitionDocument;
  if (typeof transitionDocument.startViewTransition !== 'function') {
    update();
    return;
  }
  try {
    const transition = transitionDocument.startViewTransition(update);
    // A skipped transition rejects its lifecycle promises; it is a normal interruption.
    void Promise.resolve(transition.finished).catch(() => undefined);
    void Promise.resolve(transition.ready).catch(() => undefined);
    void Promise.resolve(transition.updateCallbackDone).catch(() => undefined);
  } catch {
    update();
  }
}
