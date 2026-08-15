'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { Alert, Button, Empty, Input, Modal, Select, Spin, Tag } from 'antd';
import { fetchServerConnections } from '@/src/services/serverApi';
import type { ServerConnectionDetail, ServerConnectionDetailsResponse } from '@/src/types/admin';

type ConnectionDetailsModalProps = {
  /** open 表示连接明细弹窗是否可见。 */
  open: boolean;
  /** onClose 关闭弹窗但不影响服务器指标自动采样。 */
  onClose: () => void;
};

/** EMPTY_STATUS_FILTER 表示不限制连接状态。 */
const EMPTY_STATUS_FILTER = 'ALL';

/** ConnectionDetailsModal 按需展示服务器网络连接的端点、状态和所属进程。 */
export function ConnectionDetailsModal({ open, onClose }: ConnectionDetailsModalProps) {
  /** snapshot、setSnapshot 保存最近一次连接明细快照。 */
  const [snapshot, setSnapshot] = useState<ServerConnectionDetailsResponse | null>(null);
  /** isLoading、setIsLoading 表示连接明细是否正在刷新。 */
  const [isLoading, setIsLoading] = useState(false);
  /** error、setError 保存连接明细请求错误。 */
  const [error, setError] = useState('');
  /** keyword、setKeyword 保存进程或端点搜索词。 */
  const [keyword, setKeyword] = useState('');
  /** statusFilter、setStatusFilter 保存当前连接状态筛选。 */
  const [statusFilter, setStatusFilter] = useState('ESTABLISHED');

  /** loadConnections 请求一次实时连接明细。 */
  const loadConnections = useCallback(async () => {
    setIsLoading(true);
    setError('');
    try {
      /** nextSnapshot 保存连接明细接口返回的新快照。 */
      const nextSnapshot = await fetchServerConnections();
      setSnapshot(nextSnapshot);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载网络连接明细失败');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) void loadConnections();
  }, [loadConnections, open]);

  /** statusOptions 根据当前快照生成可用状态筛选项。 */
  const statusOptions = useMemo(() => {
    /** statuses 保存连接明细中实际出现的状态。 */
    const statuses = Array.from(new Set((snapshot?.connections ?? []).map((connection) => connection.status))).sort();
    return [
      { value: EMPTY_STATUS_FILTER, label: '全部状态' },
      ...statuses.map((status) => ({ value: status, label: connectionStatusLabel(status) })),
    ];
  }, [snapshot?.connections]);

  /** filteredConnections 保存符合状态和搜索词的连接明细。 */
  const filteredConnections = useMemo(() => {
    /** normalizedKeyword 保存小写搜索词。 */
    const normalizedKeyword = keyword.trim().toLocaleLowerCase();
    return (snapshot?.connections ?? []).filter((connection) => {
      /** statusMatches 表示连接状态是否命中筛选。 */
      const statusMatches = statusFilter === EMPTY_STATUS_FILTER || connection.status === statusFilter;
      if (!statusMatches) return false;
      if (!normalizedKeyword) return true;
      /** searchableText 保存连接可搜索字段的合并文本。 */
      const searchableText = [
        connection.protocol,
        connection.addressFamily,
        connection.localAddress,
        connection.localPort,
        connection.remoteAddress,
        connection.remotePort,
        connection.status,
        connection.pid,
        connection.processName,
      ].join(' ').toLocaleLowerCase();
      return searchableText.includes(normalizedKeyword);
    });
  }, [keyword, snapshot?.connections, statusFilter]);

  /** summary 保存连接协议和常见状态的准确汇总。 */
  const summary = snapshot?.summary;

  return (
    <Modal
      title="活动连接详情"
      open={open}
      width={1100}
      footer={null}
      onCancel={onClose}
      mask={{ closable: true }}
      className="connection-details-modal"
    >
      <div className="connection-details-content">
        <section className="connection-details-summary" aria-label="网络连接汇总">
          <Tag color="blue">已采样 {snapshot?.sampled ?? 0}</Tag>
          <Tag>TCP {summary?.tcp ?? 0}</Tag>
          <Tag>UDP {summary?.udp ?? 0}</Tag>
          <Tag color="green">已建立 {summary?.established ?? 0}</Tag>
          <Tag color="gold">监听 {summary?.listen ?? 0}</Tag>
          <Tag color="orange">TIME_WAIT {summary?.timeWait ?? 0}</Tag>
          <span>{formatSampledAt(snapshot?.sampledAt)}</span>
        </section>

        <section className="connection-details-toolbar" aria-label="连接明细筛选">
          <Select
            aria-label="连接状态"
            value={statusFilter}
            options={statusOptions}
            onChange={setStatusFilter}
          />
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="搜索地址、端口、PID 或进程"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <Button icon={<ReloadOutlined spin={isLoading} />} onClick={() => void loadConnections()} disabled={isLoading}>
            刷新
          </Button>
        </section>

        {error && <Alert type="error" showIcon title={error} />}
        {snapshot && !snapshot.available && <Alert type="warning" showIcon title={snapshot.warning || '当前环境无法读取网络连接明细'} />}
        {(snapshot?.truncated || snapshot?.detailsTruncated) && (
          <Alert type="warning" showIcon title="连接数量较多，当前仅展示采集上限内的前 500 条明细。" />
        )}

        <Spin spinning={isLoading} tip="正在读取连接明细">
          <div className="connection-details-table-scroll">
            <table className="dashboard-metrics-table connection-details-table">
              <thead>
                <tr><th>协议</th><th>状态</th><th>本地端点</th><th>远端端点</th><th>进程</th><th>PID</th></tr>
              </thead>
              <tbody>
                {filteredConnections.map((connection, index) => (
                  <tr key={connectionRowKey(connection, index)}>
                    <td><Tag>{connection.protocol}</Tag><small>{connection.addressFamily}</small></td>
                    <td><Tag color={connectionStatusColor(connection.status)}>{connectionStatusLabel(connection.status)}</Tag></td>
                    <td><code>{formatEndpoint(connection.localAddress, connection.localPort)}</code></td>
                    <td><code>{formatEndpoint(connection.remoteAddress, connection.remotePort)}</code></td>
                    <td>{connection.processName || '权限未提供'}</td>
                    <td>{connection.pid > 0 ? connection.pid : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!isLoading && !filteredConnections.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当前筛选条件下没有连接" />}
          </div>
        </Spin>
      </div>
    </Modal>
  );
}

/** formatEndpoint 将 IP 和端口组合为可读端点。 */
function formatEndpoint(address: string, port: number) {
  if (!address && !port) return '-';
  /** visibleAddress 保存空地址的通配符回退和 IPv6 方括号格式。 */
  const visibleAddress = address || '*';
  const formattedAddress = visibleAddress.includes(':') ? `[${visibleAddress}]` : visibleAddress;
  return port > 0 ? `${formattedAddress}:${port}` : formattedAddress;
}

/** connectionStatusLabel 将系统状态转换为紧凑中文标签。 */
function connectionStatusLabel(status: string) {
  /** labels 保存常见状态的中文显示名称。 */
  const labels: Record<string, string> = {
    ESTABLISHED: '已建立',
    LISTEN: '监听',
    TIME_WAIT: '等待关闭',
    CLOSE_WAIT: '等待本地关闭',
    SYN_SENT: '正在连接',
    SYN_RECV: '等待确认',
    NONE: '无状态',
  };
  return labels[status] ?? status;
}

/** connectionStatusColor 返回连接状态对应的 Ant Design 标签色。 */
function connectionStatusColor(status: string) {
  if (status === 'ESTABLISHED') return 'green';
  if (status === 'LISTEN') return 'blue';
  if (status === 'CLOSE_WAIT') return 'red';
  if (status === 'TIME_WAIT') return 'orange';
  return 'default';
}

/** formatSampledAt 将连接采样时间转换为本地显示。 */
function formatSampledAt(sampledAt?: string) {
  if (!sampledAt) return '等待采样';
  /** sampledDate 保存解析后的连接采样时间。 */
  const sampledDate = new Date(sampledAt);
  return Number.isNaN(sampledDate.getTime()) ? '等待采样' : sampledDate.toLocaleString('zh-CN', { hour12: false });
}

/** connectionRowKey 为重复端点连接生成稳定的当前快照行键。 */
function connectionRowKey(connection: ServerConnectionDetail, index: number) {
  return `${connection.protocol}-${connection.localAddress}-${connection.localPort}-${connection.remoteAddress}-${connection.remotePort}-${connection.pid}-${index}`;
}
