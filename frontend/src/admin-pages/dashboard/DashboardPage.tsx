'use client';

import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
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
import { Alert, Button, Empty, Progress, Switch, Tag, Tooltip } from 'antd';
import {
  Activity,
  ArrowDownToLine,
  ArrowUpFromLine,
  CircuitBoard,
  Cpu,
  Database,
  Gauge,
  HardDrive,
  MemoryStick,
  Network,
  Server,
  Thermometer,
} from 'lucide-react';
import { API_BASE_URL } from '@/src/config/constants';
import type { ServerMetrics } from '@/src/types/admin';
import {
  ADMIN_THEME_EVENT,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  type AdminTheme,
} from '@/src/theme/themes';

/** ReactECharts 保存按需加载的客户端图表组件。 */
const ReactECharts = dynamic(() => import('echarts-for-react'), { ssr: false });
/** SERVER_SAMPLE_INTERVAL_MS 表示服务器指标自动采样间隔。 */
const SERVER_SAMPLE_INTERVAL_MS = 5_000;
/** SERVER_HISTORY_LIMIT 表示前端保留的最近指标样本数量。 */
const SERVER_HISTORY_LIMIT = 60;

type DashboardPageProps = {
  /** usersCount 表示平台用户总数。 */
  usersCount: number;
  /** activeUsers 表示当前可登录用户数。 */
  activeUsers: number;
  /** menusCount 表示平台菜单总数。 */
  menusCount: number;
  /** enabledMenus 表示已启用菜单数。 */
  enabledMenus: number;
  /** articlesCount 表示平台文章总数。 */
  articlesCount: number;
  /** publishedArticles 表示已发布文章数。 */
  publishedArticles: number;
  /** isLoading 表示平台业务数据是否正在加载。 */
  isLoading: boolean;
  /** serverMetrics 表示后端实际运行环境的资源快照。 */
  serverMetrics: ServerMetrics | null;
  /** isLoadingServerMetrics 表示服务器资源快照是否正在更新。 */
  isLoadingServerMetrics: boolean;
  /** onRefresh 表示同步刷新平台和服务器数据的回调。 */
  onRefresh: () => void;
  /** onRefreshMetrics 表示仅刷新服务器指标的回调。 */
  onRefreshMetrics: () => void | Promise<unknown>;
};

type StatCardProps = {
  /** label 表示指标名称。 */
  label: string;
  /** value 表示指标数值。 */
  value: number;
  /** note 表示指标补充信息。 */
  note: string;
  /** icon 表示指标图标。 */
  icon: ReactNode;
  /** tone 表示指标提示色调。 */
  tone: 'primary' | 'success' | 'warning' | 'accent';
  /** suffix 表示指标数字后的单位。 */
  suffix?: string;
  /** decimals 表示指标数字保留的小数位。 */
  decimals?: number;
};

type MetricHistoryPoint = {
  /** sampledAt 表示样本时间戳。 */
  sampledAt: number;
  /** label 表示图表横轴时间文本。 */
  label: string;
  /** cpu、memory、disk 表示三类资源使用率。 */
  cpu: number;
  memory: number;
  disk: number;
  /** download、upload 表示网络实时速率。 */
  download: number;
  upload: number;
  /** diskRead、diskWrite 表示磁盘实时吞吐。 */
  diskRead: number;
  diskWrite: number;
  /** receivedTotal、sentTotal 表示累计网络字节数。 */
  receivedTotal: number;
  sentTotal: number;
  /** diskReadTotal、diskWriteTotal 表示所有块设备累计读写字节数。 */
  diskReadTotal: number;
  diskWriteTotal: number;
};

/** PartialDeep 将旧版运行时对象的嵌套字段视为可选。 */
type PartialDeep<T> = {
  [Property in keyof T]?: T[Property] extends Array<infer Entry>
    ? Array<PartialDeep<Entry>>
    : T[Property] extends object
      ? PartialDeep<T[Property]>
      : T[Property];
};

/** DashboardPage 展示服务器监控和平台业务概览。 */
export function DashboardPage({
  usersCount,
  activeUsers,
  menusCount,
  enabledMenus,
  articlesCount,
  publishedArticles,
  isLoading,
  serverMetrics: rawServerMetrics,
  isLoadingServerMetrics,
  onRefresh,
  onRefreshMetrics,
}: DashboardPageProps) {
  /** theme 保存当前管理端主题。 */
  const theme = useDashboardTheme();
  /** serverMetrics 将旧版热更新快照补齐为当前完整数据结构。 */
  const serverMetrics = useMemo(() => normalizeServerMetrics(rawServerMetrics), [rawServerMetrics]);
  /** autoRefreshEnabled、setAutoRefreshEnabled 表示是否每五秒自动采样。 */
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true);
  /** metricHistory、setMetricHistory 保存当前页面最近五分钟指标样本。 */
  const [metricHistory, setMetricHistory] = useState<MetricHistoryPoint[]>([]);
  /** refreshMetricsRef 保存最新的服务器指标刷新回调，避免重建定时器。 */
  const refreshMetricsRef = useRef(onRefreshMetrics);
  /** totalResources 表示平台业务资源总数。 */
  const totalResources = usersCount + menusCount + articlesCount;
  /** enabledRatio 表示菜单启用率。 */
  const enabledRatio = getRatio(enabledMenus, menusCount);
  /** publishedRatio 表示文章发布率。 */
  const publishedRatio = getRatio(publishedArticles, articlesCount);
  /** accountRatio 表示账号可用率。 */
  const accountRatio = getRatio(activeUsers, usersCount);
  /** isRefreshing 表示平台数据或服务器指标是否正在刷新。 */
  const isRefreshing = isLoading || isLoadingServerMetrics;
  /** serverScopeLabel 表示资源快照对应的运行边界。 */
  const serverScopeLabel = serverMetrics?.scope === 'container' ? '容器视角' : '宿主机视角';
  /** currentRate 保存最后一个样本计算出的网络和磁盘速率。 */
  const currentRate = metricHistory.at(-1);
  /** healthPresentation 保存健康状态的中文文案和标签颜色。 */
  const healthPresentation = getHealthPresentation(serverMetrics?.health.status);

  useEffect(() => {
    refreshMetricsRef.current = onRefreshMetrics;
  }, [onRefreshMetrics]);

  useEffect(() => {
    if (!autoRefreshEnabled) return undefined;
    /** timer 表示工作台存活期间的服务器采样定时器。 */
    const timer = window.setInterval(() => void refreshMetricsRef.current(), SERVER_SAMPLE_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [autoRefreshEnabled]);

  useEffect(() => {
    if (!serverMetrics) return;
    setMetricHistory((currentHistory) => appendMetricHistory(currentHistory, serverMetrics));
  }, [serverMetrics]);

  /** serverTrendOption 保存服务器资源使用率趋势图。 */
  const serverTrendOption = useMemo<EChartsOption>(
    () => createResourceTrendOption(theme, metricHistory),
    [metricHistory, theme],
  );
  /** networkTrendOption 保存网络实时吞吐趋势图。 */
  const networkTrendOption = useMemo<EChartsOption>(
    () => createThroughputTrendOption(theme, metricHistory, 'network'),
    [metricHistory, theme],
  );
  /** diskTrendOption 保存磁盘实时吞吐趋势图。 */
  const diskTrendOption = useMemo<EChartsOption>(
    () => createThroughputTrendOption(theme, metricHistory, 'disk'),
    [metricHistory, theme],
  );
  /** coreUsageOption 保存每个逻辑核心的即时使用率图。 */
  const coreUsageOption = useMemo<EChartsOption>(
    () => createCoreUsageOption(theme, serverMetrics?.cpu.perCoreUsagePercent ?? []),
    [serverMetrics?.cpu.perCoreUsagePercent, theme],
  );
  /** overviewOption 保存平台资源总量与有效量图表。 */
  const overviewOption = useMemo<EChartsOption>(
    () => createOverviewOption(theme, {
      total: [usersCount, menusCount, articlesCount],
      available: [activeUsers, enabledMenus, publishedArticles],
    }),
    [activeUsers, articlesCount, enabledMenus, menusCount, publishedArticles, theme, usersCount],
  );
  /** compositionOption 保存平台业务资源构成图表。 */
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
          <p className="dashboard-eyebrow">服务器监控</p>
          <h1>{serverMetrics?.hostname || '服务器资源工作台'}</h1>
          <p>{serverMetrics ? `${serverMetrics.platform} ${serverMetrics.platformVersion} · ${serverMetrics.architecture} · ${serverScopeLabel}` : '正在读取后端运行环境的资源状态'}</p>
        </div>
        <div className="dashboard-hero-actions">
          <Tooltip title="每 5 秒采样">
            <span className="dashboard-live-toggle"><Activity size={15} />实时<Switch size="small" checked={autoRefreshEnabled} onChange={setAutoRefreshEnabled} /></span>
          </Tooltip>
          <Button type="primary" icon={<ReloadOutlined spin={isRefreshing} />} onClick={onRefresh} disabled={isRefreshing}>
            {isRefreshing ? '正在同步' : '刷新数据'}
          </Button>
        </div>
      </section>

      <section className="dashboard-stat-grid dashboard-server-stat-grid" aria-label="服务器核心指标">
        <DashboardStatCard label="健康评分" value={serverMetrics?.health.score ?? 0} suffix="分" note={healthPresentation.label} icon={<Gauge size={21} />} tone={serverMetrics?.health.status === 'critical' ? 'warning' : 'success'} />
        <DashboardStatCard label="CPU 使用率" value={serverMetrics?.cpu.usagePercent ?? 0} decimals={1} suffix="%" note={`${serverMetrics?.cpu.physicalCores ?? 0} 物理 / ${serverMetrics?.cpu.logicalCores ?? 0} 逻辑核心`} icon={<Cpu size={21} />} tone="primary" />
        <DashboardStatCard label="内存使用率" value={serverMetrics?.memory.usagePercent ?? 0} decimals={1} suffix="%" note={`${formatBytes(serverMetrics?.memory.usedBytes ?? 0)} / ${formatBytes(serverMetrics?.memory.totalBytes ?? 0)}`} icon={<MemoryStick size={21} />} tone="success" />
        <DashboardStatCard label="主分区使用率" value={serverMetrics?.disk.usagePercent ?? 0} decimals={1} suffix="%" note={`${formatBytes(serverMetrics?.disk.freeBytes ?? 0)} 可用`} icon={<HardDrive size={21} />} tone="warning" />
        <DashboardStatCard label="实时下载" value={toMegabytes(currentRate?.download ?? 0)} decimals={2} suffix=" MB/s" note={`累计 ${formatBytes(serverMetrics?.network.bytesReceived ?? 0)}`} icon={<ArrowDownToLine size={21} />} tone="primary" />
        <DashboardStatCard label="实时上传" value={toMegabytes(currentRate?.upload ?? 0)} decimals={2} suffix=" MB/s" note={`累计 ${formatBytes(serverMetrics?.network.bytesSent ?? 0)}`} icon={<ArrowUpFromLine size={21} />} tone="accent" />
        <DashboardStatCard label="持续运行" value={Math.floor((serverMetrics?.uptimeSeconds ?? 0) / 86400)} suffix="天" note={formatDuration(serverMetrics?.uptimeSeconds ?? 0)} icon={<Server size={21} />} tone="accent" />
        <DashboardStatCard label="活动连接" value={serverMetrics?.network.connections.established ?? 0} note={serverMetrics?.network.connections.available ? `${serverMetrics.network.connections.tcp} TCP · ${serverMetrics.network.connections.udp} UDP` : '当前环境不可读取'} icon={<Network size={21} />} tone="success" />
      </section>

      <section className="dashboard-health-band" aria-label="服务器健康状态">
        <div className="dashboard-health-summary">
          <Tag color={healthPresentation.color}>{healthPresentation.label}</Tag>
          <strong>即时健康检查</strong>
          <span>{serverMetrics?.health.alerts.length ? `${serverMetrics.health.alerts.length} 项需要关注` : '未触发资源阈值'}</span>
        </div>
        <span>采样于 {formatSampleTime(serverMetrics?.sampledAt)}</span>
      </section>

      {!!serverMetrics?.health.alerts.length && (
        <section className="dashboard-alert-list" aria-label="服务器资源告警">
          {serverMetrics.health.alerts.map((alert, index) => (
            <Alert key={`${alert.code}-${index}`} type={alert.severity === 'critical' ? 'error' : 'warning'} showIcon title={alert.title} description={alert.message} />
          ))}
        </section>
      )}

      <section className="dashboard-chart-grid dashboard-monitor-chart-grid" aria-label="服务器资源趋势">
        <DashboardChartPanel eyebrow="最近 5 分钟" title="资源利用率趋势" tag={`${metricHistory.length}/${SERVER_HISTORY_LIMIT} 样本`} option={serverTrendOption} className="dashboard-wide-chart" ariaLabel="CPU、内存和磁盘使用率趋势图" />
        <DashboardChartPanel eyebrow="每秒速率" title="网络吞吐" tag={`↓ ${formatRate(currentRate?.download ?? 0)} · ↑ ${formatRate(currentRate?.upload ?? 0)}`} option={networkTrendOption} ariaLabel="网络下载和上传速率趋势图" />
      </section>

      <section className="dashboard-chart-grid dashboard-monitor-chart-grid" aria-label="处理器与磁盘吞吐">
        <DashboardChartPanel eyebrow="逻辑核心" title="CPU 核心使用率" tag={`${serverMetrics?.cpu.logicalCores ?? 0} 核`} option={coreUsageOption} className="dashboard-wide-chart" ariaLabel="每个逻辑 CPU 核心使用率图" />
        <DashboardChartPanel eyebrow="块设备" title="磁盘实时吞吐" tag={`读 ${formatRate(currentRate?.diskRead ?? 0)} · 写 ${formatRate(currentRate?.diskWrite ?? 0)}`} option={diskTrendOption} ariaLabel="磁盘读取和写入速率趋势图" />
      </section>

      <div className="dashboard-section-heading">
        <div><p>计算与内存</p><h2>处理器、负载和内存明细</h2></div>
        <Tag>{serverMetrics?.cpu.modelName || '等待采样'}</Tag>
      </div>
      <section className="dashboard-detail-grid">
        <article className="dashboard-panel">
          <PanelHeading eyebrow="处理器" title="硬件与累计时间" icon={<Cpu size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="型号" value={serverMetrics?.cpu.modelName || '未提供'} />
            <InfoRow label="厂商" value={serverMetrics?.cpu.vendorId || '未提供'} />
            <InfoRow label="频率" value={serverMetrics?.cpu.frequencyMHz ? `${serverMetrics.cpu.frequencyMHz.toFixed(0)} MHz` : '未提供'} />
            <InfoRow label="缓存" value={serverMetrics?.cpu.cacheSizeKB ? formatBytes(serverMetrics.cpu.cacheSizeKB * 1024) : '未提供'} />
            <InfoRow label="用户态" value={formatDuration(serverMetrics?.cpu.times.userSeconds ?? 0)} />
            <InfoRow label="内核态" value={formatDuration(serverMetrics?.cpu.times.systemSeconds ?? 0)} />
            <InfoRow label="I/O 等待" value={formatDuration(serverMetrics?.cpu.times.ioWaitSeconds ?? 0)} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="系统调度" title="平均负载与进程" icon={<Activity size={20} />} />
          <div className="dashboard-load-values">
            <LoadValue label="1 分钟" value={serverMetrics?.load.load1 ?? 0} />
            <LoadValue label="5 分钟" value={serverMetrics?.load.load5 ?? 0} />
            <LoadValue label="15 分钟" value={serverMetrics?.load.load15 ?? 0} />
          </div>
          <dl className="dashboard-info-list">
            <InfoRow label="进程总数" value={formatInteger(serverMetrics?.load.processTotal ?? 0)} />
            <InfoRow label="运行 / 阻塞" value={`${formatInteger(serverMetrics?.load.processRunning ?? 0)} / ${formatInteger(serverMetrics?.load.processBlocked ?? 0)}`} />
            <InfoRow label="累计创建" value={formatInteger(serverMetrics?.load.processesCreated ?? 0)} />
            <InfoRow label="上下文切换" value={formatInteger(serverMetrics?.load.contextSwitches ?? 0)} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="物理内存" title="容量与缓存" icon={<MemoryStick size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="总容量" value={formatBytes(serverMetrics?.memory.totalBytes ?? 0)} />
            <InfoRow label="可用" value={formatBytes(serverMetrics?.memory.availableBytes ?? 0)} />
            <InfoRow label="完全空闲" value={formatBytes(serverMetrics?.memory.freeBytes ?? 0)} />
            <InfoRow label="缓存 / 缓冲" value={`${formatBytes(serverMetrics?.memory.cachedBytes ?? 0)} / ${formatBytes(serverMetrics?.memory.buffersBytes ?? 0)}`} />
            <InfoRow label="活跃 / 非活跃" value={`${formatBytes(serverMetrics?.memory.activeBytes ?? 0)} / ${formatBytes(serverMetrics?.memory.inactiveBytes ?? 0)}`} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="交换区" title="Swap 使用与换页" icon={<Database size={20} />} />
          <div className="dashboard-meter-block">
            <div><span>使用率</span><strong>{(serverMetrics?.memory.swap.usagePercent ?? 0).toFixed(1)}%</strong></div>
            <Progress percent={clampPercent(serverMetrics?.memory.swap.usagePercent ?? 0)} showInfo={false} strokeColor={theme.palette.charts[3]} railColor="var(--surface-active)" />
          </div>
          <dl className="dashboard-info-list">
            <InfoRow label="已用 / 总量" value={`${formatBytes(serverMetrics?.memory.swap.usedBytes ?? 0)} / ${formatBytes(serverMetrics?.memory.swap.totalBytes ?? 0)}`} />
            <InfoRow label="累计换入" value={formatBytes(serverMetrics?.memory.swap.bytesIn ?? 0)} />
            <InfoRow label="累计换出" value={formatBytes(serverMetrics?.memory.swap.bytesOut ?? 0)} />
          </dl>
        </article>
      </section>

      <div className="dashboard-section-heading">
        <div><p>存储</p><h2>文件系统与块设备 I/O</h2></div>
        <Tag>{serverMetrics?.partitions.length ?? 0} 个分区</Tag>
      </div>
      <section className="dashboard-panel dashboard-table-panel" aria-label="文件系统分区">
        <div className="dashboard-table-scroll">
          <table className="dashboard-metrics-table">
            <thead><tr><th>挂载点</th><th>设备 / 文件系统</th><th>已用</th><th>可用</th><th>使用率</th><th>inode</th></tr></thead>
            <tbody>
              {(serverMetrics?.partitions ?? []).map((partition) => (
                <tr key={`${partition.device}-${partition.path}`}>
                  <td><strong>{partition.path}</strong>{isSameMountPath(partition.path, serverMetrics?.disk.path) && <Tag color="blue">工作目录</Tag>}</td>
                  <td>{partition.device || '未提供'}<small>{partition.fileSystem || '未知文件系统'}</small></td>
                  <td>{formatBytes(partition.usedBytes)}<small>共 {formatBytes(partition.totalBytes)}</small></td>
                  <td>{formatBytes(partition.freeBytes)}</td>
                  <td><UsageMeter value={partition.usagePercent} /></td>
                  <td>{partition.inodesTotal ? <UsageMeter value={partition.inodesUsagePercent} /> : '平台未提供'}</td>
                </tr>
              ))}
              {!serverMetrics?.partitions.length && <tr><td colSpan={6}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无分区数据" /></td></tr>}
            </tbody>
          </table>
        </div>
      </section>
      <section className="dashboard-panel dashboard-table-panel" aria-label="块设备 I/O">
        <div className="dashboard-table-scroll">
          <table className="dashboard-metrics-table">
            <thead><tr><th>块设备</th><th>累计读取</th><th>累计写入</th><th>读操作</th><th>写操作</th><th>进行中 I/O</th></tr></thead>
            <tbody>
              {(serverMetrics?.diskIO ?? []).map((device) => (
                <tr key={device.name}><td><strong>{device.name}</strong></td><td>{formatBytes(device.readBytes)}</td><td>{formatBytes(device.writeBytes)}</td><td>{formatInteger(device.readOperations)}</td><td>{formatInteger(device.writeOperations)}</td><td>{formatInteger(device.ioOperationsInProgress)}</td></tr>
              ))}
              {!serverMetrics?.diskIO.length && <tr><td colSpan={6}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无块设备 I/O 数据" /></td></tr>}
            </tbody>
          </table>
        </div>
      </section>

      <div className="dashboard-section-heading">
        <div><p>网络</p><h2>流量、网卡与连接状态</h2></div>
        <Tag>{serverMetrics?.network.interfaces.length ?? 0} 个网卡</Tag>
      </div>
      <section className="dashboard-detail-grid dashboard-network-summary-grid">
        <article className="dashboard-panel">
          <PanelHeading eyebrow="累计流量" title="数据包与传输质量" icon={<Network size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="接收 / 发送" value={`${formatBytes(serverMetrics?.network.bytesReceived ?? 0)} / ${formatBytes(serverMetrics?.network.bytesSent ?? 0)}`} />
            <InfoRow label="接收包 / 发送包" value={`${formatInteger(serverMetrics?.network.packetsReceived ?? 0)} / ${formatInteger(serverMetrics?.network.packetsSent ?? 0)}`} />
            <InfoRow label="接收 / 发送错误" value={`${formatInteger(serverMetrics?.network.errorsIn ?? 0)} / ${formatInteger(serverMetrics?.network.errorsOut ?? 0)}`} />
            <InfoRow label="接收 / 发送丢包" value={`${formatInteger(serverMetrics?.network.dropsIn ?? 0)} / ${formatInteger(serverMetrics?.network.dropsOut ?? 0)}`} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="套接字" title="连接状态汇总" icon={<CircuitBoard size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="TCP / UDP" value={`${formatInteger(serverMetrics?.network.connections.tcp ?? 0)} / ${formatInteger(serverMetrics?.network.connections.udp ?? 0)}`} />
            <InfoRow label="已建立" value={formatInteger(serverMetrics?.network.connections.established ?? 0)} />
            <InfoRow label="监听" value={formatInteger(serverMetrics?.network.connections.listen ?? 0)} />
            <InfoRow label="TIME_WAIT" value={formatInteger(serverMetrics?.network.connections.timeWait ?? 0)} />
            <InfoRow label="CLOSE_WAIT" value={formatInteger(serverMetrics?.network.connections.closeWait ?? 0)} />
          </dl>
        </article>
      </section>
      <section className="dashboard-panel dashboard-table-panel" aria-label="网络接口">
        <div className="dashboard-table-scroll">
          <table className="dashboard-metrics-table dashboard-network-table">
            <thead><tr><th>网卡</th><th>地址</th><th>状态</th><th>接收 / 发送</th><th>错误 / 丢包</th></tr></thead>
            <tbody>
              {(serverMetrics?.network.interfaces ?? []).map((networkInterface) => (
                <tr key={networkInterface.name}>
                  <td><strong>{networkInterface.name}</strong><small>MTU {networkInterface.mtu}{networkInterface.hardwareAddress ? ` · ${networkInterface.hardwareAddress}` : ''}</small></td>
                  <td>{networkInterface.addresses.length ? networkInterface.addresses.map((address) => <small className="dashboard-address" key={address}>{address}</small>) : '无地址'}</td>
                  <td>{networkInterface.flags.map((flag) => <Tag key={flag}>{flag}</Tag>)}</td>
                  <td>↓ {formatBytes(networkInterface.bytesReceived)}<small>↑ {formatBytes(networkInterface.bytesSent)}</small></td>
                  <td>{formatInteger(networkInterface.errorsIn + networkInterface.errorsOut)} 错误<small>{formatInteger(networkInterface.dropsIn + networkInterface.dropsOut)} 丢包</small></td>
                </tr>
              ))}
              {!serverMetrics?.network.interfaces.length && <tr><td colSpan={5}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无网卡数据" /></td></tr>}
            </tbody>
          </table>
        </div>
      </section>

      <div className="dashboard-section-heading">
        <div><p>运行环境</p><h2>主机、后端进程与传感器</h2></div>
        <Tag>{serverScopeLabel}</Tag>
      </div>
      <section className="dashboard-detail-grid">
        <article className="dashboard-panel">
          <PanelHeading eyebrow="主机" title="系统与虚拟化" icon={<Server size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="系统" value={serverMetrics ? `${serverMetrics.os} / ${serverMetrics.kernelVersion}` : '等待采样'} />
            <InfoRow label="发行版" value={serverMetrics ? `${serverMetrics.platform} ${serverMetrics.platformVersion}` : '等待采样'} />
            <InfoRow label="架构" value={serverMetrics?.architecture || '未提供'} />
            <InfoRow label="虚拟化" value={[serverMetrics?.virtualizationSystem, serverMetrics?.virtualizationRole].filter(Boolean).join(' / ') || '未检测到'} />
            <InfoRow label="启动时间" value={formatSampleTime(serverMetrics?.bootedAt)} />
            <InfoRow label="采样时间" value={formatSampleTime(serverMetrics?.sampledAt)} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="后端进程" title={`PID ${serverMetrics?.process.pid ?? '-'}`} icon={<Database size={20} />} />
          <dl className="dashboard-info-list">
            <InfoRow label="Go / 协程" value={`${serverMetrics?.process.goVersion || '-'} / ${formatInteger(serverMetrics?.process.goroutines ?? 0)}`} />
            <InfoRow label="线程 / 文件句柄" value={`${formatInteger(serverMetrics?.process.threads ?? 0)} / ${formatInteger(serverMetrics?.process.openFileDescriptors ?? 0)}`} />
            <InfoRow label="常驻 / 虚拟内存" value={`${formatBytes(serverMetrics?.process.residentBytes ?? 0)} / ${formatBytes(serverMetrics?.process.virtualBytes ?? 0)}`} />
            <InfoRow label="Go 堆 / 系统" value={`${formatBytes(serverMetrics?.process.heapInUseBytes ?? 0)} / ${formatBytes(serverMetrics?.process.systemBytes ?? 0)}`} />
            <InfoRow label="堆对象 / GC" value={`${formatInteger(serverMetrics?.process.heapObjects ?? 0)} / ${formatInteger(serverMetrics?.process.gcCycles ?? 0)}`} />
            <InfoRow label="进程读 / 写" value={`${formatBytes(serverMetrics?.process.readBytes ?? 0)} / ${formatBytes(serverMetrics?.process.writeBytes ?? 0)}`} />
            <InfoRow label="运行时间" value={formatDuration(serverMetrics?.process.uptimeSeconds ?? 0)} />
          </dl>
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="硬件传感器" title="温度" icon={<Thermometer size={20} />} />
          {serverMetrics?.temperatures.length ? (
            <dl className="dashboard-info-list">
              {serverMetrics.temperatures.map((temperature) => <InfoRow key={temperature.sensorKey} label={temperature.sensorKey} value={`${temperature.temperatureCelsius.toFixed(1)} °C${temperature.criticalCelsius > 0 ? ` / 临界 ${temperature.criticalCelsius.toFixed(0)} °C` : ''}`} />)}
            </dl>
          ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当前环境未提供温度读数" />}
        </article>
        <article className="dashboard-panel">
          <PanelHeading eyebrow="采集完整性" title="平台与权限说明" icon={<Activity size={20} />} />
          {serverMetrics?.collectionWarnings.length ? (
            <ul className="dashboard-warning-list">{serverMetrics.collectionWarnings.map((warning) => <li key={warning}>{warning}</li>)}</ul>
          ) : <Alert type="success" showIcon title="全部监控分组均已采集" />}
        </article>
      </section>

      <div className="dashboard-section-heading">
        <div><p>平台概览</p><h2>业务资源与可用状态</h2></div>
        <Tag>{API_BASE_URL}</Tag>
      </div>
      <section className="dashboard-stat-grid" aria-label="平台核心指标">
        <DashboardStatCard label="用户总数" value={usersCount} note={`${activeUsers} 个账号可登录`} icon={<TeamOutlined />} tone="primary" />
        <DashboardStatCard label="可登录账号" value={activeUsers} note={`账号可用率 ${accountRatio}%`} icon={<SafetyCertificateOutlined />} tone="success" />
        <DashboardStatCard label="启用菜单" value={enabledMenus} note={`共 ${menusCount} 个菜单节点`} icon={<ApartmentOutlined />} tone="warning" />
        <DashboardStatCard label="已发布文章" value={publishedArticles} note={`共 ${articlesCount} 篇内容`} icon={<FileDoneOutlined />} tone="accent" />
      </section>
      <section className="dashboard-chart-grid" aria-label="平台资源图表">
        <DashboardChartPanel eyebrow="资源状态" title="总量与有效资源" tag="实时快照" option={overviewOption} ariaLabel="用户、菜单和文章的总量与有效资源柱状图" />
        <DashboardChartPanel eyebrow="资源构成" title="平台数据分布" tag={formatInteger(totalResources)} option={compositionOption} ariaLabel="平台资源构成环形图" />
      </section>
      <section className="dashboard-panel dashboard-availability-panel" aria-label="平台可用率">
        <DashboardProgress label="账号可用率" value={accountRatio} color={theme.palette.charts[0]} />
        <DashboardProgress label="菜单启用率" value={enabledRatio} color={theme.palette.charts[1]} />
        <DashboardProgress label="文章发布率" value={publishedRatio} color={theme.palette.charts[2]} />
      </section>
    </div>
  );
}

/** DashboardStatCard 展示一个固定尺寸的核心指标。 */
function DashboardStatCard({ label, value, note, icon, tone, suffix, decimals = 0 }: StatCardProps) {
  return (
    <article className={`dashboard-stat-card is-${tone}`}>
      <div className="dashboard-stat-icon">{icon}</div>
      <div><span>{label}</span><strong><CountUp end={value} decimals={decimals} duration={0.7} preserveValue separator="," suffix={suffix} /></strong><small title={note}>{note}</small></div>
    </article>
  );
}

/** DashboardChartPanel 展示统一标题和尺寸的监控图表面板。 */
function DashboardChartPanel({ eyebrow, title, tag, option, ariaLabel, className = '' }: { eyebrow: string; title: string; tag: string; option: EChartsOption; ariaLabel: string; className?: string }) {
  return (
    <article className={`dashboard-panel dashboard-chart-panel ${className}`}>
      <div className="dashboard-panel-heading"><div><p>{eyebrow}</p><h2>{title}</h2></div><Tag>{tag}</Tag></div>
      <ReactECharts option={option} notMerge lazyUpdate opts={{ renderer: 'svg' }} className="dashboard-chart" aria-label={ariaLabel} />
    </article>
  );
}

/** PanelHeading 展示服务器明细面板的标题和图标。 */
function PanelHeading({ eyebrow, title, icon }: { eyebrow: string; title: string; icon: ReactNode }) {
  return <div className="dashboard-panel-heading"><div><p>{eyebrow}</p><h2>{title}</h2></div><span className="dashboard-heading-lucide">{icon}</span></div>;
}

/** InfoRow 展示定义列表中的一项服务器明细。 */
function InfoRow({ label, value }: { label: string; value: ReactNode }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>;
}

/** LoadValue 展示一个平均负载窗口值。 */
function LoadValue({ label, value }: { label: string; value: number }) {
  return <div><span>{label}</span><strong>{value.toFixed(2)}</strong></div>;
}

/** UsageMeter 展示表格内紧凑的百分比和进度条。 */
function UsageMeter({ value }: { value: number }) {
  /** percent 表示限制到有效范围后的百分比。 */
  const percent = clampPercent(value);
  /** statusClass 表示当前使用率对应的视觉状态。 */
  const statusClass = percent >= 92 ? 'is-critical' : percent >= 80 ? 'is-warning' : '';
  return <div className={`dashboard-table-meter ${statusClass}`}><span>{percent.toFixed(1)}%</span><i><b style={{ width: `${percent}%` }} /></i></div>;
}

/** DashboardProgress 展示平台业务资源可用率。 */
function DashboardProgress({ label, value, color }: { label: string; value: number; color: string }) {
  return <div className="dashboard-progress-row"><div><span>{label}</span><strong>{value}%</strong></div><Progress percent={value} showInfo={false} strokeColor={color} railColor="var(--surface-active)" /></div>;
}

/** normalizeServerMetrics 为热更新期间残留的旧版资源快照补齐新增字段。 */
function normalizeServerMetrics(metrics: ServerMetrics | null): ServerMetrics | null {
  if (!metrics) return null;
  /** partialMetrics 允许兼容热更新期间仍保留在内存中的旧版嵌套对象。 */
  const partialMetrics = metrics as unknown as PartialDeep<ServerMetrics>;
  /** primaryDisk 保存补齐文件系统和 inode 字段后的工作分区。 */
  const primaryDisk = {
    device: '', fileSystem: '', inodesTotal: 0, inodesUsed: 0, inodesUsagePercent: 0,
    ...partialMetrics.disk,
  };
  /** cpuTimes 保存补齐后的处理器累计时间。 */
  const cpuTimes = {
    userSeconds: 0, systemSeconds: 0, idleSeconds: 0, ioWaitSeconds: 0,
    irqSeconds: 0, softIrqSeconds: 0, stealSeconds: 0,
    ...partialMetrics.cpu?.times,
  };
  /** swap 保存补齐后的交换区统计。 */
  const swap = {
    totalBytes: 0, usedBytes: 0, freeBytes: 0, usagePercent: 0, bytesIn: 0, bytesOut: 0,
    ...partialMetrics.memory?.swap,
  };
  return {
    ...metrics,
    bootedAt: metrics.bootedAt ?? '',
    virtualizationSystem: metrics.virtualizationSystem ?? '',
    virtualizationRole: metrics.virtualizationRole ?? '',
    cpu: {
      physicalCores: 0, perCoreUsagePercent: [], modelName: '', vendorId: '', frequencyMHz: 0, cacheSizeKB: 0,
      ...partialMetrics.cpu,
      times: cpuTimes,
    },
    load: {
      load1: 0, load5: 0, load15: 0, processTotal: 0, processRunning: 0,
      processBlocked: 0, processesCreated: 0, contextSwitches: 0,
      ...partialMetrics.load,
    },
    memory: {
      freeBytes: 0, cachedBytes: 0, buffersBytes: 0, activeBytes: 0, inactiveBytes: 0,
      ...partialMetrics.memory,
      swap,
    },
    disk: primaryDisk,
    partitions: (partialMetrics.partitions ?? [primaryDisk]).map((partition) => ({
      device: '', fileSystem: '', inodesTotal: 0, inodesUsed: 0, inodesUsagePercent: 0,
      ...partition,
    })),
    diskIO: metrics.diskIO ?? [],
    network: {
      packetsSent: 0, packetsReceived: 0, errorsIn: 0, errorsOut: 0, dropsIn: 0, dropsOut: 0,
      ...partialMetrics.network,
      interfaces: partialMetrics.network?.interfaces as ServerMetrics['network']['interfaces'] ?? [],
      connections: {
        available: false, sampled: 0, truncated: false, tcp: 0, udp: 0,
        established: 0, listen: 0, timeWait: 0, closeWait: 0,
        ...partialMetrics.network?.connections,
      },
    },
    process: {
      threads: 0, cpuUsagePercent: 0, systemBytes: 0, heapInUseBytes: 0, heapObjects: 0,
      gcCycles: 0, residentBytes: 0, virtualBytes: 0, readBytes: 0, writeBytes: 0,
      openFileDescriptors: 0, uptimeSeconds: 0,
      ...partialMetrics.process,
    },
    temperatures: partialMetrics.temperatures as ServerMetrics['temperatures'] ?? [],
    health: partialMetrics.health as ServerMetrics['health'] ?? { status: 'healthy', score: 100, alerts: [] },
    collectionWarnings: partialMetrics.collectionWarnings ?? [],
  } as ServerMetrics;
}

/** useDashboardTheme 返回随管理端主题切换的图表配色。 */
function useDashboardTheme() {
  /** themeId、setThemeId 分别保存主题标识及其更新函数。 */
  const [themeId, setThemeId] = useState(DEFAULT_THEME_ID);
  useEffect(() => {
    /** syncTheme 根据文档根节点同步当前主题。 */
    const syncTheme = () => setThemeId(resolveThemeId(document.documentElement.dataset.theme));
    syncTheme();
    window.addEventListener(ADMIN_THEME_EVENT, syncTheme);
    return () => window.removeEventListener(ADMIN_THEME_EVENT, syncTheme);
  }, []);
  return useMemo(() => getAdminTheme(themeId), [themeId]);
}

/** appendMetricHistory 将新快照转换为速率样本并限制历史长度。 */
function appendMetricHistory(history: MetricHistoryPoint[], metrics: ServerMetrics) {
  /** sampledAt 表示快照解析后的毫秒时间戳。 */
  const sampledAt = new Date(metrics.sampledAt).getTime();
  if (!Number.isFinite(sampledAt)) return history;
  /** previous 表示上一条历史样本。 */
  const previous = history.at(-1);
  if (previous?.sampledAt === sampledAt) return history;
  /** diskReadTotal、diskWriteTotal 表示当前所有块设备累计吞吐。 */
  const { read: diskReadTotal, write: diskWriteTotal } = sumDiskIO(metrics);
  /** elapsedSeconds 表示相邻快照的实际采样间隔。 */
  const elapsedSeconds = previous ? Math.max(0, (sampledAt - previous.sampledAt) / 1000) : 0;
  /** point 保存当前使用率、累计量和通过差值计算的每秒速率。 */
  const point: MetricHistoryPoint = {
    sampledAt,
    label: new Date(sampledAt).toLocaleTimeString('zh-CN', { hour12: false, minute: '2-digit', second: '2-digit' }),
    cpu: clampPercent(metrics.cpu.usagePercent),
    memory: clampPercent(metrics.memory.usagePercent),
    disk: clampPercent(metrics.disk.usagePercent),
    download: calculateRate(metrics.network.bytesReceived, previous?.receivedTotal, elapsedSeconds),
    upload: calculateRate(metrics.network.bytesSent, previous?.sentTotal, elapsedSeconds),
    diskRead: calculateRate(diskReadTotal, previous?.diskReadTotal, elapsedSeconds),
    diskWrite: calculateRate(diskWriteTotal, previous?.diskWriteTotal, elapsedSeconds),
    receivedTotal: metrics.network.bytesReceived,
    sentTotal: metrics.network.bytesSent,
    diskReadTotal,
    diskWriteTotal,
  };
  return [...history, point].slice(-SERVER_HISTORY_LIMIT);
}

/** sumDiskIO 汇总所有块设备累计读写量。 */
function sumDiskIO(metrics: ServerMetrics) {
  return metrics.diskIO.reduce((total, device) => ({ read: total.read + device.readBytes, write: total.write + device.writeBytes }), { read: 0, write: 0 });
}

/** calculateRate 根据相邻累计计数器计算每秒速率，并处理计数器重置。 */
function calculateRate(current: number, previous: number | undefined, elapsedSeconds: number) {
  if (previous === undefined || elapsedSeconds <= 0 || current < previous) return 0;
  return (current - previous) / elapsedSeconds;
}

/** isSameMountPath 判断带或不带末尾分隔符的路径是否属于同一挂载点。 */
function isSameMountPath(left: string, right?: string) {
  /** normalize 将挂载点统一为小写且不带末尾分隔符。 */
  const normalize = (path: string) => path.trim().toLocaleLowerCase().replace(/[\\/]+$/, '');
  return right !== undefined && normalize(left) === normalize(right);
}

/** getRatio 返回限制在 0 到 100 的整数百分比。 */
function getRatio(value: number, total: number) {
  if (total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((value / total) * 100)));
}

/** clampPercent 将浮点使用率限制在图表可展示范围。 */
function clampPercent(value: number) {
  return Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0));
}

/** toMegabytes 将每秒字节数转换为每秒 MiB 数值。 */
function toMegabytes(bytes: number) {
  return Number.isFinite(bytes) && bytes > 0 ? bytes / (1024 ** 2) : 0;
}

/** formatBytes 将字节数转换为紧凑的二进制容量。 */
function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  /** units 保存容量单位。 */
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  /** unitIndex 保存当前容量单位索引。 */
  const unitIndex = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  /** value 保存按目标单位换算后的容量。 */
  const value = bytes / (1024 ** unitIndex);
  return `${value >= 100 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`;
}

/** formatRate 将每秒字节数转换为可读吞吐速率。 */
function formatRate(bytesPerSecond: number) {
  return `${formatBytes(bytesPerSecond)}/s`;
}

/** formatInteger 将整数指标转换为带千位分隔符的文本。 */
function formatInteger(value: number) {
  return Number.isFinite(value) ? Math.max(0, Math.round(value)).toLocaleString('zh-CN') : '0';
}

/** formatDuration 将秒数转换为天、小时、分钟和秒。 */
function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) return '等待采样';
  /** totalSeconds 表示向下取整后的有效秒数。 */
  const totalSeconds = Math.floor(seconds);
  /** days、hours、minutes、remainingSeconds 表示拆分后的持续时间。 */
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const remainingSeconds = totalSeconds % 60;
  if (days > 0) return `${days} 天 ${hours} 小时 ${minutes} 分钟`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`;
  if (minutes > 0) return `${minutes} 分钟 ${remainingSeconds} 秒`;
  return `${remainingSeconds} 秒`;
}

/** formatSampleTime 将 ISO 时间转换为浏览器本地时间。 */
function formatSampleTime(sampledAt?: string) {
  if (!sampledAt) return '等待采样';
  /** sampledDate 表示解析后的时间。 */
  const sampledDate = new Date(sampledAt);
  if (Number.isNaN(sampledDate.getTime())) return '等待采样';
  return sampledDate.toLocaleString('zh-CN', { hour12: false });
}

/** getHealthPresentation 返回健康状态的标签配置。 */
function getHealthPresentation(status?: ServerMetrics['health']['status']) {
  if (status === 'critical') return { label: '严重', color: 'error' } as const;
  if (status === 'warning') return { label: '需关注', color: 'warning' } as const;
  return { label: status === 'healthy' ? '健康' : '等待采样', color: status === 'healthy' ? 'success' : 'default' } as const;
}

/** createResourceTrendOption 创建 CPU、内存和磁盘使用率趋势图。 */
function createResourceTrendOption(theme: AdminTheme, history: MetricHistoryPoint[]): EChartsOption {
  return createLineChartBase(theme, history.map((point) => point.label), [
    { name: 'CPU', values: history.map((point) => point.cpu), color: theme.palette.charts[0] },
    { name: '内存', values: history.map((point) => point.memory), color: theme.palette.charts[1] },
    { name: '主分区', values: history.map((point) => point.disk), color: theme.palette.charts[3] },
  ], { max: 100, axisFormatter: '{value}%', tooltipFormatter: (value) => `${Number(value).toFixed(1)}%` });
}

/** createThroughputTrendOption 创建网络或磁盘吞吐速率趋势图。 */
function createThroughputTrendOption(theme: AdminTheme, history: MetricHistoryPoint[], kind: 'network' | 'disk'): EChartsOption {
  /** series 保存目标吞吐方向的折线数据。 */
  const series = kind === 'network'
    ? [
      { name: '下载', values: history.map((point) => point.download), color: theme.palette.charts[0] },
      { name: '上传', values: history.map((point) => point.upload), color: theme.palette.charts[4] },
    ]
    : [
      { name: '读取', values: history.map((point) => point.diskRead), color: theme.palette.charts[1] },
      { name: '写入', values: history.map((point) => point.diskWrite), color: theme.palette.charts[3] },
    ];
  return createLineChartBase(theme, history.map((point) => point.label), series, { axisFormatter: (value) => formatBytes(Number(value)), tooltipFormatter: (value) => formatRate(Number(value)) });
}

/** createLineChartBase 创建监控趋势图的共享坐标、图例和折线配置。 */
function createLineChartBase(
  theme: AdminTheme,
  labels: string[],
  seriesDefinitions: Array<{ name: string; values: number[]; color: string }>,
  format: { max?: number; axisFormatter: string | ((value: number) => string); tooltipFormatter: (value: unknown) => string },
): EChartsOption {
  return {
    animationDuration: 350,
    color: seriesDefinitions.map((series) => series.color),
    tooltip: {
      trigger: 'axis', confine: true, backgroundColor: theme.palette.panel, borderColor: theme.palette.border,
      textStyle: { color: theme.palette.text },
      formatter: (parameters) => {
        /** points 表示当前横轴位置的所有折线点。 */
        const points = Array.isArray(parameters) ? parameters : [parameters];
        if (!points.length) return '';
        /** title 表示当前样本时间。 */
        const title = String(points[0]?.name ?? '');
        return [title, ...points.map((point) => `${point.marker ?? ''}${point.seriesName}：${format.tooltipFormatter(point.value)}`)].join('<br/>');
      },
    },
    legend: { top: 2, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { color: theme.palette.textSecondary } },
    grid: { top: 48, right: 12, bottom: 24, left: 12, containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: labels, axisLabel: { color: theme.palette.textSecondary, hideOverlap: true }, axisLine: { lineStyle: { color: theme.palette.border } }, axisTick: { show: false } },
    yAxis: { type: 'value', min: 0, max: format.max, axisLabel: { color: theme.palette.textSecondary, formatter: format.axisFormatter }, splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } } },
    series: seriesDefinitions.map((series) => ({ name: series.name, type: 'line', data: series.values, showSymbol: false, smooth: 0.2, lineStyle: { width: 2 }, areaStyle: { opacity: 0.06 } })),
  };
}

/** createCoreUsageOption 创建每个逻辑处理器核心的横向使用率图。 */
function createCoreUsageOption(theme: AdminTheme, values: number[]): EChartsOption {
  /** visibleValues 限制异常使用率并保留所有逻辑核心。 */
  const visibleValues = values.map(clampPercent);
  return {
    animationDuration: 350,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, confine: true, formatter: '{b}：{c}%', backgroundColor: theme.palette.panel, borderColor: theme.palette.border, textStyle: { color: theme.palette.text } },
    grid: { top: 10, right: 42, bottom: 28, left: 48, containLabel: true },
    xAxis: { type: 'value', min: 0, max: 100, axisLabel: { color: theme.palette.textSecondary, formatter: '{value}%' }, splitLine: { lineStyle: { color: theme.palette.border, type: 'dashed' } } },
    yAxis: { type: 'category', data: visibleValues.map((_, index) => `CPU ${index}`), axisLabel: { color: theme.palette.textSecondary }, axisTick: { show: false }, axisLine: { show: false } },
    dataZoom: visibleValues.length > 12 ? [{ type: 'inside', yAxisIndex: 0 }, { type: 'slider', yAxisIndex: 0, width: 8, right: 2, showDetail: false }] : undefined,
    series: [{ name: '使用率', type: 'bar', data: visibleValues.map((value) => Number(value.toFixed(1))), barMaxWidth: 14, showBackground: true, backgroundStyle: { color: theme.palette.active }, itemStyle: { color: theme.palette.charts[0], borderRadius: [0, 3, 3, 0] } }],
  };
}

/** createOverviewOption 创建平台总量与有效资源柱状图。 */
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

/** createCompositionOption 创建平台业务资源构成环形图。 */
function createCompositionOption(theme: AdminTheme, resources: Array<{ name: string; value: number }>): EChartsOption {
  /** hasResources 表示是否存在可展示的业务资源。 */
  const hasResources = resources.some((resource) => resource.value > 0);
  return {
    animationDuration: 500,
    color: [...theme.palette.charts],
    tooltip: { trigger: 'item', confine: true, backgroundColor: theme.palette.panel, borderColor: theme.palette.border, textStyle: { color: theme.palette.text }, formatter: '{b}<br/>{c} 项 · {d}%' },
    legend: { bottom: 0, left: 'center', icon: 'circle', itemWidth: 9, itemHeight: 9, textStyle: { color: theme.palette.textSecondary } },
    series: [{ name: '资源构成', type: 'pie', radius: ['48%', '70%'], center: ['50%', '43%'], itemStyle: { borderColor: theme.palette.panel, borderWidth: 3, borderRadius: 4 }, label: { show: false }, data: hasResources ? resources : [{ name: '暂无资源', value: 1, itemStyle: { color: theme.palette.active } }] }],
  };
}
