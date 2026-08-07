/**
 * 门户导航封装，导出基于 next-intl 的 Link、useRouter、usePathname 等能力。
 * 使客户端导航自动携带语言前缀并保持当前语言一致。
 */
import { createNavigation } from 'next-intl/navigation';
import { routing } from './i18n/routing';

/** 导出 next-intl 提供的语言感知导航能力。 */
export const { Link, redirect, usePathname, useRouter, getPathname } = createNavigation(routing);