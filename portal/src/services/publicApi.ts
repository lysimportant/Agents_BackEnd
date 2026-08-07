/**
 * 门户公开数据服务，封装后端 /api/public/* 接口的请求与错误归一化。
 * 客户端请求统一走此层，对 4xx 不做重试并对潜在网络错误分类。
 * 提供商超时、去重与有限重试能力，降低门户页面加载失败率。
 */
import { API_BASE_URL, PORTAL_REVALIDATE_SECONDS } from '@/config/constants';
import type {
  PublicArticleDetail,
  PublicArticleListItem,
  PublicCategory,
  PublicFileListItem,
  PublicListResponse,
  PublicSearchResult,
  PublicSiteSummary,
} from '@/types/publicContent';

/** MAX_PAGE_SIZE 保存后端允许的最大每页条数。 */
export const MAX_PAGE_SIZE = 50;

/** DEFAULT_PAGE_SIZE 保存公开列表的默认每页条数。 */
export const DEFAULT_PAGE_SIZE = 24;

/** CLIENT_TIMEOUT_MS 保存客户端请求超时毫秒数。 */
const CLIENT_TIMEOUT_MS = 15_000;

/** RetryableError 表示可重试的错误类型，5xx 与网络错误重试，4xx 不重试。 */
type RetryableError = 'NETWORK' | 'SERVER';

/** PublicServiceError 表示门户公开服务抛出的标准化错误，供 C 端页面提示。 */
export interface PublicServiceError {
  /** code 错误编码。 */
  code: string;
  /** kind 错误类型，用于判断是否可重试。 */
  kind: RetryableError;
  /** message 错误描述信息。 */
  message: string;
}

/** buildQuery 根据参数对象生成查询字符串。 */
function buildQuery(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '' || value === null) continue;
    search.set(key, String(value));
  }
  const query = search.toString();
  return query ? `?${query}` : '';
}

/** inflight 保存在途请求，用于合并相同路径的并发请求。 */
const inflight = new Map<string, Promise<unknown>>();

/** RETRY_DELAYS 保存重试前的等待毫秒数序列。 */
const RETRY_DELAYS = [300, 800];

/** normalizeError 将异常归一化为标准服务错误，区分超时与网络错误。 */
function normalizeError(error: unknown, url: string): PublicServiceError {
  if (error instanceof Error && error.name === 'AbortError') {
    return { code: 'timeout', kind: 'NETWORK', message: `Request timeout: ${url}` };
  }
  if (error instanceof TypeError) {
    return { code: 'network', kind: 'NETWORK', message: `Network error: ${url}` };
  }
  return { code: 'unknown', kind: 'NETWORK', message: String(error) };
}

/** request 发起带缓存、去重与有限重试的公开 API 请求。 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const fullUrl = `${API_BASE_URL}${path}`;
  const cacheKey = path;
  const existing = inflight.get(cacheKey);
  if (existing) return existing as Promise<T>;

  const task = (async () => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), CLIENT_TIMEOUT_MS);
    let attempts = 0;
    try {
      while (true) {
        attempts += 1;
        const started = Date.now();
        try {
          const res = await fetch(fullUrl, {
            ...init,
            next: { revalidate: PORTAL_REVALIDATE_SECONDS },
            signal: controller.signal,
            cache: 'force-cache',
          } as RequestInit);
          if (!res.ok) {
            if (res.status === 404) {
              throw Object.assign(new Error('not_found'), { code: 'not_found', status: 404 });
            }
            if (res.status === 400) {
              throw Object.assign(new Error('bad_request'), { code: 'bad_request', status: 400 });
            }
            let body: { code?: string; error?: string } | null = null;
            try {
              body = (await res.json()) as { code?: string; error?: string };
            } catch {
              body = null;
            }
            const code = body?.code ?? `http_${res.status}`;
            const err = new Error(body?.error ?? `HTTP ${res.status}`);
            Object.assign(err, { code, status: res.status, kind: 'SERVER' as RetryableError });
            throw err;
          }
          return (await res.json()) as T;
        } catch (err) {
          const status = (err as { status?: number }).status;
          // 4xx 为客户端错误，直接抛出不重试。
          if (status && status >= 400 && status < 500) throw err;
          if (attempts < RETRY_DELAYS.length + 1) {
            const delay = RETRY_DELAYS[attempts - 1] ?? 500;
            await new Promise((resolve) => setTimeout(resolve, delay));
            continue;
          }
          throw normalizeError(err, fullUrl);
        } finally {
          if (Date.now() - started > CLIENT_TIMEOUT_MS) controller.abort();
        }
      }
    } finally {
      clearTimeout(timer);
      inflight.delete(cacheKey);
    }
  })();

  inflight.set(cacheKey, task);
  return task;
}

/** PagedResult 表示分页数据与分页信息的聚合结果。 */
export interface PagedResult<T> {
  items: T[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

/** fetchPublicArticles 获取公开文章列表。 */
export async function fetchPublicArticles(params: {
  page?: number;
  pageSize?: number;
  category?: string;
  keyword?: string;
  sort?: 'latest' | 'popular' | 'featured';
}): Promise<PublicListResponse<PublicArticleListItem>> {
  return request<PublicListResponse<PublicArticleListItem>>(
    `/api/public/articles${buildQuery({ ...params, pageSize: params.pageSize ?? DEFAULT_PAGE_SIZE })}`,
  );
}

/** fetchPublicArticle 获取公开文章详情。 */
export async function fetchPublicArticle(id: number): Promise<PublicArticleDetail> {
  const data = await request<{ item: PublicArticleDetail }>(`/api/public/articles/${id}`);
  return data.item;
}

/** fetchPublicImages 获取公开图片列表。 */
export async function fetchPublicImages(params: {
  page?: number;
  pageSize?: number;
  category?: string;
  keyword?: string;
  sort?: 'latest' | 'popular' | 'featured';
}): Promise<PublicListResponse<PublicFileListItem>> {
  return request<PublicListResponse<PublicFileListItem>>(
    `/api/public/images${buildQuery({ ...params, pageSize: params.pageSize ?? DEFAULT_PAGE_SIZE })}`,
  );
}

/** fetchPublicResources 获取公开资源列表。 */
export async function fetchPublicResources(params: {
  page?: number;
  pageSize?: number;
  category?: string;
  keyword?: string;
  sort?: 'latest' | 'popular' | 'featured';
}): Promise<PublicListResponse<PublicFileListItem>> {
  return request<PublicListResponse<PublicFileListItem>>(
    `/api/public/resources${buildQuery({ ...params, pageSize: params.pageSize ?? DEFAULT_PAGE_SIZE })}`,
  );
}

/** fetchPublicCategories 获取公开分类列表。 */
export async function fetchPublicCategories(): Promise<PublicCategory[]> {
  const data = await request<{ items: PublicCategory[] }>('/api/public/categories');
  return data.items;
}

/** fetchPublicSiteSummary 获取站点聚合概览数据。 */
export async function fetchPublicSiteSummary(): Promise<PublicSiteSummary> {
  return request<PublicSiteSummary>('/api/public/site-summary');
}

/** fetchPublicSearch 获取聚合搜索结果。 */
export async function fetchPublicSearch(params: {
  keyword: string;
  page?: number;
  pageSize?: number;
  type?: 'articles' | 'images' | 'resources';
}): Promise<PublicSearchResult> {
  return request<PublicSearchResult>(
    `/api/public/search${buildQuery({ ...params, pageSize: params.pageSize ?? 12 })}`,
  );
}

/** publicPreviewUrl 生成公开文件的预览地址。 */
export function publicPreviewUrl(id: number): string {
  return `${API_BASE_URL}/api/public/files/${id}/preview`;
}

/** publicThumbnailUrl 生成公开文件的缩略图地址。 */
export function publicThumbnailUrl(id: number): string {
  return `${API_BASE_URL}/api/public/files/${id}/thumbnail`;
}

/** publicDownloadUrl 生成公开文件的下载地址。 */
export function publicDownloadUrl(id: number): string {
  return `${API_BASE_URL}/api/public/files/${id}/download`;
}