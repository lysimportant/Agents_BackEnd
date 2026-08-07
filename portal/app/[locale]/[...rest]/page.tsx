/**
 * 兜底路由：未匹配到任何页面的地址统一返回 404。
 */
import { notFound } from 'next/navigation';

/** CatchAllPage 对任意未匹配路径触发 404 页面。 */
export default function CatchAllPage() {
  notFound();
}