import type { VisitorAnalyticsRange, VisitorAnalyticsResponse } from '@/src/types/admin';
import { API_BASE_URL } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';

export async function fetchVisitorAnalytics(options: {
  range: VisitorAnalyticsRange;
  page: number;
  pageSize: number;
  keyword?: string;
  statusCode?: number;
}) {
  const params = new URLSearchParams({
    range: options.range,
    page: String(options.page),
    pageSize: String(options.pageSize),
  });
  if (options.keyword?.trim()) params.set('keyword', options.keyword.trim());
  if (options.statusCode) params.set('statusCode', String(options.statusCode));
  const response = await requestWithSession(`${API_BASE_URL}/api/visitor-analytics?${params.toString()}`);
  if (!response.ok) {
    let message = '加载访问分析失败';
    try {
      const payload = await response.json() as { error?: string };
      message = payload.error || message;
    } catch {
      // Keep the local fallback for non-JSON errors.
    }
    throw new Error(message);
  }
  return await response.json() as VisitorAnalyticsResponse;
}
