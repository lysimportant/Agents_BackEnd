'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import dynamic from 'next/dynamic';
import type { EChartsOption } from 'echarts';
import CountUp from 'react-countup';
import {
  ApartmentOutlined,
  FileDoneOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { Button, Progress, Tag } from 'antd';
import { API_BASE_URL } from '@/src/config/constants';
import {
  ADMIN_THEME_EVENT,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  type AdminTheme,
} from '@/src/theme/themes';

/** ReactECharts 保存按需加载的客户端图表组件。 */
const ReactECharts = dynamic(() => import('echarts-for-react'), { ssr: false });

type BusinessResourcesPageProps = {
  /** usersCount、activeUsers 表示用户总数与可登录账号数。 */
  usersCount: number;
  activeUsers: number;
  /** menusCount、enabledMenus 表示菜单总数与启用菜单数。 */
  menusCount: number;
  enabledMenus: number;
  /** articlesCount、publishedArticles 表示文章总数与已发布文章数。 */
  articlesCount: number;
  publishedArticles: number;
  /** isLoading 表示平台业务数据是否正在刷新。 */
  isLoading: boolean;
  /** onRefresh 重新加载当前账号可访问的业务数据。 */
  onRefresh: () => void;
};

type BusinessStatCardProps = {
  /** label、value、note 表示指标名称、数值和补充说明。 */
  label: string;
  value: number;
  note: string;
  /** icon 表示指标图标。 */
  icon: ReactNode;
  /** tone 表示指标提示色调。 */
  tone: 'primary' | 'success' | 'warning' | 'accent';
};

/** BusinessResourcesPage 独立展示平台业务资源总量、有效量和可用率。 */
export function BusinessResourcesPage({
  usersCount,
  activeUsers,
  menusCount,
  enabledMenus,
  articlesCount,
  publishedArticles,
  isLoading,
  onRefresh,
}: BusinessResourcesPageProps) {
  /** theme 保存当前管理端主题。 */
  const theme = useBusinessResourcesTheme();
  /** totalResources 表示平台三类业务资源总数。 */
  const totalResources = usersCount + menusCount + articlesCount;
  /** accountRatio、menuRatio、articleRatio 表示三类资源可用率。 */
  const accountRatio = getRatio(activeUsers, usersCount);
  const menuRatio = getRatio(enabledMenus, menusCount);
  const articleRatio = getRatio(publishedArticles, articlesCount);
  /** overviewOption 保存总量与有效资源柱状图。 */
  const overviewOption = useMemo<EChartsOption>(() => createOverviewOption(theme, {
    total: [usersCount, menusCount, articlesCount],
    available: [activeUsers, enabledMenus, publishedArticles],
  }), [activeUsers, articlesCount, enabledMenus, menusCount, publishedArticles, theme, usersCount]);
  /** compositionOption 保存业务资源构成环形图。 */
  const compositionOption = useMemo<EChartsOption>(() => createCompositionOption(theme, [
    { name: '用户账号', value: usersCount },
    { name: '菜单节点', value: menusCount },
    { name: '文章内容', value: articlesCount },
  ]), [articlesCount, menusCount, theme, usersCount]);

  return (
    <div className="dashboard-page business-resources-page">
      <section className="dashboard-hero">
        <div>
          <p className="dashboard-eyebrow">平台概览</p>
          <h1>业务资源与可用状态</h1>
          <p>用户账号、菜单节点与文章内容的实时数量和有效状态</p>
        </div>
        <div className="dashboard-hero-actions">
          <Tag>{API_BASE_URL}</Tag>
          <Button type="primary" icon={<ReloadOutlined spin={isLoading} />} onClick={onRefresh} disabled={isLoading}>
            {isLoading ? '正在刷新' : '刷新数据'}
          </Button>
        </div>
      </section>

      <section className="dashboard-stat-grid" aria-label="平台核心指标">
        <BusinessStatCard label="用户总数" value={usersCount} note={`${activeUsers} 个账号可登录`} icon={<TeamOutlined />} tone="primary" />
        <BusinessStatCard label="可登录账号" value={activeUsers} note={`账号可用率 ${accountRatio}%`} icon={<SafetyCertificateOutlined />} tone="success" />
        <BusinessStatCard label="启用菜单" value={enabledMenus} note={`共 ${menusCount} 个菜单节点`} icon={<ApartmentOutlined />} tone="warning" />
        <BusinessStatCard label="已发布文章" value={publishedArticles} note={`共 ${articlesCount} 篇内容`} icon={<FileDoneOutlined />} tone="accent" />
      </section>

      <section className="dashboard-chart-grid" aria-label="平台资源图表">
        <BusinessChartPanel eyebrow="资源状态" title="总量与有效资源" tag="实时快照" option={overviewOption} ariaLabel="用户、菜单和文章的总量与有效资源柱状图" />
        <BusinessChartPanel eyebrow="资源构成" title="平台数据分布" tag={formatInteger(totalResources)} option={compositionOption} ariaLabel="平台资源构成环形图" />
      </section>

      <section className="dashboard-panel dashboard-availability-panel" aria-label="平台可用率">
        <BusinessProgress label="账号可用率" value={accountRatio} color={theme.palette.charts[0]} />
        <BusinessProgress label="菜单启用率" value={menuRatio} color={theme.palette.charts[1]} />
        <BusinessProgress label="文章发布率" value={articleRatio} color={theme.palette.charts[2]} />
      </section>
    </div>
  );
}

/** BusinessStatCard 展示业务资源核心指标。 */
function BusinessStatCard({ label, value, note, icon, tone }: BusinessStatCardProps) {
  return (
    <article className={`dashboard-stat-card is-${tone}`}>
      <div className="dashboard-stat-icon">{icon}</div>
      <div><span>{label}</span><strong><CountUp end={value} duration={0.7} preserveValue separator="," /></strong><small title={note}>{note}</small></div>
    </article>
  );
}

/** BusinessChartPanel 展示业务资源图表及其当前摘要。 */
function BusinessChartPanel({ eyebrow, title, tag, option, ariaLabel }: { eyebrow: string; title: string; tag: string; option: EChartsOption; ariaLabel: string }) {
  return (
    <article className="dashboard-panel dashboard-chart-panel">
      <div className="dashboard-panel-heading"><div><p>{eyebrow}</p><h2>{title}</h2></div><Tag>{tag}</Tag></div>
      <ReactECharts option={option} notMerge lazyUpdate opts={{ renderer: 'svg' }} className="dashboard-chart" aria-label={ariaLabel} />
    </article>
  );
}

/** BusinessProgress 展示一类平台资源的有效比例。 */
function BusinessProgress({ label, value, color }: { label: string; value: number; color: string }) {
  return <div className="dashboard-progress-row"><div><span>{label}</span><strong>{value}%</strong></div><Progress percent={value} showInfo={false} strokeColor={color} railColor="var(--surface-active)" /></div>;
}

/** useBusinessResourcesTheme 返回随管理端主题切换的图表配色。 */
function useBusinessResourcesTheme() {
  /** themeId、setThemeId 保存当前主题标识。 */
  const [themeId, setThemeId] = useState(DEFAULT_THEME_ID);
  useEffect(() => {
    /** syncTheme 从文档根节点读取当前管理端主题。 */
    const syncTheme = () => setThemeId(resolveThemeId(document.documentElement.dataset.theme));
    syncTheme();
    window.addEventListener(ADMIN_THEME_EVENT, syncTheme);
    return () => window.removeEventListener(ADMIN_THEME_EVENT, syncTheme);
  }, []);
  return useMemo(() => getAdminTheme(themeId), [themeId]);
}

/** createOverviewOption 创建业务资源总量与有效量柱状图。 */
function createOverviewOption(theme: AdminTheme, values: { total: number[]; available: number[] }): EChartsOption {
  /** axisStyle 保存坐标轴文字样式。 */
  const axisStyle = { color: theme.palette.textSecondary };
  return {
    animationDuration: 500,
    color: [theme.palette.charts[0], theme.palette.charts[1]],
    tooltip: { trigger: 'axis', confine: true, backgroundColor: theme.palette.panel, borderColor: theme.palette.border, textStyle: { color: theme.palette.text }, formatter: '{b}<br/>{a0}：{c0}<br/>{a1}：{c1}' },
    legend: { top: 2, right: 0, itemWidth: 10, itemHeight: 10, textStyle: axisStyle },
    grid: { top: 54, right: 12, bottom: 30, left: 42, containLabel: true },
    xAxis: { type: 'category', data: ['用户', '菜单', '文章'], axisTick: { show: false }, axisLine: { lineStyle: { color: theme.palette.border } }, axisLabel: axisStyle },
    yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } }, axisLabel: axisStyle },
    series: [
      { name: '资源总量', type: 'bar', data: values.total, barMaxWidth: 34, itemStyle: { borderRadius: [4, 4, 0, 0] } },
      { name: '有效资源', type: 'bar', data: values.available, barMaxWidth: 34, itemStyle: { borderRadius: [4, 4, 0, 0] } },
    ],
  };
}

/** createCompositionOption 创建业务资源构成环形图。 */
function createCompositionOption(theme: AdminTheme, resources: Array<{ name: string; value: number }>): EChartsOption {
  /** hasResources 表示是否存在可展示资源。 */
  const hasResources = resources.some((resource) => resource.value > 0);
  return {
    animationDuration: 500,
    color: [...theme.palette.charts],
    tooltip: { trigger: 'item', confine: true, backgroundColor: theme.palette.panel, borderColor: theme.palette.border, textStyle: { color: theme.palette.text }, formatter: '{b}<br/>{c} 项 · {d}%' },
    legend: { bottom: 0, left: 'center', icon: 'circle', itemWidth: 9, itemHeight: 9, textStyle: { color: theme.palette.textSecondary } },
    series: [{ name: '资源构成', type: 'pie', radius: ['48%', '70%'], center: ['50%', '43%'], itemStyle: { borderColor: theme.palette.panel, borderWidth: 3, borderRadius: 4 }, label: { show: false }, data: hasResources ? resources : [{ name: '暂无资源', value: 1, itemStyle: { color: theme.palette.active } }] }],
  };
}

/** getRatio 返回限制在零到一百的整数百分比。 */
function getRatio(value: number, total: number) {
  if (total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((value / total) * 100)));
}

/** formatInteger 将业务资源数格式化为千位分隔文本。 */
function formatInteger(value: number) {
  return Number.isFinite(value) ? Math.max(0, Math.round(value)).toLocaleString('zh-CN') : '0';
}
