import type { Article } from '@/src/types/admin';

/** getUserStatusClass 获取对应业务记录。 */
export function getUserStatusClass(status: string) {
  if (status === '在岗') {
    return 'online';
  }
  if (status === '巡检') {
    return 'info';
  }
  if (status === '待命') {
    return 'warning';
  }
  return 'offline';
}

/** getArticleStatusClass 获取对应业务记录。 */
export function getArticleStatusClass(status: Article['status']) {
  if (status === '已发布') {
    return 'online';
  }
  if (status === '待审核') {
    return 'warning';
  }
  return 'offline';
}
