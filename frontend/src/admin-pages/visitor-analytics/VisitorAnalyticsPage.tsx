'use client';

import dynamic from 'next/dynamic';
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Button, Empty, Input, Select, Table, Tag, Tooltip } from 'antd';
import type { EChartsOption } from 'echarts';
import { ReloadOutlined, SafetyCertificateOutlined, SearchOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import CountUp from 'react-countup';
import type { VisitorAccessRecord, VisitorAnalyticsRange, VisitorAnalyticsResponse } from '@/src/types/admin';
import { ADMIN_THEME_EVENT, DEFAULT_THEME_ID, getAdminTheme, resolveThemeId, type AdminTheme } from '@/src/theme/themes';
import styles from './VisitorAnalyticsPage.module.css';

const ReactECharts = dynamic(() => import('echarts-for-react'), { ssr: false });

type VisitorAnalyticsPageProps = {
  data: VisitorAnalyticsResponse | null;
  isLoading: boolean;
  range: VisitorAnalyticsRange;
  keyword: string;
  onRangeChange: (range: VisitorAnalyticsRange) => void;
  onKeywordChange: (keyword: string) => void;
  onRefresh: (page?: number, pageSize?: number) => void;
};

export function VisitorAnalyticsPage({
  data,
  isLoading,
  range,
  keyword,
  onRangeChange,
  onKeywordChange,
  onRefresh,
}: VisitorAnalyticsPageProps) {
  const theme = useVisitorAnalyticsTheme();
  const summary = data?.summary;
  const hasVisitorData = (summary?.totalRequests ?? 0) > 0;
  const timelineOption = useMemo(() => createTimelineOption(theme, summary?.timeline ?? []), [summary?.timeline, theme]);
  const countryOption = useMemo(() => createDimensionOption(theme, summary?.countries ?? [], '国家/地区访问量'), [summary?.countries, theme]);
  const pathOption = useMemo(() => createDimensionOption(theme, summary?.paths ?? [], '访问路径 Top 8'), [summary?.paths, theme]);

  const columns = useMemo<ColumnsType<VisitorAccessRecord>>(() => [
    {
      title: '访问时间', dataIndex: 'createdAt', width: 174,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: 'IP / 代理 IP', key: 'ip', width: 170,
      render: (_value, record) => <div><strong>{record.ip || '未知'}</strong>{record.forwardedIp && <small className={styles.muted}>代理：{record.forwardedIp}</small>}</div>,
    },
    {
      title: '地区 / 网络', key: 'location', width: 180,
      render: (_value, record) => <Tooltip title={record.isp || undefined}><span className={styles.location}>{[record.country, record.region, record.city].filter(Boolean).join(' / ') || '未知地区'}</span></Tooltip>,
    },
    {
      title: '请求', key: 'request', width: 330,
      render: (_value, record) => <div><Tag color={methodColor(record.method)}>{record.method}</Tag><Tooltip title={record.path}><span className={styles.path}>{record.path}</span></Tooltip></div>,
    },
    {
      title: '状态', dataIndex: 'statusCode', width: 76,
      render: (value: number) => <span className={`${styles.status} ${value >= 400 ? styles.statusError : styles.statusOk}`}>{value}</span>,
    },
    {
      title: '耗时', dataIndex: 'durationMs', width: 84,
      render: (value: number) => `${value} ms`,
    },
    {
      title: '设备', key: 'device', width: 160,
      render: (_value, record) => <Tooltip title={`${record.browser} · ${record.os}`}><span>{record.device} · {record.browser}</span></Tooltip>,
    },
    {
      title: '访问者', key: 'visitor', width: 150,
      align: 'center',
      render: (_value, record) => <div className={styles.visitorCell}>{record.authenticated ? <Tag icon={<SafetyCertificateOutlined />} color="success">{record.userName || '已登录'}</Tag> : <Tag>网站访客</Tag>}</div>,
    },
  ], []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    onRefresh(pagination.current || 1, pagination.pageSize);
  };

  return (
    <main className={styles.page}>
      <section className={styles.stats} aria-label="访问核心指标">
        <Stat label="请求总量" value={summary?.totalRequests ?? 0} note={`${rangeLabel(range)} 内`} />
        <Stat label="独立 IP" value={summary?.uniqueIps ?? 0} note="按连接 IP 去重" />
        <Stat label="登录访问" value={summary?.authenticatedRequests ?? 0} note="可关联到后台账号" />
        <Stat label="异常请求" value={summary?.errorRequests ?? 0} note={<span>平均耗时 <CountUp end={summary?.averageDurationMs ?? 0} duration={0.9} preserveValue separator="," /> ms</span>} />
      </section>

      <section className={styles.filters} aria-label="访问分析筛选">
        <div className={styles.filterFields}>
          <Select value={range} aria-label="统计时间范围" onChange={onRangeChange} options={[{ value: '24h', label: '最近 24 小时' }, { value: '7d', label: '最近 7 天' }, { value: '30d', label: '最近 30 天' }]} />
          <Input value={keyword} allowClear placeholder="搜索 IP、地区、路径或 User-Agent" onChange={(event) => onKeywordChange(event.target.value)} onPressEnter={() => onRefresh()} />
          <Button className={styles.filterButton} type="primary" icon={<SearchOutlined />} onClick={() => onRefresh()}>查询</Button>
          <Button className={styles.filterButton} icon={<ReloadOutlined spin={isLoading} />} onClick={() => onRefresh()} disabled={isLoading}>{isLoading ? '正在同步' : '刷新数据'}</Button>
        </div>
        <span className={styles.filterNote}>地区依赖反向代理的 Geo Header，未配置时显示“未知”</span>
      </section>

      <section className={styles.chartGrid} aria-label="访问趋势图表">
        <article className={styles.panel} data-tilt-card="true"><PanelHeading eyebrow="请求趋势" title={`${rangeLabel(range)}访问量`} />{hasVisitorData ? <ReactECharts option={timelineOption} notMerge lazyUpdate opts={{ renderer: 'svg' }} className={styles.chart} aria-label="访问趋势折线图" /> : <Empty className={styles.chartEmpty} image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无访问记录" />}</article>
        <article className={styles.panel} data-tilt-card="true"><PanelHeading eyebrow="全球分布" title="国家/地区 Top 8" />{hasVisitorData ? <ReactECharts option={countryOption} notMerge lazyUpdate opts={{ renderer: 'svg' }} className={styles.chart} aria-label="国家地区访问量柱状图" /> : <Empty className={styles.chartEmpty} image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无访问记录" />}</article>
        <article className={`${styles.panel} ${styles.tablePanel}`} data-tilt-card="true"><PanelHeading eyebrow="热门入口" title="访问路径 Top 8" />{hasVisitorData ? <ReactECharts option={pathOption} notMerge lazyUpdate opts={{ renderer: 'svg' }} className={styles.chart} aria-label="访问路径柱状图" /> : <Empty className={styles.chartEmpty} image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无访问记录" />}</article>
        <article className={`${styles.panel} ${styles.tablePanel}`} data-tilt-card="true"><PanelHeading eyebrow="数据说明" title="隐私与保留策略" /><div className={styles.empty}>这是访问数据的固定策略说明，不代表当前没有数据。IP、User-Agent、来源页等访问元数据默认保留 90 天，仅超级管理员和系统管理员可查看。国家/地区需要由可信反向代理写入 `CF-IPCountry` 等 Header。</div></article>
      </section>

      <section className={`${styles.panel} ${styles.tablePanel}`} data-tilt-card="true" aria-label="访问明细表格">
        <PanelHeading eyebrow="原始明细" title="访问记录" />
        <div className={styles.tableWrap}>
          {data && data.records.length > 0 ? (
            <Table<VisitorAccessRecord> rowKey="id" columns={columns} dataSource={data.records} loading={isLoading} scroll={{ x: 1420 }} pagination={{ current: data.page, pageSize: data.pageSize, total: data.total, showSizeChanger: true, pageSizeOptions: ['10', '20', '30', '50', '100'], showTotal: (total) => `共 ${total} 条` }} onChange={handleTableChange} />
          ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={isLoading ? '正在加载访问记录…' : '暂无访问记录'} />}
        </div>
      </section>
    </main>
  );
}

function Stat({ label, value, note }: { label: string; value: number; note: ReactNode }) {
  return <article className={styles.stat} data-tilt-card="true"><span>{label}</span><strong><CountUp end={value} duration={1.05} preserveValue separator="," /></strong><small>{note}</small></article>;
}

function PanelHeading({ eyebrow, title }: { eyebrow: string; title: string }) {
  return <div className={styles.panelHeading}><div><p>{eyebrow}</p><h2>{title}</h2></div></div>;
}

function useVisitorAnalyticsTheme() {
  const [themeId, setThemeId] = useState(DEFAULT_THEME_ID);
  useEffect(() => {
    const sync = () => setThemeId(resolveThemeId(document.documentElement.dataset.theme));
    sync();
    window.addEventListener(ADMIN_THEME_EVENT, sync);
    return () => window.removeEventListener(ADMIN_THEME_EVENT, sync);
  }, []);
  return useMemo(() => getAdminTheme(themeId), [themeId]);
}

function createTimelineOption(theme: AdminTheme, points: Array<{ label: string; value: number }>): EChartsOption {
  return { animationDuration: 650, color: [theme.palette.charts[0]], tooltip: { trigger: 'axis', confine: true }, grid: { top: 22, right: 20, bottom: 34, left: 42, containLabel: true }, xAxis: { type: 'category', boundaryGap: false, data: points.map((point) => formatBucket(point.label)), axisLabel: { color: theme.palette.textSecondary, hideOverlap: true }, axisLine: { lineStyle: { color: theme.palette.border } } }, yAxis: { type: 'value', minInterval: 1, axisLabel: { color: theme.palette.textSecondary }, splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } } }, series: [{ type: 'line', smooth: true, symbol: 'circle', symbolSize: 6, areaStyle: { color: `${theme.palette.charts[0]}22` }, data: points.map((point) => point.value) }] };
}

function createDimensionOption(theme: AdminTheme, items: Array<{ name: string; value: number }>, seriesName: string): EChartsOption {
  const sorted = [...items].reverse();
  return { animationDuration: 650, color: [theme.palette.charts[1]], tooltip: { trigger: 'axis', confine: true }, grid: { top: 12, right: 18, bottom: 24, left: 86, containLabel: true }, xAxis: { type: 'value', minInterval: 1, axisLabel: { color: theme.palette.textSecondary }, splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } } }, yAxis: { type: 'category', data: sorted.map((item) => item.name), axisLabel: { color: theme.palette.textSecondary, width: 100, overflow: 'truncate' }, axisLine: { show: false } }, series: [{ name: seriesName, type: 'bar', barMaxWidth: 22, data: sorted.map((item) => item.value), itemStyle: { borderRadius: [0, 5, 5, 0] } }] };
}

function rangeLabel(range: VisitorAnalyticsRange) {
  return range === '24h' ? '最近 24 小时' : range === '30d' ? '最近 30 天' : '最近 7 天';
}

function formatBucket(value: string) {
  if (value.length >= 13) return `${value.slice(5, 10)} ${value.slice(11, 13)}时`;
  return value.slice(5, 10);
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '--' : new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(date);
}

function methodColor(method: string) {
  return method === 'GET' ? 'blue' : method === 'POST' ? 'green' : method === 'DELETE' ? 'red' : 'gold';
}
