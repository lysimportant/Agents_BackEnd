import { useMemo, useState, type FormEvent } from 'react';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Descriptions, Modal, Popconfirm, Space, Tag } from 'antd';
import type { ResourceActionAccess } from '@/src/utils/actionPermissions';
import type { Menu, MenuForm, MenuNode } from '@/src/types/admin';
import { menuStatusOptions } from '@/src/config/constants';

type MenusPageProps = {
  /** menus 表示菜单。 */
  menus: Menu[];
  /** menuTree 表示菜单树形数据。 */
  menuTree: MenuNode[];
  /** menuForm 表示菜单表单。 */
  menuForm: MenuForm;
  /** editingMenuId 表示菜单标识。 */
  editingMenuId: number | null;
  /** isLoading 表示加载状态。 */
  isLoading: boolean;
  /** isSavingMenu 表示菜单。 */
  isSavingMenu: boolean;
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** onRefresh 表示刷新回调。 */
  onRefresh: () => void;
  /** onMenuFormChange 表示菜单表单。 */
  onMenuFormChange: (form: MenuForm) => void;
  /** onSubmitMenu 表示菜单。 */
  onSubmitMenu: (event: FormEvent<HTMLFormElement>) => Promise<boolean>;
  /** onResetMenuForm 表示菜单表单。 */
  onResetMenuForm: () => void;
  /** onEditMenu 表示菜单。 */
  onEditMenu: (menu: Menu) => void;
  /** onDeleteMenu 表示菜单。 */
  onDeleteMenu: (menuId: number) => void;
};

/** MenusPage 保存模块使用的固定配置或共享状态。 */
export function MenusPage({ menus, menuTree, menuForm, editingMenuId, isLoading, isSavingMenu, actions, onRefresh, onMenuFormChange, onSubmitMenu, onResetMenuForm, onEditMenu, onDeleteMenu }: MenusPageProps) {
  /** dialogOpen、setDialogOpen 分别保存对话框状态及其更新函数。 */
  const [dialogOpen, setDialogOpen] = useState(false);
  /** viewingMenu、setViewingMenu 保存菜单、菜单。 */
  const [viewingMenu, setViewingMenu] = useState<Menu | null>(null);
  /** enabledMenus 负责计算或维护已启用菜单。 */
  const enabledMenus = menus.filter((menu) => menu.status === '启用');
  /** tableRows 保存数据行。 */
  const tableRows = menuTree.flatMap(function flatten(node): MenuNode[] {
    return [node, ...node.children.flatMap(flatten)];
  });
  /** parentOptions 缓存计算得到的父级选项。 */
  const parentOptions = useMemo(() => {
    if (!editingMenuId) return menus;
    /** blocked 保存阻止状态。 */
    const blocked = new Set<number>([editingMenuId]);
    /** collectDescendants 负责计算或维护收集结果后代节点。 */
    const collectDescendants = (parentId: number) => {
      menus.filter((menu) => menu.parentId === parentId).forEach((menu) => {
        if (blocked.has(menu.id)) return;
        blocked.add(menu.id);
        collectDescendants(menu.id);
      });
    };
    collectDescendants(editingMenuId);
    return menus.filter((menu) => !blocked.has(menu.id));
  }, [editingMenuId, menus]);

  /** openCreate 负责计算或维护打开状态。 */
  const openCreate = () => {
    if (!actions.create) return;
    onResetMenuForm();
    setDialogOpen(true);
  };
  /** openEdit 负责计算或维护打开状态。 */
  const openEdit = (menu: Menu) => {
    if (!actions.update) return;
    onEditMenu(menu);
    setDialogOpen(true);
  };
  /** closeDialog 负责删除或清理对应业务状态。 */
  const closeDialog = () => {
    setDialogOpen(false);
    onResetMenuForm();
  };
  /** submit 负责执行对应业务操作。 */
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    if (editingMenuId ? !actions.update : !actions.create) return;
    if (await onSubmitMenu(event)) setDialogOpen(false);
  };

  return (
    <div className="page-stack">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">系统管理</p>
          <h1>菜单管理</h1>
          <span>维护后端真实菜单树、权限编码与访问路径。</span>
        </div>
        <Space className="page-header-actions" size={12} wrap>
          <Button color="cyan" variant="filled" icon={<ReloadOutlined />} onClick={onRefresh} loading={isLoading}>刷新菜单</Button>
          {actions.create && <Button className="menu-create-button" type="primary" icon={<PlusOutlined />} aria-label="新建菜单" onClick={openCreate}>新建菜单</Button>}
        </Space>
      </section>

      <section className="menu-summary-grid">
        <article className="panel-card menu-summary-card"><strong>{menus.length}</strong><span>菜单总数</span></article>
        <article className="panel-card menu-summary-card"><strong>{enabledMenus.length}</strong><span>启用中</span></article>
        <article className="panel-card menu-summary-card"><strong>{menus.length - enabledMenus.length}</strong><span>停用中</span></article>
      </section>

      <section className="panel-card menu-table-card">
        <div className="panel-heading">
          <div><p className="page-kicker">菜单结构</p><h2>全部菜单节点</h2></div>
          <span className="count-tag">{menus.length} 个节点</span>
        </div>
        <div className="table-wrap">
          <table>
            <thead><tr><th>菜单名称</th><th>编码</th><th>路径</th><th>排序</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {tableRows.map((menu) => (
                <tr key={menu.id}>
                  <td><div className="menu-tree-cell" style={{ paddingInlineStart: `${menu.depth * 24}px` }}><span className="menu-tree-branch">{menu.depth ? '└' : '●'}</span><strong>{menu.name}</strong></div></td>
                  <td><code>{menu.code}</code></td>
                  <td>{menu.path || '—'}</td>
                  <td>{menu.sort}</td>
                  <td><Tag color={menu.status === '启用' ? 'success' : 'default'}>{menu.status}</Tag></td>
                  <td>
                    <div className="action-group">
                      <button type="button" onClick={() => setViewingMenu(menu)}>查看</button>
                      {actions.update && <button type="button" onClick={() => openEdit(menu)}>编辑</button>}
                      {actions.delete && (
                        <Popconfirm
                          title="确认删除该菜单？"
                          description={`菜单“${menu.name}”删除后不可恢复。`}
                          okText="确认删除"
                          cancelText="取消"
                          okButtonProps={{ danger: true }}
                          onConfirm={() => onDeleteMenu(menu.id)}
                        >
                          <button className="danger" type="button">删除</button>
                        </Popconfirm>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!isLoading && menus.length === 0 && <p className="empty-state">暂无真实菜单。</p>}
        </div>
      </section>

      {(actions.create || actions.update) && <Modal open={dialogOpen} title={editingMenuId ? '编辑菜单' : '创建菜单'} footer={null} onCancel={closeDialog} destroyOnHidden width={620} className="menu-form-modal">
        <form className="form-panel modal-form" onSubmit={(event) => void submit(event)}>
          <label>菜单名称<input required value={menuForm.name} onChange={(event) => onMenuFormChange({ ...menuForm, name: event.target.value })} placeholder="如：文件管理" /></label>
          <label>权限编码<input required value={menuForm.code} onChange={(event) => onMenuFormChange({ ...menuForm, code: event.target.value })} placeholder="如：files" /></label>
          <label>访问路径<input value={menuForm.path} onChange={(event) => onMenuFormChange({ ...menuForm, path: event.target.value })} placeholder="如：files" /></label>
          <div className="form-row">
            <label>父级菜单<select value={menuForm.parentId ?? ''} onChange={(event) => onMenuFormChange({ ...menuForm, parentId: event.target.value ? Number(event.target.value) : null })}><option value="">顶级菜单</option>{parentOptions.map((menu) => <option key={menu.id} value={menu.id}>{menu.name}（{menu.code}）</option>)}</select></label>
            <label>排序<input type="number" value={menuForm.sort} onChange={(event) => onMenuFormChange({ ...menuForm, sort: Number(event.target.value) })} /></label>
          </div>
          <div className="form-row">
            <label>图标<input value={menuForm.icon} onChange={(event) => onMenuFormChange({ ...menuForm, icon: event.target.value })} placeholder="如：folder-open" /></label>
            <label>状态<select value={menuForm.status} onChange={(event) => onMenuFormChange({ ...menuForm, status: event.target.value })}>{menuStatusOptions.map((status) => <option key={status} value={status}>{status}</option>)}</select></label>
          </div>
          <div className="modal-form-actions"><Button onClick={closeDialog}>取消</Button><Button type="primary" htmlType="submit" loading={isSavingMenu}>{editingMenuId ? '保存菜单' : '创建菜单'}</Button></div>
        </form>
      </Modal>}

      <Modal open={Boolean(viewingMenu)} title="菜单详情" footer={<Button onClick={() => setViewingMenu(null)}>关闭</Button>} onCancel={() => setViewingMenu(null)} destroyOnHidden width={560}>
        {viewingMenu && (
          <Descriptions bordered size="small" column={1}>
            <Descriptions.Item label="菜单名称">{viewingMenu.name}</Descriptions.Item>
            <Descriptions.Item label="权限编码">{viewingMenu.code}</Descriptions.Item>
            <Descriptions.Item label="访问路径">{viewingMenu.path || '未设置'}</Descriptions.Item>
            <Descriptions.Item label="父级菜单">{menus.find((menu) => menu.id === viewingMenu.parentId)?.name || '顶级菜单'}</Descriptions.Item>
            <Descriptions.Item label="图标">{viewingMenu.icon || '未设置'}</Descriptions.Item>
            <Descriptions.Item label="排序">{viewingMenu.sort}</Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={viewingMenu.status === '启用' ? 'success' : 'default'}>{viewingMenu.status}</Tag></Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}
