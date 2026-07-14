'use client';

import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Button, Modal, Switch, Tag } from 'antd';
import type { Menu, User, UserForm } from '../types/admin';
import { roleOptions, shiftOptions, statusOptions } from '../lib/constants';

type UsersPageProps = {
  users: User[];
  menus: Menu[];
  userForm: UserForm;
  editingUserId: number | null;
  selectedUserId: number | null;
  selectedMenuIds: number[];
  isLoading: boolean;
  isSavingUser: boolean;
  isSavingPermission: boolean;
  onRefresh: () => void;
  onUserFormChange: (form: UserForm) => void;
  onSubmitUser: (event: FormEvent<HTMLFormElement>) => void;
  onResetUserForm: () => void;
  onEditUser: (user: User) => void;
  onDeleteUser: (userId: number) => void;
  onSelectUser: (userId: number) => void;
  onToggleMenuPermission: (menuId: number) => void;
  onSavePermissions: () => void;
};

export function UsersPage({
  users,
  menus,
  userForm,
  editingUserId,
  selectedUserId,
  selectedMenuIds,
  isLoading,
  isSavingUser,
  isSavingPermission,
  onRefresh,
  onUserFormChange,
  onSubmitUser,
  onResetUserForm,
  onEditUser,
  onDeleteUser,
  onSelectUser,
  onToggleMenuPermission,
  onSavePermissions,
}: UsersPageProps) {
  const selectedUser = useMemo(() => users.find((user) => user.id === selectedUserId) ?? null, [users, selectedUserId]);
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    if (editingUserId !== null) {
      setDialogOpen(true);
    }
  }, [editingUserId]);

  useEffect(() => {
    if (!isSavingUser && editingUserId === null && !userForm.username && !userForm.name && !userForm.password) {
      setDialogOpen(false);
    }
  }, [isSavingUser, editingUserId, userForm.username, userForm.name, userForm.password]);

  const openCreateDialog = () => {
    onResetUserForm();
    onUserFormChange({
      username: '',
      name: '',
      role: roleOptions[0] ?? '内容编辑',
      department: '',
      status: statusOptions[0] ?? '在岗',
      shift: shiftOptions[0] ?? '白班',
      phone: '',
      email: '',
      canLogin: true,
      password: '',
    });
    setDialogOpen(true);
  };

  const closeDialog = () => {
    setDialogOpen(false);
    onResetUserForm();
  };

  return (
    <div className="page-stack">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">系统管理</p>
          <h1>用户管理</h1>
          <span>维护账号档案、登录权限与菜单授权；编辑用户通过弹窗完成，不占用主列表空间。</span>
        </div>
        <div className="action-group">
          <button className="ghost-button" type="button" onClick={onRefresh} disabled={isLoading}>
            {isLoading ? '刷新中' : '刷新'}
          </button>
          <button className="primary-button" type="button" onClick={openCreateDialog}>
            新增用户
          </button>
        </div>
      </section>

      <section className="content-grid user-layout">
        <section className="panel-card">
          <div className="panel-heading">
            <div>
              <p className="page-kicker">账号列表</p>
              <h2>用户档案</h2>
            </div>
            <span className="count-tag">{users.length} 人</span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>账号</th>
                  <th>姓名</th>
                  <th>角色</th>
                  <th>部门</th>
                  <th>状态</th>
                  <th>登录权限</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className={selectedUserId === user.id ? 'is-selected' : undefined}>
                    <td>{user.username}</td>
                    <td>{user.name}</td>
                    <td>{user.role}</td>
                    <td>{user.department || '-'}</td>
                    <td>
                      <span className={`status-badge ${user.status === '在岗' ? 'online' : 'offline'}`}>{user.status}</span>
                    </td>
                    <td>
                      <Tag color={user.canLogin ? 'success' : 'default'}>{user.canLogin ? '可登录' : '禁止登录'}</Tag>
                    </td>
                    <td>
                      <div className="action-group">
                        <button type="button" onClick={() => onSelectUser(user.id)}>
                          授权
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            onEditUser(user);
                            setDialogOpen(true);
                          }}
                        >
                          编辑
                        </button>
                        <button className="danger" type="button" onClick={() => onDeleteUser(user.id)}>
                          删除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!isLoading && users.length === 0 && <p className="empty-state">暂无用户数据。</p>}
          </div>
        </section>

        <section className="panel-card">
          <div className="panel-heading">
            <div>
              <p className="page-kicker">菜单授权</p>
              <h2>{selectedUser ? `${selectedUser.name} 的菜单权限` : '选择用户进行授权'}</h2>
            </div>
            <button className="primary-button" type="button" disabled={!selectedUserId || isSavingPermission} onClick={onSavePermissions}>
              {isSavingPermission ? '保存中...' : '保存权限'}
            </button>
          </div>
          {!selectedUserId ? (
            <p className="empty-state">请先在左侧选择用户，再配置其可访问菜单。</p>
          ) : (
            <div className="permission-grid">
              {menus.map((menu) => {
                const checked = selectedMenuIds.includes(menu.id);
                return (
                  <label className={`permission-item ${checked ? 'checked' : ''}`} key={menu.id}>
                    <input type="checkbox" checked={checked} onChange={() => onToggleMenuPermission(menu.id)} />
                    <span>
                      <strong>{menu.name}</strong>
                      <small>
                        {menu.code} · {menu.path || '无路径'}
                      </small>
                    </span>
                  </label>
                );
              })}
              {menus.length === 0 && <p className="empty-state">暂无真实菜单数据，请先在菜单管理中维护。</p>}
            </div>
          )}
        </section>
      </section>

      <Modal open={dialogOpen} title={editingUserId ? '编辑用户' : '新增用户'} footer={null} destroyOnClose width={640} onCancel={closeDialog}>
        <form className="panel-card form-panel user-dialog-form" onSubmit={onSubmitUser}>
          <div className="form-row">
            <label>
              登录账号
              <input required value={userForm.username} onChange={(event) => onUserFormChange({ ...userForm, username: event.target.value })} placeholder="唯一用户名" />
            </label>
            <label>
              姓名
              <input required value={userForm.name} onChange={(event) => onUserFormChange({ ...userForm, name: event.target.value })} placeholder="显示名称" />
            </label>
          </div>
          <div className="form-row">
            <label>
              角色
              <select value={userForm.role} onChange={(event) => onUserFormChange({ ...userForm, role: event.target.value })}>
                {roleOptions.map((role) => (
                  <option key={role} value={role}>
                    {role}
                  </option>
                ))}
              </select>
            </label>
            <label>
              部门
              <input value={userForm.department} onChange={(event) => onUserFormChange({ ...userForm, department: event.target.value })} placeholder="所属部门" />
            </label>
          </div>
          <div className="form-row">
            <label>
              状态
              <select value={userForm.status} onChange={(event) => onUserFormChange({ ...userForm, status: event.target.value })}>
                {statusOptions.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </label>
            <label>
              班次
              <select value={userForm.shift} onChange={(event) => onUserFormChange({ ...userForm, shift: event.target.value })}>
                {shiftOptions.map((shift) => (
                  <option key={shift} value={shift}>
                    {shift}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="form-row">
            <label>
              手机
              <input value={userForm.phone} onChange={(event) => onUserFormChange({ ...userForm, phone: event.target.value })} placeholder="可选" />
            </label>
            <label>
              邮箱
              <input value={userForm.email} onChange={(event) => onUserFormChange({ ...userForm, email: event.target.value })} placeholder="可选" />
            </label>
          </div>
          <label>
            {editingUserId ? '重置密码（留空表示不修改）' : '初始密码'}
            <input
              type="password"
              required={!editingUserId}
              value={userForm.password}
              onChange={(event) => onUserFormChange({ ...userForm, password: event.target.value })}
              placeholder={editingUserId ? '不修改请留空' : '请设置初始密码'}
            />
          </label>
          <div className="privacy-switch-row">
            <div>
              <strong>登录权限</strong>
              <small>关闭后该账号无法登录系统，已有会话也会被拦截。</small>
            </div>
            <Switch checked={userForm.canLogin} onChange={(checked) => onUserFormChange({ ...userForm, canLogin: checked })} checkedChildren="可登录" unCheckedChildren="禁止" />
          </div>
          <div className="rich-editor-actions">
            <Button onClick={closeDialog}>取消</Button>
            <Button type="primary" htmlType="submit" loading={isSavingUser}>
              {editingUserId ? '保存用户' : '创建用户'}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
