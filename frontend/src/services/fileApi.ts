import { API_BASE_URL } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';
import type { ManagedFile } from '@/src/types/admin';

/** parseApiError 解析对应业务数据。 */
async function parseApiError(response: Response, fallback: string) {
  try {
    /** payload 保存请求载荷。 */
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

/** 读取已授权文件在服务端保存的最新文本内容。 */
export async function readTextFileContent(fileId: number) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/preview`, {
    // 文本内容刚保存后必须读取服务器最新字节，不能使用浏览器预览缓存。
    cache: 'no-store',
  });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '读取文本内容失败'));
  }
  return response.text();
}

/** 以 Blob 形式读取已授权预览，同时不暴露物理存储路径。 */
export async function readFilePreviewBlob(file: Pick<ManagedFile, 'id' | 'previewUrl'>) {
  /** previewPath 保存预览路径。 */
  const previewPath = file.previewUrl || `/api/files/${file.id}/preview`;
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}${previewPath}`, {
    cache: 'no-store',
  });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '读取文件预览失败'));
  }
  return response.blob();
}

/** 更新受管文件可编辑的展示和隐私元数据。 */
export async function updateFileMetadata(
  fileId: number,
  data: Pick<ManagedFile, 'displayName' | 'category' | 'description'> & { isPrivate?: boolean },
) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      displayName: data.displayName,
      category: data.category,
      description: data.description,
      isPrivate: Boolean(data.isPrivate),
    }),
  });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '保存文件信息失败'));
  }
  return response.json() as Promise<ManagedFile>;
}

/** 替换可编辑文本文件的内容。 */
export async function updateTextFileContent(fileId: number, content: string) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/content`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '保存文本内容失败'));
  }
  return response.json() as Promise<ManagedFile>;
}

/**
 * 在界面明确确认后永久删除已经软删除的文件。
 *
 * @param fileId 回收站内文件的唯一标识。
 * @returns 服务端以 204 No Content 确认删除完成后结束，不读取响应体。
 * @throws 请求失败或服务端返回非成功状态时抛出用户可见错误。
 */
export async function permanentlyDeleteFile(fileId: number): Promise<void> {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/permanent`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '永久删除文件失败'));
  }
  // 永久删除成功时后端返回 204 No Content，解析 JSON 会把成功结果误判为异常。
  return;
}
