'use client';

import { Alert, Avatar, Button, Empty, Modal, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { User } from '../types/admin';
import styles from './AssociatedUsersDialog.module.css';

type AssociatedUsersDialogProps = {
  open: boolean;
  title: string;
  users: User[];
  isLoading: boolean;
  error: string;
  onClose: () => void;
};

const columns: ColumnsType<User> = [
  {
    title: '用户',
    key: 'identity',
    fixed: 'left',
    width: 180,
    render: (_, user) => (
      <div className={styles.identity}>
        <Avatar src={user.avatarUrl || undefined}>{Array.from(user.name.trim() || user.username || '?')[0]?.toUpperCase()}</Avatar>
        <span><strong>{user.name}</strong><small>@{user.username}</small></span>
      </div>
    ),
  },
  { title: '角色', dataIndex: 'role', key: 'role', width: 130, render: (value: string) => value || '未分配' },
  { title: '部门', dataIndex: 'department', key: 'department', width: 150, render: (value: string) => value || '未分配' },
  {
    title: '联系方式',
    key: 'contact',
    width: 210,
    render: (_, user) => (
      <span className={styles.contact}>
        <span>{user.email || '未绑定邮箱'}</span>
        <small>{user.phone || '未填写电话'}</small>
      </span>
    ),
  },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render: (_, user) => (
      <span className={styles.status}>
        <Tag color={user.status === '停用' ? 'default' : 'success'}>{user.status}</Tag>
        <small>{user.canLogin && user.status !== '停用' ? '允许登录' : '禁止登录'}</small>
      </span>
    ),
  },
];

export function AssociatedUsersDialog({ open, title, users, isLoading, error, onClose }: AssociatedUsersDialogProps) {
  return (
    <Modal
      open={open}
      title={title}
      width={900}
      destroyOnHidden
      onCancel={onClose}
      footer={<Button onClick={onClose}>关闭</Button>}
    >
      <div className={styles.summary}>
        <span>当前归属</span>
        <strong>{isLoading ? '—' : users.length}</strong>
        <small>名用户</small>
      </div>
      {error && <Alert className={styles.error} type="error" showIcon title={error} />}
      <Table<User>
        rowKey="id"
        size="middle"
        loading={isLoading}
        columns={columns}
        dataSource={users}
        pagination={users.length > 8 ? { pageSize: 8, showSizeChanger: false } : false}
        scroll={{ x: 790 }}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无归属用户" /> }}
      />
    </Modal>
  );
}
