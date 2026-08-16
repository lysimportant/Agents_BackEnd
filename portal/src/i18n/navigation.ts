import { createNavigation } from 'next-intl/navigation';
import { routing } from './routing';

/**
 * 基于语言路由导出的导航原语，页面与客户端组件统一使用这里的 Link 与跳转。
 */
export const { Link, redirect, usePathname, useRouter, getPathname } =
  createNavigation(routing);
