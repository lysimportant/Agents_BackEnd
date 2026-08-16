import { redirect } from 'next/navigation';
import { resolveRequestLocale } from '@/utils/locale';

/**
 * 根路径 / 根据已验证的语言偏好跳转到对应语言首页，不存在合法偏好时回退默认语言。
 */
export default async function RootPage() {
  const locale = await resolveRequestLocale();
  redirect('/' + locale);
}
