const REQUEST_TIMEOUT_MS = 12_000;
const RETRY_DELAYS_MS = [350, 900];

function sleep(delay: number) {
  return new Promise((resolve) => window.setTimeout(resolve, delay));
}

function canRetry(method: string) {
  return method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
}

export async function requestWithSession(input: string, init: RequestInit = {}) {
  const method = (init.method ?? 'GET').toUpperCase();
  const retryDelays = canRetry(method) ? RETRY_DELAYS_MS : [];
  let lastError: unknown;

  for (let attempt = 0; attempt <= retryDelays.length; attempt += 1) {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    const abortFromCaller = () => controller.abort(init.signal?.reason);

    if (init.signal?.aborted) {
      window.clearTimeout(timeout);
      throw init.signal.reason;
    }
    init.signal?.addEventListener('abort', abortFromCaller, { once: true });

    try {
      return await fetch(input, {
        ...init,
        credentials: 'include',
        cache: init.cache,
        headers: init.headers,
        signal: controller.signal,
      });
    } catch (error) {
      lastError = error;
      if (init.signal?.aborted || attempt === retryDelays.length) {
        throw error;
      }
      await sleep(retryDelays[attempt]);
    } finally {
      window.clearTimeout(timeout);
      init.signal?.removeEventListener('abort', abortFromCaller);
    }
  }

  throw lastError;
}
