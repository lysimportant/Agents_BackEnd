import type { VisitorAnalyticsRange, VisitorAnalyticsResponse } from '@/src/types/admin';
import { API_BASE_URL } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';

/** 获取按最新优先排列的一页访问记录及对应图表聚合数据。 */
export async function fetchVisitorAnalytics(options: {
  range: VisitorAnalyticsRange;
  page: number;
  pageSize: number;
  keyword?: string;
  statusCode?: number;
}) {
  /** params 保存变量 params。 */
  const params = new URLSearchParams({
    range: options.range,
    page: String(options.page),
    pageSize: String(options.pageSize),
  });
  if (options.keyword?.trim()) params.set('keyword', options.keyword.trim());
  if (options.statusCode) params.set('statusCode', String(options.statusCode));
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/visitor-analytics?${params.toString()}`);
  if (!response.ok) {
    /** message 保存消息。 */
    let message = '加载访问分析失败';
    try {
      /** payload 保存请求载荷。 */
      const payload = await response.json() as { error?: string };
      message = payload.error || message;
    } catch {
      // 非 JSON 错误继续使用本地兜底文案。
    }
    throw new Error(message);
  }
  return await response.json() as VisitorAnalyticsResponse;
}
