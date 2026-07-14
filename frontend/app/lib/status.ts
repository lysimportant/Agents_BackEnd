import type { Article } from '../types/admin';

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

export function getArticleStatusClass(status: Article['status']) {
  if (status === '已发布') {
    return 'online';
  }
  if (status === '待审核') {
    return 'warning';
  }
  return 'offline';
}
