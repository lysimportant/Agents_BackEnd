/** REQUEST_TIMEOUT_MS 保存模块使用的固定配置或共享状态。 */
const REQUEST_TIMEOUT_MS = 12_000;

/** RETRY_DELAYS_MS 保存模块使用的固定配置或共享状态。 */
const RETRY_DELAYS_MS = [350, 900];

/** SessionRequestInit 扩展会话请求选项，允许特定长耗时请求显式关闭超时。 */
interface SessionRequestInit extends RequestInit {
  /** timeoutMs 设置单次请求超时；传入 null 时由浏览器持续等待。 */
  timeoutMs?: number | null;
}

/** sleep 实现对应业务逻辑。 */
function sleep(delay: number) {
  return new Promise((resolve) => window.setTimeout(resolve, delay));
}

/** canRetry 校验对应业务条件。 */
function canRetry(method: string) {
  return method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
}

/**
 * 发送带会话凭证的 API 请求，默认设置明确超时上限。
 * 仅在短暂网络故障后重试幂等的读取请求。
 */
export async function requestWithSession(input: string, init: SessionRequestInit = {}) {
  /** requestTimeoutMs 保存调用方显式传入的超时策略。 */
  const { timeoutMs: requestTimeoutMs, ...requestInit } = init;
  /** timeoutMs 保存当前请求的超时策略，null 表示不主动中断。 */
  const timeoutMs = requestTimeoutMs === undefined ? REQUEST_TIMEOUT_MS : requestTimeoutMs;
  /** method 保存请求方法。 */
  const method = (requestInit.method ?? 'GET').toUpperCase();
  /** retryDelays 保存重试。 */
  const retryDelays = canRetry(method) ? RETRY_DELAYS_MS : [];
  let lastError: unknown;

  for (let attempt = 0; attempt <= retryDelays.length; attempt += 1) {
    /** controller 保存请求控制器。 */
    const controller = new AbortController();
    /** timeout 负责计算或维护变量 timeout。 */
    const timeout = timeoutMs === null ? null : window.setTimeout(() => controller.abort(), timeoutMs);
    /** abortFromCaller 负责计算或维护起始时间。 */
    const abortFromCaller = () => controller.abort(requestInit.signal?.reason);

    if (requestInit.signal?.aborted) {
      if (timeout !== null) {
        window.clearTimeout(timeout);
      }
      throw requestInit.signal.reason;
    }
    requestInit.signal?.addEventListener('abort', abortFromCaller, { once: true });

    try {
      return await fetch(input, {
        ...requestInit,
        credentials: 'include',
        cache: requestInit.cache,
        headers: requestInit.headers,
        signal: controller.signal,
      });
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      lastError = error;
      if (requestInit.signal?.aborted || attempt === retryDelays.length) {
        throw error;
      }
      await sleep(retryDelays[attempt]);
    } finally {
      if (timeout !== null) {
        window.clearTimeout(timeout);
      }
      requestInit.signal?.removeEventListener('abort', abortFromCaller);
    }
  }

  throw lastError;
}
