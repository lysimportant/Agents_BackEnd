import type { FormEvent } from 'react';
import type { DepthStyle, Menu, MenuForm, MenuNode } from '../types/admin';
import { menuStatusOptions } from '../lib/constants';

type MenusPageProps = {
  menus: Menu[];
  menuTree: MenuNode[];
  menuForm: MenuForm;
  editingMenuId: number | null;
  isLoading: boolean;
  isSavingMenu: boolean;
  onRefresh: () => void;
  onMenuFormChange: (form: MenuForm) => void;
  onSubmitMenu: (event: FormEvent<HTMLFormElement>) => void;
  onResetMenuForm: () => void;
  onEditMenu: (menu: Menu) => void;
  onDeleteMenu: (menuId: number) => void;
};

export function MenusPage({
  menus,
  menuTree,
  menuForm,
  editingMenuId,
  isLoading,
  isSavingMenu,
  onRefresh,
  onMenuFormChange,
  onSubmitMenu,
  onResetMenuForm,
  onEditMenu,
  onDeleteMenu,
}: MenusPageProps) {
  const enabledMenus = menus.filter((menu) => menu.status === '启用');
  const disabledMenus = menus.length - enabledMenus.length;

  return (
    <div className="page-stack">
      <section className="section-header-card">
        <div>
          <p className="page-kicker">系统管理</p>
          <h1>菜单管理</h1>
          <span>基于当前后端 `/api/menus` 的真实数据维护后台菜单树、权限编码与访问路径，不使用虚假菜单。</span>
        </div>
        <button className="ghost-button" type="button" onClick={onRefresh} disabled={isLoading}>
          {isLoading ? '刷新中' : '刷新'}
        </button>
      </section>

      <section className="menu-summary-grid">
        <article className="panel-card menu-summary-card">
          <strong>{menus.length}</strong>
          <span>真实菜单总数</span>
        </article>
        <article className="panel-card menu-summary-card">
          <strong>{enabledMenus.length}</strong>
          <span>启用中</span>
        </article>
        <article className="panel-card menu-summary-card">
          <strong>{disabledMenus}</strong>
          <span>停用中</span>
        </article>
      </section>

      <section className="content-grid menu-layout">
        <form className="panel-card form-panel" onSubmit={onSubmitMenu}>
          <div className="panel-heading">
            <div>
              <p className="page-kicker">菜单表单</p>
              <h2>{editingMenuId ? '编辑菜单' : '新增菜单'}</h2>
            </div>
            {editingMenuId && (
              <button className="ghost-button" type="button" onClick={onResetMenuForm}>
                取消
              </button>
            )}
          </div>
          <label>
            菜单名称
            <input required value={menuForm.name} onChange={(event) => onMenuFormChange({ ...menuForm, name: event.target.value })} placeholder="如：文件管理" />
          </label>
          <label>
            权限编码
            <input required value={menuForm.code} onChange={(event) => onMenuFormChange({ ...menuForm, code: event.target.value })} placeholder="如：files" />
          </label>
          <label>
            访问路径
            <input value={menuForm.path} onChange={(event) => onMenuFormChange({ ...menuForm, path: event.target.value })} placeholder="如：/files" />
          </label>
          <div className="form-row">
            <label>
              父级菜单
              <select
                value={menuForm.parentId ?? ''}
                onChange={(event) => onMenuFormChange({ ...menuForm, parentId: event.target.value ? Number(event.target.value) : null })}
              >
                <option value="">顶级菜单</option>
                {menus
                  .filter((menu) => menu.id !== editingMenuId)
                  .map((menu) => (
                    <option key={menu.id} value={menu.id}>
                      {menu.name}（{menu.code}）
                    </option>
                  ))}
              </select>
            </label>
            <label>
              排序
              <input type="number" value={menuForm.sort} onChange={(event) => onMenuFormChange({ ...menuForm, sort: Number(event.target.value) })} />
            </label>
          </div>
          <div className="form-row">
            <label>
              图标文案
              <input value={menuForm.icon} onChange={(event) => onMenuFormChange({ ...menuForm, icon: event.target.value })} placeholder="如：FolderOpenOutlined" />
            </label>
            <label>
              状态
              <select value={menuForm.status} onChange={(event) => onMenuFormChange({ ...menuForm, status: event.target.value })}>
                {menuStatusOptions.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <button className="primary-button" type="submit" disabled={isSavingMenu}>
            {isSavingMenu ? '提交中...' : editingMenuId ? '保存菜单' : '创建菜单'}
          </button>
        </form>

        <section className="panel-card">
          <div className="panel-heading">
            <div>
              <p className="page-kicker">菜单树</p>
              <h2>真实菜单节点</h2>
            </div>
            <span className="count-tag">{menus.length} 个节点</span>
          </div>
          <div className="menu-list">
            {menuTree.map((menu) => (
              <article className="menu-node" key={menu.id} style={{ '--depth': menu.depth } as DepthStyle}>
                <div>
                  <span className="menu-icon">{menu.icon || '菜单'}</span>
                  <strong>{menu.name}</strong>
                  <small>
                    {menu.code} · {menu.path || '无路径'} · 排序 {menu.sort}
                  </small>
                </div>
                <span className={`status-badge ${menu.status === '启用' ? 'online' : 'offline'}`}>{menu.status}</span>
                <div className="action-group">
                  <button type="button" onClick={() => onEditMenu(menu)}>
                    编辑
                  </button>
                  <button className="danger" type="button" onClick={() => onDeleteMenu(menu.id)}>
                    删除
                  </button>
                </div>
              </article>
            ))}
            {!isLoading && menus.length === 0 && <p className="empty-state">暂无真实菜单，请先新增菜单。</p>}
          </div>
        </section>
      </section>
    </div>
  );
}
