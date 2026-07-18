'use client';

import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Button, Checkbox, Col, Descriptions, Modal, Popconfirm, Row, Switch, Tag, Tooltip, Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { LockKeyhole } from 'lucide-react';
import { actionPermissionGroups, allActionPermissionCodes } from '../lib/actionPermissions';
import type { Department, Menu, Role, User, UserForm } from '../types/admin';
import { shiftOptions, statusOptions } from '../lib/constants';
import { isAdministratorRoleCode, isSuperAdminRoleCode } from '../lib/roleAccess';

type UsersPageProps = {
  canCreate: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  canConfigurePermissions: boolean;
  actorRoleCode: string;
  users: User[];
  departments: Department[];
  roles: Role[];
  menus: Menu[];
  userForm: UserForm;
  editingUserId: number | null;
  selectedUserId: number | null;
  selectedMenuIds: number[];
  departmentMenuIds: number[];
  roleMenuIds: number[];
  effectiveMenuIds: number[];
  roleActionCodes: string[];
  userActionCodes: string[];
  effectiveActionCodes: string[];
  isLoading: boolean;
  isSavingUser: boolean;
  isSavingPermission: boolean;
  isSavingActionPermission: boolean;
  onRefresh: () => void;
  onUserFormChange: (form: UserForm) => void;
  onSubmitUser: (event: FormEvent<HTMLFormElement>) => void;
  onResetUserForm: () => void;
  onEditUser: (user: User) => void;
  onDeleteUser: (userId: number) => void;
  onSelectUser: (userId: number) => Promise<boolean>;
  onSavePermissions: (menuIds: number[]) => Promise<boolean>;
  onSaveActionPermissions: (actionCodes: string[]) => Promise<boolean>;
};

export function UsersPage({
  canCreate,
  canUpdate,
  canDelete,
  canConfigurePermissions,
  actorRoleCode,
  users,
  departments,
  roles,
  menus,
  userForm,
  editingUserId,
  selectedUserId,
  selectedMenuIds,
  departmentMenuIds,
  roleMenuIds,
  effectiveMenuIds,
  roleActionCodes,
  userActionCodes,
  effectiveActionCodes,
  isLoading,
  isSavingUser,
  isSavingPermission,
  isSavingActionPermission,
  onRefresh,
  onUserFormChange,
  onSubmitUser,
  onResetUserForm,
  onEditUser,
  onDeleteUser,
  onSelectUser,
  onSavePermissions,
  onSaveActionPermissions,
}: UsersPageProps) {
  const selectedUser = useMemo(() => users.find((user) => user.id === selectedUserId) ?? null, [users, selectedUserId]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [permissionDialogOpen, setPermissionDialogOpen] = useState(false);
  const [viewUser, setViewUser] = useState<User | null>(null);
  const [draftMenuIds, setDraftMenuIds] = useState<number[]>([]);
  const [draftActionCodes, setDraftActionCodes] = useState<string[]>([]);
  const [actionPermissionOpen, setActionPermissionOpen] = useState(false);
  const [keyword, setKeyword] = useState('');
  const selectedRole = useMemo(() => roles.find((role) => role.id === selectedUser?.roleId) ?? null, [roles, selectedUser]);
  const departmentPermissionSource = selectedUser?.department || '未分配部门';
  const rolePermissionSource = selectedRole?.name || selectedUser?.role || '未分配角色';
  const isAdministratorTarget = isAdministratorRoleCode(selectedUser?.roleCode);
  const actorIsSuperAdmin = isSuperAdminRoleCode(actorRoleCode);
  const canEditSelectedPermissions = canConfigurePermissions && !isAdministratorTarget;
  const inheritedMenuIds = useMemo(() => [...new Set([...departmentMenuIds, ...roleMenuIds])], [departmentMenuIds, roleMenuIds]);
  const inheritedActionCodes = useMemo(
    () => isAdministratorTarget ? allActionPermissionCodes : [...new Set(roleActionCodes)],
    [isAdministratorTarget, roleActionCodes],
  );
  const draftEffectiveActionCodes = useMemo(
    () => isAdministratorTarget ? allActionPermissionCodes : [...new Set([...roleActionCodes, ...draftActionCodes])],
    [draftActionCodes, isAdministratorTarget, roleActionCodes],
  );
  const filteredUsers = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return users;
    return users.filter((user) => [user.username, user.name, user.role, user.department, user.email, user.phone]
      .some((value) => String(value ?? '').toLowerCase().includes(query)));
  }, [keyword, users]);

  const treeData = useMemo<DataNode[]>(() => {
    const sorted = [...menus].filter((menu) => menu.status === '启用').sort((a, b) => a.sort - b.sort || a.id - b.id);
    const childrenByParent = new Map<number | null, Menu[]>();
    sorted.forEach((menu) => {
      const parentId = menu.parentId && sorted.some((candidate) => candidate.id === menu.parentId) ? menu.parentId : null;
      childrenByParent.set(parentId, [...(childrenByParent.get(parentId) ?? []), menu]);
    });
    const mapNodes = (parentId: number | null): DataNode[] => (childrenByParent.get(parentId) ?? []).map((menu) => ({
      key: menu.id,
      title: menu.name,
      disableCheckbox: inheritedMenuIds.includes(menu.id),
      children: mapNodes(menu.id),
    }));
    return mapNodes(null);
  }, [inheritedMenuIds, menus]);

  const roleOptions = useMemo(
    () => [...roles]
      .filter((role) => {
        if (isSuperAdminRoleCode(role.code)) return actorIsSuperAdmin || role.id === userForm.roleId;
        if (role.code === 'system-admin') return actorIsSuperAdmin || role.id === userForm.roleId;
        return true;
      })
      .sort((first, second) => first.sort - second.sort || first.id - second.id),
    [actorIsSuperAdmin, roles, userForm.roleId],
  );

  const departmentOptions = useMemo(() => {
    const sorted = [...departments].filter((department) => department.status === '启用').sort((a, b) => a.sort - b.sort || a.id - b.id);
    const flatten = (parentId: number | null, depth: number): Array<Department & { depth: number }> => sorted
      .filter((department) => (department.parentId ?? null) === parentId)
      .flatMap((department) => [{ ...department, depth }, ...flatten(department.id, depth + 1)]);
    return flatten(null, 0);
  }, [departments]);

  const openPermissionDialog = async (userId: number) => {
    if (await onSelectUser(userId)) setPermissionDialogOpen(true);
  };

  useEffect(() => {
    if (permissionDialogOpen) {
      setDraftMenuIds(selectedMenuIds);
      const inherited = new Set(roleActionCodes);
      setDraftActionCodes(userActionCodes.filter((code) => !inherited.has(code)));
      setActionPermissionOpen(false);
    }
  }, [permissionDialogOpen, roleActionCodes, selectedMenuIds, userActionCodes]);

  const confirmPermissions = async () => {
    if (!canEditSelectedPermissions) return;
    if (!await onSavePermissions(draftMenuIds)) return;
    if (await onSaveActionPermissions(draftActionCodes)) setPermissionDialogOpen(false);
  };

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
    const defaultRole = roleOptions.find((role) => role.code === 'viewer' && role.status === '启用') ?? null;
    onResetUserForm();
    onUserFormChange({
      username: '',
      name: '',
      role: defaultRole?.name ?? '',
      roleId: defaultRole?.id ?? null,
      department: '',
      departmentId: null,
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
          {canCreate && (
            <button className="primary-button" type="button" onClick={openCreateDialog}>
              新增用户
            </button>
          )}
        </div>
      </section>

      <section className="content-grid user-layout">
        <section className="panel-card">
          <div className="panel-heading user-list-heading">
            <div>
              <p className="page-kicker">账号列表</p>
              <h2>用户档案</h2>
            </div>
            <div className="user-search-tools">
              <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索用户名、姓名、角色或部门" aria-label="搜索用户" />
              {keyword && <Button onClick={() => setKeyword('')}>清空</Button>}
              <span className="count-tag">{filteredUsers.length} / {users.length} 人</span>
            </div>
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
                {filteredUsers.map((user) => (
                  <tr key={user.id} className={selectedUserId === user.id ? 'is-selected' : undefined}>
                    <td>{user.username}</td>
                    <td>{user.name}</td>
                    <td>{roles.find((role) => role.id === user.roleId)?.name ?? user.role ?? '-'}</td>
                    <td>{user.department || '-'}</td>
                    <td>
                      <span className={`status-badge ${user.status === '在岗' ? 'online' : 'offline'}`}>{user.status}</span>
                    </td>
                    <td>
                      <Tag color={user.canLogin && user.status !== '停用' ? 'success' : 'default'}>{user.canLogin && user.status !== '停用' ? '可登录' : '禁止登录'}</Tag>
                    </td>
                    <td>
                      <div className="action-group">
                        <button type="button" onClick={() => setViewUser(user)}>查看</button>
                        {(() => {
                          const targetIsAdministrator = isAdministratorRoleCode(user.roleCode);
                          const canAuthorizeTarget = canConfigurePermissions && !targetIsAdministrator;
                          const canEditTarget = canUpdate && (!targetIsAdministrator || actorIsSuperAdmin);
                          const canDeleteTarget = canDelete && user.username.toLowerCase() !== 'mh' && (!targetIsAdministrator || actorIsSuperAdmin);
                          if (!canAuthorizeTarget && !canEditTarget && !canDeleteTarget) return null;
                          return (
                          <>
                            {canAuthorizeTarget && <button type="button" onClick={() => void openPermissionDialog(user.id)}>授权</button>}
                            {canEditTarget && (
                            <button
                              type="button"
                              onClick={() => {
                                onEditUser(user);
                                setDialogOpen(true);
                              }}
                            >
                              编辑
                            </button>
                            )}
                            {canDeleteTarget && (
                              <Popconfirm
                                title="确认删除该用户？"
                                description={`账号“${user.username}”删除后不可恢复。`}
                                okText="确认删除"
                                cancelText="取消"
                                okButtonProps={{ danger: true }}
                                onConfirm={() => onDeleteUser(user.id)}
                              >
                                <button className="danger" type="button">删除</button>
                              </Popconfirm>
                            )}
                          </>
                          );
                        })()}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!isLoading && filteredUsers.length === 0 && <p className="empty-state">{keyword ? '没有匹配的用户。' : '暂无用户数据。'}</p>}
          </div>
        </section>
      </section>

      {viewUser && (
        <section className="panel-card user-detail-panel">
          <div className="panel-heading">
            <div>
              <p className="page-kicker">用户详情</p>
              <h2>{viewUser.name}</h2>
              <span>@{viewUser.username}</span>
            </div>
            <Button onClick={() => setViewUser(null)}>收起详情</Button>
          </div>
          {viewUser && (
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, sm: 2 }}
              items={[
                { key: 'username', label: '登录账号', children: viewUser.username },
                { key: 'name', label: '姓名', children: viewUser.name },
                { key: 'role', label: '所属角色', children: roles.find((role) => role.id === viewUser.roleId)?.name ?? viewUser.role ?? '-' },
                { key: 'department', label: '所属部门', children: viewUser.department || '-' },
                { key: 'status', label: '状态', children: viewUser.status },
                { key: 'login', label: '登录权限', children: viewUser.canLogin && viewUser.status !== '停用' ? '可登录' : '禁止登录' },
                { key: 'phone', label: '手机', children: viewUser.phone || '-' },
                { key: 'email', label: '邮箱', children: viewUser.email || '-' },
                { key: '创建时间', label: '创建时间', children: new Date(viewUser.createdAt).toLocaleString() },
                { key: '更新时间', label: '更新时间', children: new Date(viewUser.updatedAt).toLocaleString() },
              ]}
            />
        )}
        </section>
      )}

      <Modal
        open={permissionDialogOpen}
        title={`用户权限配置${selectedUser ? ` · ${selectedUser.name}` : ''}`}
        onCancel={() => setPermissionDialogOpen(false)}
        onOk={canEditSelectedPermissions ? () => void confirmPermissions() : undefined}
        okText="确认保存"
        cancelText="取消"
        confirmLoading={isSavingPermission || isSavingActionPermission}
        width={1120}
        destroyOnHidden
        className="permission-tree-modal user-permission-modal"
        footer={canEditSelectedPermissions ? undefined : <Button onClick={() => setPermissionDialogOpen(false)}>关闭</Button>}
      >
        <p className="section-subtitle">
          {isAdministratorTarget
            ? '超级管理员和系统管理员始终拥有全部菜单和按钮权限，此处仅供查看。'
            : '部门与角色权限自动继承且不可取消；管理员可在这里为普通用户追加个人菜单和按钮权限。'}
        </p>
        <div className="permission-section-header">
          <div>
            <strong>菜单权限</strong>
            <small>控制用户可以进入的功能页面</small>
          </div>
          <Tag color="blue">有效 {isAdministratorTarget ? menus.filter((menu) => menu.status === '启用').length : effectiveMenuIds.length} 项</Tag>
        </div>
        <Row gutter={[10, 10]} className="permission-source-grid">
          <Col xs={24} sm={8}>
            <article>
              <Tooltip title="用户所属部门提供的菜单权限"><span title="用户所属部门提供的菜单权限">部门权限</span></Tooltip>
              <strong>{departmentMenuIds.length}</strong>
              <Tooltip title={departmentPermissionSource}><small title={departmentPermissionSource}>{departmentPermissionSource}</small></Tooltip>
            </article>
          </Col>
          <Col xs={24} sm={8}>
            <article>
              <Tooltip title="用户所属角色提供的菜单权限"><span title="用户所属角色提供的菜单权限">角色权限</span></Tooltip>
              <strong>{roleMenuIds.length}</strong>
              <Tooltip title={rolePermissionSource}><small title={rolePermissionSource}>{rolePermissionSource}</small></Tooltip>
            </article>
          </Col>
          <Col xs={24} sm={8}>
            <article className="is-extra">
              <Tooltip title="只对当前用户单独追加的菜单权限"><span title="只对当前用户单独追加的菜单权限">用户额外权限</span></Tooltip>
              <strong>{draftMenuIds.length}</strong>
              <Tooltip title="可在下方菜单树中单独调整"><small title="可在下方菜单树中单独调整">可在下方单独调整</small></Tooltip>
            </article>
          </Col>
        </Row>
        <div className="permission-inheritance-note">
          <LockKeyhole size={15} />
          <span>{isAdministratorTarget ? '管理员全权限已锁定，不允许移除。' : `继承权限 ${inheritedMenuIds.length} 项，个人额外权限 ${draftMenuIds.length} 项，当前有效权限共 ${effectiveMenuIds.length} 项。`}</span>
        </div>
        <div className="permission-tree-panel">
          {treeData.length > 0 ? (
            <Tree
              checkable
              disabled={!canEditSelectedPermissions}
              selectable={false}
              defaultExpandAll
              treeData={treeData}
              checkedKeys={isAdministratorTarget ? menus.filter((menu) => menu.status === '启用').map((menu) => menu.id) : [...new Set([...inheritedMenuIds, ...draftMenuIds])]}
              onCheck={(checked) => {
                if (!canEditSelectedPermissions) return;
                const inherited = new Set(inheritedMenuIds);
                setDraftMenuIds((Array.isArray(checked) ? checked : checked.checked).map(Number).filter((id) => !inherited.has(id)));
              }}
            />
          ) : <p className="empty-state">暂无可授权菜单</p>}
        </div>

        <section className={`permission-collapse${actionPermissionOpen ? ' is-open' : ''}`}>
          <button
            className="permission-collapse-summary"
            type="button"
            aria-expanded={actionPermissionOpen}
            onClick={() => setActionPermissionOpen((current) => !current)}
          >
            <span>
              <strong>按钮权限</strong>
              <Tooltip title="控制查询、查看、新增、编辑、删除等具体操作">
                <small title="控制查询、查看、新增、编辑、删除等具体操作">控制查询、查看、新增、编辑、删除等具体操作</small>
              </Tooltip>
            </span>
            <Tag color="purple">有效 {draftEffectiveActionCodes.length} 项</Tag>
          </button>
          {actionPermissionOpen && (
            <div className="permission-collapse-body">
              <Row gutter={[10, 10]} className="permission-source-grid permission-action-sources">
                <Col xs={24} sm={8}>
                  <article>
                    <Tooltip title="用户所属角色继承的按钮操作权限"><span title="用户所属角色继承的按钮操作权限">角色动作</span></Tooltip>
                    <strong>{inheritedActionCodes.length}</strong>
                    <Tooltip title={isAdministratorTarget ? '管理员全权限' : rolePermissionSource}>
                      <small title={isAdministratorTarget ? '管理员全权限' : rolePermissionSource}>{isAdministratorTarget ? '管理员全权限' : rolePermissionSource}</small>
                    </Tooltip>
                  </article>
                </Col>
                <Col xs={24} sm={8}>
                  <article className="is-extra">
                    <Tooltip title="只对当前用户额外授予的按钮操作权限"><span title="只对当前用户额外授予的按钮操作权限">个人附加动作</span></Tooltip>
                    <strong>{isAdministratorTarget ? 0 : draftActionCodes.length}</strong>
                    <Tooltip title={isAdministratorTarget ? '管理员已经拥有全部按钮权限，无需个人追加' : '可由管理员在下方单独调整'}>
                      <small title={isAdministratorTarget ? '管理员已经拥有全部按钮权限，无需个人追加' : '可由管理员在下方单独调整'}>{isAdministratorTarget ? '无需个人追加' : '可由管理员调整'}</small>
                    </Tooltip>
                  </article>
                </Col>
                <Col xs={24} sm={8}>
                  <article>
                    <Tooltip title="角色继承权限与个人附加权限合并后的结果"><span title="角色继承权限与个人附加权限合并后的结果">当前有效动作</span></Tooltip>
                    <strong>{isAdministratorTarget ? allActionPermissionCodes.length : draftEffectiveActionCodes.length}</strong>
                    <Tooltip title="角色继承权限与个人附加权限的并集"><small title="角色继承权限与个人附加权限的并集">角色与个人附加权限并集</small></Tooltip>
                  </article>
                </Col>
              </Row>
              <div className="permission-inheritance-note">
                <LockKeyhole size={15} />
                <span>
                  {isAdministratorTarget
                    ? `超级管理员和系统管理员固定拥有全部 ${allActionPermissionCodes.length} 项按钮权限。`
                    : `后端已返回角色动作 ${roleActionCodes.length} 项、个人附加动作 ${userActionCodes.length} 项、有效动作 ${effectiveActionCodes.length} 项；勾选变化将作为该用户的个人附加动作保存。`}
                </span>
              </div>
              <Row gutter={[12, 12]} className="action-permission-groups">
                {actionPermissionGroups.map((group) => (
                  <Col xs={24} md={12} lg={8} className="action-permission-group-column" key={group.resource}>
                    <details className="action-permission-group">
                      <summary className="action-permission-group-title">
                        <span>
                          <Tooltip title={`${group.label}（${group.resource}）`}><strong title={`${group.label}（${group.resource}）`}>{group.label}</strong></Tooltip>
                          <Tooltip title={`权限资源：${group.resource}`}><small title={`权限资源：${group.resource}`}>{group.resource}</small></Tooltip>
                        </span>
                        <Tag color="blue">{group.actions.filter((action) => draftEffectiveActionCodes.includes(action.code)).length} / {group.actions.length}</Tag>
                      </summary>
                      <div className="action-permission-options">
                        {group.actions.map((action) => {
                          const inherited = inheritedActionCodes.includes(action.code);
                          const personal = !isAdministratorTarget && draftActionCodes.includes(action.code);
                          const effective = isAdministratorTarget || inherited || personal;
                          return (
                            <label className={`action-permission-option${effective ? ' is-checked' : ''}${inherited ? ' is-inherited' : ''}`} key={action.code}>
                              <Checkbox
                                checked={effective}
                                disabled={!canEditSelectedPermissions || inherited}
                                onChange={(event) => {
                                  if (!canEditSelectedPermissions || inherited) return;
                                  setDraftActionCodes((current) => event.target.checked
                                    ? [...new Set([...current, action.code])]
                                    : current.filter((code) => code !== action.code));
                                }}
                              />
                              <span>
                                <Tooltip title={`${action.label}（${action.code}）`}><strong title={`${action.label}（${action.code}）`}>{action.label}</strong></Tooltip>
                                <Tooltip title={action.description}><small title={action.description}>{action.description}</small></Tooltip>
                              </span>
                              {inherited && <em>{isAdministratorTarget ? '管理员全权限' : '角色继承'}</em>}
                              {personal && <em className="is-personal">个人附加</em>}
                            </label>
                          );
                        })}
                      </div>
                    </details>
                  </Col>
                ))}
              </Row>
            </div>
          )}
        </section>
      </Modal>

      <Modal open={dialogOpen} title={editingUserId ? '编辑用户' : '新增用户'} footer={null} destroyOnHidden width={640} onCancel={closeDialog} className="user-edit-modal">
        <form className="form-panel user-dialog-form" onSubmit={onSubmitUser}>
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
              所属角色
              <select
                required
                value={userForm.roleId ?? ''}
                onChange={(event) => {
                  const roleId = event.target.value ? Number(event.target.value) : null;
                  const role = roles.find((item) => item.id === roleId);
                  onUserFormChange({ ...userForm, roleId, role: role?.name ?? '' });
                }}
              >
                <option value="">请选择角色</option>
                {roleOptions.map((role) => (
                  <option key={role.id} value={role.id} disabled={(isSuperAdminRoleCode(role.code) && !actorIsSuperAdmin) || (role.status !== '启用' && role.id !== userForm.roleId)}>
                    {role.name}（{role.code}{role.status === '启用' ? '' : ' · 已停用'}）
                  </option>
                ))}
              </select>
            </label>
            <label>
              所属部门
              <select
                required
                value={userForm.departmentId ?? ''}
                onChange={(event) => {
                  const departmentId = event.target.value ? Number(event.target.value) : null;
                  const department = departments.find((item) => item.id === departmentId);
                  onUserFormChange({ ...userForm, departmentId, department: department?.name ?? '' });
                }}
              >
                <option value="">请选择部门</option>
                {departmentOptions.map((department) => (
                  <option key={department.id} value={department.id}>
                    {'　'.repeat(department.depth)}{department.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="form-row">
            <label>
              状态
              <select
                value={userForm.status}
                onChange={(event) => {
                  const status = event.target.value;
                  onUserFormChange({ ...userForm, status, canLogin: status === '停用' ? false : userForm.canLogin });
                }}
              >
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
            <Switch
              checked={userForm.status !== '停用' && userForm.canLogin}
              disabled={userForm.status === '停用'}
              onChange={(checked) => onUserFormChange({ ...userForm, canLogin: checked })}
              checkedChildren="可登录"
              unCheckedChildren={userForm.status === '停用' ? '已停用' : '禁止'}
            />
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
