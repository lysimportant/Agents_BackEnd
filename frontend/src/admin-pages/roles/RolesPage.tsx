'use client';

import { useMemo, useState, type FormEvent } from 'react';
import { Button, Empty, Input, InputNumber, Modal, Popconfirm, Select, Tag, Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { Eye, Pencil, Plus, RefreshCw, Search, ShieldCheck, Trash2, UsersRound } from 'lucide-react';
import { AssociatedUsersDialog } from '@/src/components/shared/AssociatedUsersDialog';
import type { ResourceActionAccess } from '@/src/utils/actionPermissions';
import { emptyRoleForm, roleStatusOptions } from '@/src/config/constants';
import type { Menu, Role, RoleForm, User } from '@/src/types/admin';
import { isAdministratorRoleCode, isSuperAdminRoleCode } from '@/src/utils/roleAccess';
import styles from './RolesPage.module.css';

type RolesPageProps = {
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** actorRoleCode 表示角色编码。 */
  actorRoleCode: string;
  /** roles 表示角色。 */
  roles: Role[];
  /** users 表示用户。 */
  users: User[];
  /** menus 表示菜单。 */
  menus: Menu[];
  /** isLoading 表示加载状态。 */
  isLoading: boolean;
  /** isSaving 表示保存状态。 */
  isSaving: boolean;
  /** isSavingPermissions 表示权限。 */
  isSavingPermissions: boolean;
  /** onRefresh 表示刷新回调。 */
  onRefresh: () => void;
  /** onSave 表示保存状态。 */
  onSave: (roleId: number | null, form: RoleForm) => Promise<boolean>;
  /** onDelete 表示删除回调。 */
  onDelete: (roleId: number) => Promise<boolean>;
  /** onLoadPermissions 表示权限。 */
  onLoadPermissions: (roleId: number) => Promise<number[] | null>;
  /** onLoadUsers 表示用户。 */
  onLoadUsers: (roleId: number) => Promise<User[] | null>;
  /** onSavePermissions 表示保存状态。 */
  onSavePermissions: (roleId: number, menuIds: number[]) => Promise<boolean>;
};

/** RolesPage 实现对应业务逻辑。 */
export function RolesPage({
  actions,
  actorRoleCode,
  roles,
  users,
  menus,
  isLoading,
  isSaving,
  isSavingPermissions,
  onRefresh,
  onSave,
  onDelete,
  onLoadPermissions,
  onLoadUsers,
  onSavePermissions,
}: RolesPageProps) {
  /** keyword、setKeyword 分别保存搜索关键词状态及其更新函数。 */
  const [keyword, setKeyword] = useState('');
  /** editorOpen、setEditorOpen 分别保存编辑器打开状态状态及其更新函数。 */
  const [editorOpen, setEditorOpen] = useState(false);
  /** permissionOpen、setPermissionOpen 分别保存权限状态及其更新函数。 */
  const [permissionOpen, setPermissionOpen] = useState(false);
  /** editingRole、setEditingRole 保存角色、角色。 */
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  /** permissionRole、setPermissionRole 保存权限角色、权限角色。 */
  const [permissionRole, setPermissionRole] = useState<Role | null>(null);
  /** form、setForm 保存表单、表单。 */
  const [form, setForm] = useState<RoleForm>(emptyRoleForm);
  /** permissionIds、setPermissionIds 保存权限标识列表、权限标识列表。 */
  const [permissionIds, setPermissionIds] = useState<number[]>([]);
  /** isLoadingPermissions、setIsLoadingPermissions 分别保存加载状态状态及其更新函数。 */
  const [isLoadingPermissions, setIsLoadingPermissions] = useState(false);
  /** deletingRoleId、setDeletingRoleId 保存角色标识、角色标识。 */
  const [deletingRoleId, setDeletingRoleId] = useState<number | null>(null);
  /** usersRole、setUsersRole 保存角色、角色。 */
  const [usersRole, setUsersRole] = useState<Role | null>(null);
  /** associatedUsers、setAssociatedUsers 保存用户、用户。 */
  const [associatedUsers, setAssociatedUsers] = useState<User[]>([]);
  /** isLoadingUsers、setIsLoadingUsers 分别保存加载状态状态及其更新函数。 */
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);
  /** usersError、setUsersError 分别保存错误状态状态及其更新函数。 */
  const [usersError, setUsersError] = useState('');

  /** filteredRoles 缓存计算得到的筛选后。 */
  const filteredRoles = useMemo(() => {
    /** query 保存查询条件。 */
    const query = keyword.trim().toLowerCase();
    return [...roles]
      .sort((first, second) => first.sort - second.sort || first.id - second.id)
      .filter((role) => !query || [role.name, role.code, role.description].some((value) => value.toLowerCase().includes(query)));
  }, [keyword, roles]);

  /** userCounts 缓存计算得到的用户。 */
  const userCounts = useMemo(() => {
    /** counts 保存数量。 */
    const counts = new Map<number, number>();
    users.forEach((user) => {
      if (user.roleId != null) counts.set(user.roleId, (counts.get(user.roleId) ?? 0) + 1);
    });
    return counts;
  }, [users]);

  /** menuTree 负责计算或维护菜单树形数据。 */
  const menuTree = useMemo<DataNode[]>(() => {
    /** sorted 负责计算或维护排序结果。 */
    const sorted = [...menus].filter((menu) => menu.status === '启用').sort((first, second) => first.sort - second.sort || first.id - second.id);
    /** childrenByParent 保存父级。 */
    const childrenByParent = new Map<number | null, Menu[]>();
    sorted.forEach((menu) => {
      /** parentId 负责计算或维护标识。 */
      const parentId = menu.parentId != null && sorted.some((candidate) => candidate.id === menu.parentId) ? menu.parentId : null;
      childrenByParent.set(parentId, [...(childrenByParent.get(parentId) ?? []), menu]);
    });
    /** mapNodes 负责计算或维护节点。 */
    const mapNodes = (parentId: number | null): DataNode[] => (childrenByParent.get(parentId) ?? []).map((menu) => ({
      key: menu.id,
      title: menu.name,
      children: mapNodes(menu.id),
    }));
    return mapNodes(null);
  }, [menus]);

  /** openCreate 负责计算或维护打开状态。 */
  const openCreate = () => {
    setEditingRole(null);
    setForm({ ...emptyRoleForm });
    setEditorOpen(true);
  };

  /** openEdit 负责计算或维护打开状态。 */
  const openEdit = (role: Role) => {
    setEditingRole(role);
    setForm({
      name: role.name,
      code: role.code,
      description: role.description,
      sort: role.sort,
      status: role.status,
    });
    setEditorOpen(true);
  };

  /** submitEditor 负责执行对应业务操作。 */
  const submitEditor = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (editingRole ? !actions.update : !actions.create) return;
    if (await onSave(editingRole?.id ?? null, form)) setEditorOpen(false);
  };

  /** openPermissions 负责计算或维护打开状态权限。 */
  const openPermissions = async (role: Role) => {
    setIsLoadingPermissions(true);
    /** ids 保存标识列表。 */
    const ids = await onLoadPermissions(role.id);
    setIsLoadingPermissions(false);
    if (ids === null) return;
    setPermissionRole(role);
    setPermissionIds(ids);
    setPermissionOpen(true);
  };

  /** deleteRole 负责删除或清理对应业务状态。 */
  const deleteRole = async (roleId: number) => {
    setDeletingRoleId(roleId);
    try {
      await onDelete(roleId);
    } finally {
      setDeletingRoleId(null);
    }
  };

  /** openUsers 负责计算或维护打开状态用户。 */
  const openUsers = async (role: Role) => {
    setUsersRole(role);
    setAssociatedUsers([]);
    setUsersError('');
    setIsLoadingUsers(true);
    /** result 保存操作结果。 */
    const result = await onLoadUsers(role.id);
    if (result === null) setUsersError('无法加载该角色的关联用户，请稍后重试。');
    else setAssociatedUsers(result);
    setIsLoadingUsers(false);
  };

  return (
    <div className="page-stack">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">组织与权限</p>
          <h1>角色管理</h1>
          <span>角色定义岗位职责的默认菜单权限；用户最终权限由部门、角色和个人额外权限共同组成。</span>
        </div>
        <div className="action-group">
          <Button icon={<RefreshCw size={15} />} onClick={onRefresh} loading={isLoading}>刷新角色</Button>
          {actions.create && <Button type="primary" icon={<Plus size={15} />} onClick={openCreate}>新增角色</Button>}
        </div>
      </section>

      <section className="panel-card">
        <div className="panel-heading user-list-heading">
          <div>
            <p className="page-kicker">岗位角色</p>
            <h2>角色与授权范围</h2>
          </div>
          <div className="user-search-tools">
            <Input
              allowClear
              value={keyword}
              prefix={<Search size={15} />}
              placeholder="搜索角色名称、编码或说明"
              onChange={(event) => setKeyword(event.target.value)}
            />
            <span className="count-tag">{filteredRoles.length} / {roles.length} 个角色</span>
          </div>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>角色</th>
                <th>角色编码</th>
                <th>说明</th>
                <th>关联用户</th>
                <th>排序</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {filteredRoles.map((role) => {
                /** isAdministratorRole 保存角色。 */
                const isAdministratorRole = isAdministratorRoleCode(role.code);
                /** actorIsSuperAdmin 保存管理员。 */
                const actorIsSuperAdmin = isSuperAdminRoleCode(actorRoleCode);
                return (
                  <tr key={role.id}>
                    <td><strong>{role.name}</strong></td>
                    <td><code>{role.code}</code></td>
                    <td><span className={styles.description}>{role.description || '暂无说明'}</span></td>
                    <td>
                      <div className={styles.userRelation}>
                        <span className={styles.userCount}><UsersRound size={14} />{userCounts.get(role.id) ?? 0} 人</span>
                        <Button size="small" icon={<Eye size={14} />} onClick={() => void openUsers(role)}>查看</Button>
                      </div>
                    </td>
                    <td>{role.sort}</td>
                    <td><Tag color={role.status === '启用' ? 'success' : 'default'}>{role.status}</Tag></td>
                    <td>
                      <div className="action-group">
                        <button className={styles.actionButton} type="button" disabled={isLoadingPermissions} onClick={() => void openPermissions(role)}><ShieldCheck size={14} />{actions.permissions && !isAdministratorRole ? '授权' : '查看权限'}</button>
                        {actions.update && (!isAdministratorRole || actorIsSuperAdmin) && <button className={styles.actionButton} type="button" onClick={() => openEdit(role)}><Pencil size={14} />编辑</button>}
                        {actions.delete && !isAdministratorRole && (
                          <Popconfirm
                            title={`删除角色“${role.name}”？`}
                            description="存在关联用户时后端会拒绝删除。"
                            okText="确认删除"
                            cancelText="取消"
                            okButtonProps={{ danger: true, loading: deletingRoleId === role.id }}
                            onConfirm={() => deleteRole(role.id)}
                          >
                            <button className={`danger ${styles.actionButton}`} type="button" disabled={deletingRoleId === role.id}><Trash2 size={14} />删除</button>
                          </Popconfirm>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {!isLoading && filteredRoles.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={keyword ? '没有匹配的角色' : '暂无角色'} />}
        </div>
      </section>

      <Modal
        open={editorOpen}
        title={editingRole ? '编辑角色' : '新增角色'}
        footer={null}
        width={640}
        destroyOnHidden
        onCancel={() => setEditorOpen(false)}
      >
        <form className={styles.formGrid} onSubmit={submitEditor}>
          <label>
            角色名称
            <input required disabled={isAdministratorRoleCode(editingRole?.code)} value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="例如：内容编辑" />
          </label>
          <label>
            角色编码
            <input
              required
              disabled={editingRole !== null}
              pattern="[a-z0-9][a-z0-9-]*"
              title="仅支持小写字母、数字和连字符"
              value={form.code}
              onChange={(event) => setForm({ ...form, code: event.target.value.toLowerCase() })}
              placeholder="例如：content-editor"
            />
          </label>
          <label>
            显示顺序
            <InputNumber min={0} precision={0} value={form.sort} onChange={(value) => setForm({ ...form, sort: Number(value ?? 0) })} />
          </label>
          <label>
            状态
            <Select disabled={isAdministratorRoleCode(editingRole?.code)} value={form.status} options={roleStatusOptions.map((status) => ({ label: status, value: status }))} onChange={(status) => setForm({ ...form, status })} />
          </label>
          <label className={styles.spanTwo}>
            角色说明
            <Input.TextArea rows={4} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} placeholder="说明该角色的职责和适用范围" />
          </label>
          <div className={`rich-editor-actions ${styles.spanTwo}`}>
            <Button onClick={() => setEditorOpen(false)}>取消</Button>
            <Button type="primary" htmlType="submit" loading={isSaving}>{editingRole ? '保存角色' : '创建角色'}</Button>
          </div>
        </form>
      </Modal>

      <Modal
        open={permissionOpen}
        title={`角色菜单权限${permissionRole ? ` · ${permissionRole.name}` : ''}`}
        okText="保存权限"
        cancelText="取消"
        confirmLoading={isSavingPermissions}
        okButtonProps={{ disabled: !actions.permissions || isAdministratorRoleCode(permissionRole?.code) }}
        width={600}
        destroyOnHidden
        onCancel={() => setPermissionOpen(false)}
        footer={!actions.permissions || isAdministratorRoleCode(permissionRole?.code) ? <Button onClick={() => setPermissionOpen(false)}>关闭</Button> : undefined}
        onOk={async () => {
          if (actions.permissions && permissionRole && await onSavePermissions(permissionRole.id, permissionIds)) setPermissionOpen(false);
        }}
      >
        <div className={styles.permissionIntro}>
          <ShieldCheck size={18} />
          <span>{isAdministratorRoleCode(permissionRole?.code) ? '超级管理员和系统管理员始终拥有全部菜单权限，此处仅用于查看。' : '选中的菜单会自动授予所有使用该角色的用户，个人额外权限不会在这里被覆盖。'}</span>
        </div>
        <div className={styles.permissionTree}>
          {menuTree.length ? (
            <Tree
              checkable
              disabled={!actions.permissions || isAdministratorRoleCode(permissionRole?.code)}
              selectable={false}
              defaultExpandAll
              treeData={menuTree}
              checkedKeys={permissionIds}
              onCheck={(checked) => setPermissionIds((Array.isArray(checked) ? checked : checked.checked).map(Number))}
            />
          ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无可授权菜单" />}
        </div>
      </Modal>

      <AssociatedUsersDialog
        open={Boolean(usersRole)}
        title={`角色关联用户${usersRole ? ` · ${usersRole.name}` : ''}`}
        users={associatedUsers}
        isLoading={isLoadingUsers}
        error={usersError}
        onClose={() => setUsersRole(null)}
      />
    </div>
  );
}
