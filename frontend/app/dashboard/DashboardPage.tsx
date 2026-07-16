'use client';

import dynamic from 'next/dynamic';
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import type { EChartsOption } from 'echarts';
import CountUp from 'react-countup';
import {
  ApartmentOutlined,
  CheckCircleOutlined,
  FileDoneOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { Button, Progress, Tag } from 'antd';
import { API_BASE_URL } from '../lib/constants';
import {
  ADMIN_THEME_EVENT,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  type AdminTheme,
} from '../theme/themes';

const ReactECharts = dynamic(() => import('echarts-for-react'), { ssr: false });

type DashboardPageProps = {
  usersCount: number;
  activeUsers: number;
  menusCount: number;
  enabledMenus: number;
  articlesCount: number;
  publishedArticles: number;
  isLoading: boolean;
  onRefresh: () => void;
};

type StatCardProps = {
  label: string;
  value: number;
  note: string;
  icon: ReactNode;
  tone: 'primary' | 'success' | 'warning' | 'accent';
};

export function DashboardPage({
  usersCount,
  activeUsers,
  menusCount,
  enabledMenus,
  articlesCount,
  publishedArticles,
  isLoading,
  onRefresh,
}: DashboardPageProps) {
  const theme = useDashboardTheme();
  const totalResources = usersCount + menusCount + articlesCount;
  const enabledRatio = getRatio(enabledMenus, menusCount);
  const publishedRatio = getRatio(publishedArticles, articlesCount);
  const accountRatio = getRatio(activeUsers, usersCount);

  const overviewOption = useMemo<EChartsOption>(
    () => createOverviewOption(theme, {
      total: [usersCount, menusCount, articlesCount],
      available: [activeUsers, enabledMenus, publishedArticles],
    }),
    [activeUsers, articlesCount, enabledMenus, menusCount, publishedArticles, theme, usersCount],
  );

  const compositionOption = useMemo<EChartsOption>(
    () => createCompositionOption(theme, [
      { name: '用户账号', value: usersCount },
      { name: '菜单节点', value: menusCount },
      { name: '文章内容', value: articlesCount },
    ]),
    [articlesCount, menusCount, theme, usersCount],
  );

  return (
    <div className="dashboard-page">
      <section className="dashboard-hero">
        <div>
          <p className="dashboard-eyebrow">运营概览</p>
          <h1>数据运营工作台</h1>
          <p>汇总账号、权限菜单与内容资源的当前状态，帮助你快速识别需要处理的项目。</p>
        </div>
        <Button type="primary" icon={<ReloadOutlined spin={isLoading} />} onClick={onRefresh} disabled={isLoading}>
          {isLoading ? '正在同步' : '刷新数据'}
        </Button>
      </section>

      <section className="dashboard-stat-grid" aria-label="核心指标">
        <DashboardStatCard label="用户总数" value={usersCount} note={`${activeUsers} 个账号可登录`} icon={<TeamOutlined />} tone="primary" />
        <DashboardStatCard label="可登录账号" value={activeUsers} note={`账号可用率 ${accountRatio}%`} icon={<SafetyCertificateOutlined />} tone="success" />
        <DashboardStatCard label="启用菜单" value={enabledMenus} note={`共 ${menusCount} 个菜单节点`} icon={<ApartmentOutlined />} tone="warning" />
        <DashboardStatCard label="已发布文章" value={publishedArticles} note={`共 ${articlesCount} 篇内容`} icon={<FileDoneOutlined />} tone="accent" />
      </section>

      <section className="dashboard-chart-grid" aria-label="资源图表">
        <article className="dashboard-panel dashboard-overview-panel">
          <div className="dashboard-panel-heading">
            <div>
              <p>资源状态</p>
              <h2>总量与有效资源</h2>
            </div>
            <Tag color="processing">实时快照</Tag>
          </div>
          <ReactECharts
            key={`overview-${theme.id}`}
            option={overviewOption}
            notMerge
            lazyUpdate
            opts={{ renderer: 'svg' }}
            className="dashboard-chart"
            aria-label="用户、菜单和文章的总量与有效资源柱状图"
          />
        </article>

        <article className="dashboard-panel dashboard-composition-panel">
          <div className="dashboard-panel-heading">
            <div>
              <p>资源构成</p>
              <h2>平台数据分布</h2>
            </div>
            <strong><CountUp end={totalResources} duration={1.1} preserveValue /></strong>
          </div>
          <ReactECharts
            key={`composition-${theme.id}`}
            option={compositionOption}
            notMerge
            lazyUpdate
            opts={{ renderer: 'svg' }}
            className="dashboard-chart"
            aria-label="平台资源构成环形图"
          />
        </article>
      </section>

      <section className="dashboard-bottom-grid">
        <article className="dashboard-panel dashboard-progress-panel">
          <div className="dashboard-panel-heading">
            <div>
              <p>可用性</p>
              <h2>资源启用情况</h2>
            </div>
            <CheckCircleOutlined className="dashboard-heading-icon" />
          </div>
          <DashboardProgress label="账号可用率" value={accountRatio} color={theme.palette.charts[0]} />
          <DashboardProgress label="菜单启用率" value={enabledRatio} color={theme.palette.charts[1]} />
          <DashboardProgress label="文章发布率" value={publishedRatio} color={theme.palette.charts[2]} />
        </article>

        <article className="dashboard-panel dashboard-system-panel">
          <div className="dashboard-panel-heading">
            <div>
              <p>联调信息</p>
              <h2>当前服务配置</h2>
            </div>
            <Tag color="success">会话已连接</Tag>
          </div>
          <dl className="dashboard-info-list">
            <div><dt>后端服务</dt><dd>{API_BASE_URL}</dd></div>
            <div><dt>认证方式</dt><dd>HttpOnly Cookie</dd></div>
            <div><dt>数据范围</dt><dd>当前账号权限</dd></div>
          </dl>
        </article>
      </section>
    </div>
  );
}

function DashboardStatCard({ label, value, note, icon, tone }: StatCardProps) {
  return (
    <article className={`dashboard-stat-card is-${tone}`}>
      <div className="dashboard-stat-icon">{icon}</div>
      <div>
        <span>{label}</span>
        <strong><CountUp end={value} duration={1.15} preserveValue separator="," /></strong>
        <small>{note}</small>
      </div>
    </article>
  );
}

function DashboardProgress({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="dashboard-progress-row">
      <div><span>{label}</span><strong>{value}%</strong></div>
      <Progress percent={value} showInfo={false} strokeColor={color} railColor="var(--surface-active)" />
    </div>
  );
}

function useDashboardTheme() {
  const [themeId, setThemeId] = useState(DEFAULT_THEME_ID);

  useEffect(() => {
    const syncTheme = () => setThemeId(resolveThemeId(document.documentElement.dataset.theme));
    syncTheme();
    window.addEventListener(ADMIN_THEME_EVENT, syncTheme);
    return () => window.removeEventListener(ADMIN_THEME_EVENT, syncTheme);
  }, []);

  return useMemo(() => getAdminTheme(themeId), [themeId]);
}

function getRatio(value: number, total: number) {
  if (total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((value / total) * 100)));
}

function createOverviewOption(
  theme: AdminTheme,
  values: { total: number[]; available: number[] },
): EChartsOption {
  const axisStyle = { color: theme.palette.textSecondary };
  return {
    animationDuration: 700,
    color: [theme.palette.charts[0], theme.palette.charts[1]],
    tooltip: {
      trigger: 'axis',
      confine: true,
      renderMode: 'html',
      extraCssText: 'pointer-events:none;max-width:220px;',
      backgroundColor: theme.palette.panel,
      borderColor: theme.palette.border,
      textStyle: { color: theme.palette.text },
      axisPointer: { type: 'line', lineStyle: { color: theme.palette.primary, width: 1, type: 'dashed' } },
      formatter: '{b}<br/>{a0}：{c0}<br/>{a1}：{c1}',
      position: (point, _params, _dom, _rect, size) => {
        const tooltipWidth = size.contentSize[0];
        const viewWidth = size.viewSize[0];
        // Keep the detail panel in the chart's header band so it never covers
        // the bars that the pointer is inspecting.
        return [Math.max(8, Math.min(viewWidth - tooltipWidth - 8, point[0] - tooltipWidth / 2)), 8];
      },
    },
    legend: {
      top: 2,
      right: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: axisStyle,
    },
    grid: { top: 54, right: 12, bottom: 30, left: 42, containLabel: true },
    xAxis: {
      type: 'category',
      data: ['用户', '菜单', '文章'],
      axisTick: { show: false },
      axisLine: { lineStyle: { color: theme.palette.border } },
      axisLabel: axisStyle,
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } },
      axisLabel: axisStyle,
    },
    series: [
      { name: '资源总量', type: 'bar', data: values.total, barMaxWidth: 34, itemStyle: { borderRadius: [4, 4, 0, 0] } },
      { name: '有效资源', type: 'bar', data: values.available, barMaxWidth: 34, itemStyle: { borderRadius: [4, 4, 0, 0] } },
    ],
  };
}

function createCompositionOption(
  theme: AdminTheme,
  data: Array<{ name: string; value: number }>,
): EChartsOption {
  const hasData = data.some((item) => item.value > 0);
  return {
    animationDuration: 750,
    color: [...theme.palette.charts],
    tooltip: {
      trigger: 'item',
      confine: true,
      renderMode: 'html',
      extraCssText: 'pointer-events:none;max-width:220px;',
      backgroundColor: theme.palette.panel,
      borderColor: theme.palette.border,
      textStyle: { color: theme.palette.text },
      formatter: '{b}<br/>{c} 项 · {d}%',
    },
    legend: {
      bottom: 0,
      left: 'center',
      icon: 'circle',
      itemWidth: 9,
      itemHeight: 9,
      textStyle: { color: theme.palette.textSecondary },
    },
    series: [
      {
        name: '资源构成',
        type: 'pie',
        radius: ['48%', '70%'],
        center: ['50%', '43%'],
        avoidLabelOverlap: true,
        itemStyle: { borderColor: theme.palette.panel, borderWidth: 3, borderRadius: 4 },
        label: { show: false },
        emphasis: { label: { show: true, color: theme.palette.text, fontSize: 14, fontWeight: 700 } },
        data: hasData ? data : [{ name: '暂无资源', value: 1, itemStyle: { color: theme.palette.active } }],
      },
    ],
  };
}
