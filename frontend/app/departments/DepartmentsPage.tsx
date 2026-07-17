'use client';

import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Button, Empty, InputNumber, Modal, Select, Tag, Tree, TreeSelect } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { Eye, Pencil, Plus, RefreshCw, ShieldCheck, Trash2, UsersRound } from 'lucide-react';
import { AssociatedUsersDialog } from '../components/AssociatedUsersDialog';
import type { ResourceActionAccess } from '../lib/actionPermissions';
import { departmentStatusOptions, emptyDepartmentForm } from '../lib/constants';
import type { Department, DepartmentForm, Menu, User } from '../types/admin';
import styles from './DepartmentsPage.module.css';

type DepartmentsPageProps = {
  actions: ResourceActionAccess;
  departments: Department[];
  users: User[];
  menus: Menu[];
  isLoading: boolean;
  isSaving: boolean;
  isSavingPermissions: boolean;
  onRefresh: () => void;
  onSave: (departmentId: number | null, form: DepartmentForm) => Promise<boolean>;
  onDelete: (departmentId: number) => Promise<boolean>;
  onLoadPermissions: (departmentId: number) => Promise<number[] | null>;
  onLoadUsers: (departmentId: number) => Promise<User[] | null>;
  onSavePermissions: (departmentId: number, menuIds: number[]) => Promise<boolean>;
};

function byOrder<T extends { sort: number; id: number }>(first: T, second: T) {
  return first.sort - second.sort || first.id - second.id;
}

export function DepartmentsPage({
  actions,
  departments,
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
}: DepartmentsPageProps) {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [permissionOpen, setPermissionOpen] = useState(false);
  const [form, setForm] = useState<DepartmentForm>(emptyDepartmentForm);
  const [permissionIds, setPermissionIds] = useState<number[]>([]);
  const [usersDepartment, setUsersDepartment] = useState<Department | null>(null);
  const [associatedUsers, setAssociatedUsers] = useState<User[]>([]);
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);
  const [usersError, setUsersError] = useState('');

  const selected = departments.find((department) => department.id === selectedId) ?? null;
  const childrenCount = selected ? departments.filter((department) => department.parentId === selected.id).length : 0;
  const userCount = selected ? users.filter((user) => user.departmentId === selected.id).length : 0;

  useEffect(() => {
    if (departments.length === 0) {
      setSelectedId(null);
      return;
    }
    if (!selectedId || !departments.some((department) => department.id === selectedId)) {
      setSelectedId([...departments].sort(byOrder)[0]?.id ?? null);
    }
  }, [departments, selectedId]);

  const departmentTree = useMemo<DataNode[]>(() => {
    const sorted = [...departments].sort(byOrder);
    const nodesFor = (parentId: number | null): DataNode[] => sorted
      .filter((department) => (department.parentId ?? null) === parentId)
      .map((department) => ({
        key: department.id,
        title: (
          <span className={styles.treeTitle}>
            <span>{department.name}</span>
            <span className={styles.code}>{department.code}</span>
          </span>
        ),
        children: nodesFor(department.id),
      }));
    return nodesFor(null);
  }, [departments]);

  const parentTree = useMemo(() => {
    const descendants = new Set<number>();
    const collect = (departmentId: number) => {
      departments.filter((item) => item.parentId === departmentId).forEach((item) => {
        descendants.add(item.id);
        collect(item.id);
      });
    };
    if (editingId) collect(editingId);
    const sorted = [...departments].sort(byOrder);
    const nodesFor = (parentId: number | null): DataNode[] => sorted
      .filter((department) => (department.parentId ?? null) === parentId)
      .map((department) => ({
        key: department.id,
        value: department.id,
        title: department.name,
        disabled: department.id === editingId || descendants.has(department.id),
        children: nodesFor(department.id),
      }));
    return nodesFor(null);
  }, [departments, editingId]);

  const menuTree = useMemo<DataNode[]>(() => {
    const sorted = [...menus].filter((menu) => menu.status === '启用').sort(byOrder);
    const nodesFor = (parentId: number | null): DataNode[] => sorted
      .filter((menu) => (menu.parentId ?? null) === parentId)
      .map((menu) => ({ key: menu.id, title: menu.name, children: nodesFor(menu.id) }));
    return nodesFor(null);
  }, [menus]);

  const openCreate = () => {
    setEditingId(null);
    setForm({ ...emptyDepartmentForm, parentId: selectedId });
    setEditorOpen(true);
  };

  const openEdit = (department: Department) => {
    setEditingId(department.id);
    setForm({
      name: department.name,
      code: department.code,
      parentId: department.parentId,
      leader: department.leader,
      phone: department.phone,
      email: department.email,
      sort: department.sort,
      status: department.status,
    });
    setEditorOpen(true);
  };

  const submitEditor = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (editingId ? !actions.update : !actions.create) return;
    if (await onSave(editingId, form)) setEditorOpen(false);
  };

  const openPermissions = async (department: Department) => {
    setSelectedId(department.id);
    const menuIds = await onLoadPermissions(department.id);
    if (menuIds === null) return;
    setPermissionIds(menuIds);
    setPermissionOpen(true);
  };

  const confirmDelete = (department: Department) => {
    Modal.confirm({
      title: `删除部门“${department.name}”？`,
      content: '存在下级部门或关联用户时，后端会拒绝删除，避免破坏组织关系。',
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        if (await onDelete(department.id)) setSelectedId(null);
      },
    });
  };

  const openUsers = async (department: Department) => {
    setUsersDepartment(department);
    setAssociatedUsers([]);
    setUsersError('');
    setIsLoadingUsers(true);
    const result = await onLoadUsers(department.id);
    if (result === null) setUsersError('无法加载该部门的归属用户，请稍后重试。');
    else setAssociatedUsers(result);
    setIsLoadingUsers(false);
  };

  return (
    <div className="page-stack">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">组织与权限</p>
          <h1>部门管理</h1>
          <span>以组织树统一管理部门资料与部门权限，用户继承所属部门权限后仍可获得个人附加权限。</span>
        </div>
        <div className="action-group">
          <Button icon={<RefreshCw size={15} />} onClick={onRefresh} loading={isLoading}>刷新组织</Button>
          {actions.create && <Button type="primary" icon={<Plus size={15} />} onClick={openCreate}>新增部门</Button>}
        </div>
      </section>

      <section className={styles.workspace}>
        <section className={`panel-card ${styles.treePanel}`}>
          <div className="panel-heading">
            <div>
              <p className="page-kicker">组织架构</p>
              <h2>部门树</h2>
            </div>
            <span className="count-tag">{departments.length} 个节点</span>
          </div>
          <div className={styles.treeWrap}>
            {departmentTree.length ? (
              <Tree
                blockNode
                defaultExpandAll
                selectedKeys={selectedId ? [selectedId] : []}
                treeData={departmentTree}
                onSelect={(keys) => setSelectedId(keys.length ? Number(keys[0]) : null)}
              />
            ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无部门" />}
          </div>
        </section>

        <section className={`panel-card ${styles.detailPanel}`}>
          {selected ? (
            <>
              <div className={styles.detailHeader}>
                <div>
                  <p className="page-kicker">当前部门</p>
                  <h2>{selected.name}</h2>
                  <Tag color={selected.status === '启用' ? 'success' : 'default'}>{selected.status}</Tag>
                </div>
                <div className={styles.actions}>
                  <Button icon={<Eye size={15} />} onClick={() => void openUsers(selected)}>查看成员</Button>
                  <Button type={actions.permissions ? 'primary' : 'default'} icon={<ShieldCheck size={15} />} onClick={() => void openPermissions(selected)}>{actions.permissions ? '部门权限' : '查看权限'}</Button>
                  {actions.update && <Button icon={<Pencil size={15} />} onClick={() => openEdit(selected)}>编辑资料</Button>}
                  {actions.delete && <Button danger icon={<Trash2 size={15} />} onClick={() => confirmDelete(selected)}>删除</Button>}
                </div>
              </div>

              <div className={styles.summaryGrid}>
                <div className={styles.summaryItem}><span>部门编码</span><strong>{selected.code}</strong></div>
                <div className={styles.summaryItem}><span>直接下级</span><strong>{childrenCount}</strong></div>
                <div className={styles.summaryItem}><span>归属用户</span><strong><UsersRound size={17} /> {userCount}</strong></div>
                <div className={styles.summaryItem}><span>显示顺序</span><strong>{selected.sort}</strong></div>
              </div>

              <div className={styles.contactGrid}>
                <div className={styles.contactItem}><span>负责人</span><strong>{selected.leader || '暂未设置'}</strong></div>
                <div className={styles.contactItem}><span>联系电话</span><strong>{selected.phone || '暂未设置'}</strong></div>
                <div className={styles.contactItem}><span>联系邮箱</span><strong>{selected.email || '暂未设置'}</strong></div>
                <div className={styles.contactItem}><span>上级部门</span><strong>{departments.find((item) => item.id === selected.parentId)?.name ?? '无（顶级部门）'}</strong></div>
              </div>
            </>
          ) : (
            <div className={styles.emptyDetail}><Empty description="选择一个部门查看详情" /></div>
          )}
        </section>
      </section>

      <Modal
        open={editorOpen}
        title={editingId ? '编辑部门' : '新增部门'}
        footer={null}
        width={680}
        destroyOnHidden
        onCancel={() => setEditorOpen(false)}
      >
        <form className={styles.formGrid} onSubmit={submitEditor}>
          <label>
            部门名称
            <input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="例如：企业业务群" />
          </label>
          <label>
            部门编码
            <input required value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} placeholder="例如：enterprise-bg" />
          </label>
          <label className={styles.spanTwo}>
            上级部门
            <TreeSelect
              allowClear
              treeDefaultExpandAll
              treeData={parentTree}
              value={form.parentId ?? undefined}
              placeholder="不选择表示顶级部门"
              onChange={(value) => setForm({ ...form, parentId: value ? Number(value) : null })}
            />
          </label>
          <label>
            负责人
            <input value={form.leader} onChange={(event) => setForm({ ...form, leader: event.target.value })} placeholder="负责人姓名" />
          </label>
          <label>
            联系电话
            <input value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} placeholder="联系电话" />
          </label>
          <label>
            联系邮箱
            <input type="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} placeholder="department@example.com" />
          </label>
          <label>
            显示顺序
            <InputNumber min={0} precision={0} value={form.sort} onChange={(value) => setForm({ ...form, sort: Number(value ?? 0) })} />
          </label>
          <label className={styles.spanTwo}>
            状态
            <Select value={form.status} options={departmentStatusOptions.map((status) => ({ label: status, value: status }))} onChange={(status) => setForm({ ...form, status })} />
          </label>
          <div className={`rich-editor-actions ${styles.spanTwo}`}>
            <Button onClick={() => setEditorOpen(false)}>取消</Button>
            <Button type="primary" htmlType="submit" loading={isSaving}>{editingId ? '保存部门' : '创建部门'}</Button>
          </div>
        </form>
      </Modal>

      <Modal
        open={permissionOpen}
        title={`部门菜单权限${selected ? ` · ${selected.name}` : ''}`}
        okText="保存权限"
        cancelText="取消"
        confirmLoading={isSavingPermissions}
        width={600}
        destroyOnHidden
        onCancel={() => setPermissionOpen(false)}
        footer={actions.permissions ? undefined : <Button onClick={() => setPermissionOpen(false)}>关闭</Button>}
        onOk={async () => {
          if (actions.permissions && selected && await onSavePermissions(selected.id, permissionIds)) setPermissionOpen(false);
        }}
      >
        <p className={styles.permissionHint}>这里配置当前部门成员默认拥有的菜单权限；下级部门独立配置，避免顶级部门权限扩散到整个组织。</p>
        <div className={styles.permissionTree}>
          {menuTree.length ? (
            <Tree
              checkable
              disabled={!actions.permissions}
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
        open={Boolean(usersDepartment)}
        title={`部门归属用户${usersDepartment ? ` · ${usersDepartment.name}` : ''}`}
        users={associatedUsers}
        isLoading={isLoadingUsers}
        error={usersError}
        onClose={() => setUsersDepartment(null)}
      />
    </div>
  );
}
