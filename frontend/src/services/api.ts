/** REQUEST_TIMEOUT_MS 保存模块使用的固定配置或共享状态。 */
const REQUEST_TIMEOUT_MS = 12_000;

/** RETRY_DELAYS_MS 保存模块使用的固定配置或共享状态。 */
const RETRY_DELAYS_MS = [350, 900];

/** sleep 实现对应业务逻辑。 */
function sleep(delay: number) {
  return new Promise((resolve) => window.setTimeout(resolve, delay));
}

/** canRetry 校验对应业务条件。 */
function canRetry(method: string) {
  return method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
}

/**
 * 发送带会话凭证且有明确超时上限的 API 请求。
 * 仅在短暂网络故障后重试幂等的读取请求。
 */
export async function requestWithSession(input: string, init: RequestInit = {}) {
  /** method 保存请求方法。 */
  const method = (init.method ?? 'GET').toUpperCase();
  /** retryDelays 保存重试。 */
  const retryDelays = canRetry(method) ? RETRY_DELAYS_MS : [];
  let lastError: unknown;

  for (let attempt = 0; attempt <= retryDelays.length; attempt += 1) {
    /** controller 保存请求控制器。 */
    const controller = new AbortController();
    /** timeout 负责计算或维护变量 timeout。 */
    const timeout = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    /** abortFromCaller 负责计算或维护起始时间。 */
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
    /** error 保存当前操作结果以及可能返回的错误状态。 */
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
