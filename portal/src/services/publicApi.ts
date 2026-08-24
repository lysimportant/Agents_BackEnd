import { API_BASE_URL, resolveRevalidateSeconds } from '@/config/constants';
import type {
  PublicApiError as PublicApiErrorBody,
  PublicArticleDetail,
  PublicArticleListItem,
  PublicCategory,
  PublicDetailResponse,
  PublicFileListItem,
  PublicFileComment,
  PublicFileInteraction,
  PublicFileTagResponse,
  PublicListResponse,
  PublicSearchResult,
  PublicSiteSummary,
} from '@/types/publicContent';

/** 单次请求超时时间（毫秒）。 */
const REQUEST_TIMEOUT_MS = 10000;

/** 服务端公开 API 地址；容器内优先使用 Compose 提供的后端服务地址。 */
const PUBLIC_API_REQUEST_BASE_URL = process.env.BACKEND_INTERNAL_URL || API_BASE_URL;

/** 网络瞬时错误与 5xx 的最大重试次数。 */
const MAX_RETRIES = 2;

/** 重试退避间隔（毫秒），按尝试次数递增。 */
const RETRY_DELAYS_MS = [350, 900];

/** 扩展 Next.js fetch 的初始化选项，承载服务端缓存配置。 */
type PortalFetchInit = RequestInit & {
  next?: { revalidate?: number; tags?: string[] };
};

/** 公开接口请求选项，供服务端与客户端分别按需传入。 */
export interface PublicFetchOptions {
  /** 客户端取消信号。 */
  signal?: AbortSignal;
  /** 是否携带门户偏好 Cookie，让后端按当前访客设置筛选内容。 */
  credentials?: RequestCredentials;
  /** 服务端转发到后端的认证 Cookie。 */
  headers?: HeadersInit;
  /** 服务端内容缓存秒数，传入时启用 Next.js revalidation。 */
  revalidate?: number;
  /** 服务端缓存标签，便于后续主动失效。 */
  tags?: string[];
}

/** 公开文章/图片/资源的列表查询参数。 */
export interface PublicListParams {
  /** 页码，从 1 开始。 */
  page?: number;
  /** 每页条目数，上限 50。 */
  pageSize?: number;
  /** 分类名，与后端精确匹配。 */
  category?: string;
  /** 搜索关键词。 */
  keyword?: string;
  /** 排序方式，当前后端解析后未生效。 */
  sort?: string;
}

/** 公开 API 统一错误，保留 HTTP 状态码与稳定错误码供页面映射文案。 */
export class PublicApiError extends Error {
  /** HTTP 状态码，网络或超时错误时为 0。 */
  readonly status: number;
  /** 后端稳定错误码，可能缺省。 */
  readonly code?: string;

  constructor(status: number, code?: string, message?: string) {
    super(message ?? '公开接口请求失败');
    this.name = 'PublicApiError';
    this.status = status;
    this.code = code;
  }
}

/** 判断错误是否为超时中断。 */
function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError';
}

/** 判断错误是否值得有限重试：仅网络瞬时错误与 5xx，4xx 确定性结果不重试。 */
function isRetryable(error: unknown): boolean {
  if (error instanceof PublicApiError) {
    return error.status >= 500;
  }
  return error instanceof TypeError || isAbortError(error);
}

/** 尝试读取后端错误响应中的稳定错误码，读取失败时返回 undefined。 */
async function readErrorCode(response: Response): Promise<string | undefined> {
  try {
    const body = (await response.json()) as PublicApiErrorBody;
    return body.code;
  } catch {
    return undefined;
  }
}

/** 按指定毫秒数挂起，用于重试退避。 */
function sleep(milliseconds: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

/**
 * 公开接口 JSON 请求核心：拼装绝对地址，带超时与取消，并对瞬时错误做有限退避重试。
 * 服务端传入 revalidate 时启用 Next.js 数据缓存，否则使用 no-store 保证新鲜度。
 */
async function requestPublicJson<T>(
  path: string,
  options: PublicFetchOptions = {},
): Promise<T> {
  const url = PUBLIC_API_REQUEST_BASE_URL + path;
  let attempt = 0;
  let lastError: unknown = null;

  while (attempt <= MAX_RETRIES) {
    // 外部已取消时立即终止，不触发重试。
    if (options.signal?.aborted) {
      throw new PublicApiError(0, 'aborted');
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    const handleExternalAbort = () => controller.abort();
    options.signal?.addEventListener('abort', handleExternalAbort, { once: true });

    try {
      const init: PortalFetchInit = {
        signal: controller.signal,
        credentials: options.credentials,
        headers: options.headers,
      };
      if (options.revalidate !== undefined) {
        init.next = { revalidate: options.revalidate, tags: options.tags };
      } else {
        init.cache = 'no-store';
      }

      const response = await fetch(url, init);
      if (response.ok) {
        return (await response.json()) as T;
      }

      const status = response.status;
      const code = await readErrorCode(response);
      // 4xx 是确定性结果，不重试，直接抛出。
      if (status >= 400 && status < 500) {
        throw new PublicApiError(status, code);
      }
      // 5xx 可有限重试。
      lastError = new PublicApiError(status, code);
    } catch (error) {
      // 外部取消优先于重试判断。
      if (options.signal?.aborted) {
        throw new PublicApiError(0, 'aborted');
      }
      lastError = error;
    } finally {
      clearTimeout(timeoutId);
      options.signal?.removeEventListener('abort', handleExternalAbort);
    }

    if (!isRetryable(lastError) || attempt >= MAX_RETRIES) {
      break;
    }
    await sleep(RETRY_DELAYS_MS[attempt] ?? RETRY_DELAYS_MS[RETRY_DELAYS_MS.length - 1]);
    attempt += 1;
  }

  if (lastError instanceof Error) {
    throw lastError;
  }
  throw new PublicApiError(0, 'network_error');
}

/** 将列表查询参数编码为 URL 查询串，无参数时返回空字符串。 */
function buildQueryString(params: PublicListParams): string {
  const searchParams = new URLSearchParams();
  if (params.page && params.page > 1) {
    searchParams.set('page', String(params.page));
  }
  if (params.pageSize) {
    searchParams.set('pageSize', String(params.pageSize));
  }
  if (params.category) {
    searchParams.set('category', params.category);
  }
  if (params.keyword) {
    searchParams.set('keyword', params.keyword);
  }
  if (params.sort) {
    searchParams.set('sort', params.sort);
  }
  const query = searchParams.toString();
  return query ? '?' + query : '';
}

/** 获取公开文章列表。 */
export function listArticles(
  params: PublicListParams = {},
  options?: PublicFetchOptions,
): Promise<PublicListResponse<PublicArticleListItem>> {
  return requestPublicJson<PublicListResponse<PublicArticleListItem>>(
    '/api/public/articles' + buildQueryString(params),
    options,
  );
}

/** 获取公开文章详情，返回响应中的 item 字段。 */
export async function getArticle(
  id: number,
  options?: PublicFetchOptions,
): Promise<PublicArticleDetail> {
  const response = await requestPublicJson<PublicDetailResponse<PublicArticleDetail>>(
    '/api/public/articles/' + String(id),
    options,
  );
  return response.item;
}

/** 获取公开图片列表。 */
export function listImages(
  params: PublicListParams = {},
  options?: PublicFetchOptions,
): Promise<PublicListResponse<PublicFileListItem>> {
  return requestPublicJson<PublicListResponse<PublicFileListItem>>(
    '/api/public/images' + buildQueryString(params),
    options,
  );
}

/** 发送不自动重试的公开写请求，避免点赞或评论因网络重试重复执行。 */
async function requestPublicMutationJson<T>(path: string, init: RequestInit): Promise<T> {
  // controller 保存本次写请求的超时取消控制器。
  const controller = new AbortController();
  // timeoutId 保存写请求的十秒超时计时器。
  const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    // response 保存公开互动接口响应。
    const response = await fetch(API_BASE_URL + path, {
      ...init,
      signal: controller.signal,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...init.headers,
      },
    });
    if (!response.ok) {
      // code 保存后端稳定错误码，供预览层区分登录状态和普通失败。
      const code = await readErrorCode(response);
      throw new PublicApiError(response.status, code);
    }
    return (await response.json()) as T;
  } catch (error) {
    if (isAbortError(error)) {
      throw new PublicApiError(0, 'timeout');
    }
    throw error;
  } finally {
    clearTimeout(timeoutId);
  }
}

/** 获取公开图片的点赞状态和最近评论。 */
export function getImageInteraction(
  fileID: number,
  options?: PublicFetchOptions,
): Promise<PublicFileInteraction> {
  return requestPublicJson<PublicFileInteraction>(
    '/api/public/files/' + String(fileID) + '/interactions',
    { ...options, credentials: 'include' },
  );
}

/** 切换当前登录用户对公开图片的点赞状态。 */
export function toggleImageLike(fileID: number): Promise<PublicFileInteraction> {
  return requestPublicMutationJson<PublicFileInteraction>(
    '/api/public/files/' + String(fileID) + '/like',
    { method: 'POST', body: '{}' },
  );
}

/** 为当前登录用户可见的公开图片追加一个标签。 */
export function addImageTag(fileID: number, tag: string): Promise<PublicFileTagResponse> {
  return requestPublicMutationJson<PublicFileTagResponse>(
    '/api/public/files/' + String(fileID) + '/tags',
    { method: 'POST', body: JSON.stringify({ tag }) },
  );
}

/** 发送当前登录用户的公开图片评论。 */
export async function createImageComment(
  fileID: number,
  content: string,
): Promise<PublicFileComment> {
  // response 保存后端用 item 包装的新评论响应。
  const response = await requestPublicMutationJson<{ item: PublicFileComment }>(
    '/api/public/files/' + String(fileID) + '/comments',
    { method: 'POST', body: JSON.stringify({ content }) },
  );
  return response.item;
}

/** 获取公开资源列表。 */
export function listResources(
  params: PublicListParams = {},
  options?: PublicFetchOptions,
): Promise<PublicListResponse<PublicFileListItem>> {
  return requestPublicJson<PublicListResponse<PublicFileListItem>>(
    '/api/public/resources' + buildQueryString(params),
    options,
  );
}

/** 获取公开分类列表。 */
export async function listCategories(
  options?: PublicFetchOptions,
): Promise<PublicCategory[]> {
  const response = await requestPublicJson<{ items: PublicCategory[] }>(
    '/api/public/categories',
    options,
  );
  return response.items;
}

/** 获取站点聚合概览，供首页与 SEO 使用。 */
export function getSiteSummary(
  options?: PublicFetchOptions,
): Promise<PublicSiteSummary> {
  return requestPublicJson<PublicSiteSummary>('/api/public/site-summary', options);
}

/** 聚合搜索公开文章、图片与资源。 */
export function searchPublic(
  keyword: string,
  options?: PublicFetchOptions,
): Promise<PublicSearchResult> {
  return requestPublicJson<PublicSearchResult>(
    '/api/public/search?keyword=' + encodeURIComponent(keyword),
    options,
  );
}

/** 获取服务端内容缓存秒数的快捷常量，供页面统一传入 revalidate。 */
export function defaultRevalidate(): number {
  return resolveRevalidateSeconds();
}

export { API_BASE_URL };
