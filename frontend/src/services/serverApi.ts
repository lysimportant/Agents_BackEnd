import { API_BASE_URL } from '@/src/config/constants';
import type { ServerMetrics } from '@/src/types/admin';
import { requestWithSession } from '@/src/services/api';

/** parseServerError 从服务器管理接口响应中提取可显示错误。 */
async function parseServerError(response: Response, fallback: string) {
  try {
    /** payload 保存服务器错误响应载荷。 */
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

/** fetchServerMetrics 读取后端实际运行环境的服务器资源快照。 */
export async function fetchServerMetrics() {
  /** response 保存服务器资源接口响应。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/server/metrics`, { cache: 'no-store' });
  if (!response.ok) {
    throw new Error(await parseServerError(response, '加载服务器资源失败'));
  }
  return await response.json() as ServerMetrics;
}

/** serverTerminalWebSocketURL 返回携带当前会话 Cookie 的 SSH 终端 WebSocket 地址。 */
export function serverTerminalWebSocketURL() {
  /** url 保存由后端基地址转换得到的终端 WebSocket 地址。 */
  const url = new URL('/api/server/terminal', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}
