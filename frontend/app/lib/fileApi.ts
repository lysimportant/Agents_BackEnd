import { API_BASE_URL } from './constants';
import { requestWithSession } from './api';
import type { ManagedFile } from '../types/admin';

async function parseApiError(response: Response, fallback: string) {
  try {
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

export async function readTextFileContent(fileId: number) {
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/preview`, {
    // 文本内容刚保存后必须读取服务器最新字节，不能使用浏览器预览缓存。
    cache: 'no-store',
  });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '读取文本内容失败'));
  }
  return response.text();
}

export async function updateFileMetadata(
  fileId: number,
  data: Pick<ManagedFile, 'displayName' | 'category' | 'description'> & { isPrivate?: boolean },
) {
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

export async function updateTextFileContent(fileId: number, content: string) {
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

export async function permanentlyDeleteFile(fileId: number) {
  const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/permanent`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error(await parseApiError(response, '永久删除文件失败'));
  }
  return response.json() as Promise<{ message: string; file: ManagedFile }>;
}
