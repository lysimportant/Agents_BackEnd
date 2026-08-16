import { redirect } from 'next/navigation';

/**
 * 首页已删除，图片瀑布流作为门户首页，这里直接跳转到图片页。
 */
export default async function HomePage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  redirect('/' + locale + '/images');
}
