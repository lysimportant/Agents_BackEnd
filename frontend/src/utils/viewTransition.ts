type ViewTransitionDocument = Document & {
  startViewTransition?: (update: () => void) => { finished: Promise<void> };
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
    transitionDocument.startViewTransition(update);
  } catch {
    update();
  }
}
