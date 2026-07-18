'use client';

import { ArticlesPage } from './articles/ArticlesPage';
import type { ResourceActionAccess } from './lib/actionPermissions';
import { isAdministratorRoleCode } from './lib/roleAccess';
import { AuthPage } from './auth/AuthPage';
import { DashboardPage } from './dashboard/DashboardPage';
import { DepartmentsPage } from './departments/DepartmentsPage';
import { FilesPage } from './files/FilesPage';
import { useAdminWorkspace } from './hooks/useAdminWorkspace';
import { MainLayout } from './layout/MainLayout';
import { MenusPage } from './menus/MenusPage';
import { ProfilePage } from './profile/ProfileDialog';
import { RolesPage } from './roles/RolesPage';
import { UsersPage } from './users/UsersPage';
import { SocketSupportPage } from './socket/SocketSupportPage';

export default function Home() {
  const workspace = useAdminWorkspace();

  if (!workspace.authUser) {
    return (
      <AuthPage
        isCheckingSession={workspace.isCheckingSession}
        loginForm={workspace.loginForm}
        loginError={workspace.loginError}
        isLoggingIn={workspace.isLoggingIn}
        onLoginFormChange={workspace.setLoginForm}
        onSubmit={workspace.handleLogin}
      />
    );
  }

  const authUser = workspace.authUser;
  const hasAction = (actionCode: string) => isAdministratorRoleCode(authUser.roleCode)
    || authUser.actionPermissions?.includes(actionCode) === true;
  const roleActions: ResourceActionAccess = {
    create: hasAction('roles.create'),
    update: hasAction('roles.update'),
    delete: hasAction('roles.delete'),
    permissions: hasAction('roles.permissions.update'),
  };
  const menuActions: ResourceActionAccess = {
    create: hasAction('menus.create'),
    update: hasAction('menus.update'),
    delete: hasAction('menus.delete'),
  };
  const departmentActions: ResourceActionAccess = {
    create: hasAction('departments.create'),
    update: hasAction('departments.update'),
    delete: hasAction('departments.delete'),
    permissions: hasAction('departments.permissions.update'),
  };
  const articleActions: ResourceActionAccess = {
    create: hasAction('articles.create'),
    update: hasAction('articles.update'),
    delete: hasAction('articles.delete'),
  };
  const fileActions: ResourceActionAccess = {
    create: hasAction('files.create'),
    update: hasAction('files.update'),
    delete: hasAction('files.delete'),
    restore: hasAction('files.restore'),
    permanentDelete: hasAction('files.permanent-delete'),
  };

  return (
    <MainLayout
      authUser={workspace.authUser}
      menus={workspace.menus}
      activePage={workspace.activePage}
      sidebarCollapsed={workspace.sidebarCollapsed}
      mobileSidebarOpen={workspace.mobileSidebarOpen}
      error={workspace.error}
      onToggleSidebar={() => workspace.setSidebarCollapsed((current) => !current)}
      onOpenMobileSidebar={() => workspace.setMobileSidebarOpen(true)}
      onCloseMobileSidebar={() => workspace.setMobileSidebarOpen(false)}
      onNavigate={workspace.handleNavigate}
      onLogout={workspace.handleLogout}
    >
      {workspace.activePage === 'dashboard' && (
        <DashboardPage
          usersCount={workspace.users.length}
          activeUsers={workspace.users.filter((user) => user.canLogin && user.status !== '停用').length}
          menusCount={workspace.menus.length}
          enabledMenus={workspace.menus.filter((menu) => menu.status === '启用').length}
          articlesCount={workspace.articles.length}
          publishedArticles={workspace.articles.filter((article) => article.status === '已发布').length}
          isLoading={workspace.isLoading}
          onRefresh={workspace.loadData}
        />
      )}

      {workspace.activePage === 'socket-support' && (
        <SocketSupportPage canSend={hasAction('socket.send')} />
      )}

      {workspace.activePage === 'users' && (
        <UsersPage
          canCreate={hasAction('users.create')}
          canUpdate={hasAction('users.update')}
          canDelete={hasAction('users.delete')}
          canConfigurePermissions={isAdministratorRoleCode(authUser.roleCode) && hasAction('users.permissions.update')}
          actorRoleCode={authUser.roleCode}
          users={workspace.users}
          menus={workspace.menus}
          departments={workspace.departments}
          roles={workspace.roles}
          userForm={workspace.userForm}
          editingUserId={workspace.editingUserId}
          selectedUserId={workspace.selectedUserId}
          selectedMenuIds={workspace.selectedMenuIds}
          departmentMenuIds={workspace.departmentMenuIds}
          roleMenuIds={workspace.roleMenuIds}
          effectiveMenuIds={workspace.effectiveMenuIds}
          roleActionCodes={workspace.roleActionCodes}
          userActionCodes={workspace.userActionCodes}
          effectiveActionCodes={workspace.effectiveActionCodes}
          isLoading={workspace.isLoading}
          isSavingUser={workspace.isSavingUser}
          isSavingPermission={workspace.isSavingPermission}
          isSavingActionPermission={workspace.isSavingActionPermission}
          onRefresh={workspace.loadData}
          onUserFormChange={workspace.setUserForm}
          onSubmitUser={workspace.handleSubmitUser}
          onResetUserForm={workspace.resetUserForm}
          onSelectUser={workspace.handleSelectUser}
          onEditUser={workspace.handleEditUser}
          onDeleteUser={workspace.handleDeleteUser}
          onSavePermissions={workspace.handleSavePermissions}
          onSaveActionPermissions={workspace.handleSaveActionPermissions}
        />
      )}

      {workspace.activePage === 'roles' && (
        <RolesPage
          actorRoleCode={authUser.roleCode}
          actions={roleActions}
          roles={workspace.roles}
          users={workspace.users}
          menus={workspace.menus}
          isLoading={workspace.isLoading}
          isSaving={workspace.isSavingRole}
          isSavingPermissions={workspace.isSavingRolePermission}
          onRefresh={workspace.loadData}
          onSave={workspace.handleSaveRole}
          onDelete={workspace.handleDeleteRole}
          onLoadPermissions={workspace.loadRolePermissions}
          onLoadUsers={workspace.loadRoleUsers}
          onSavePermissions={workspace.handleSaveRolePermissions}
        />
      )}

      {workspace.activePage === 'menus' && (
        <MenusPage
          actions={menuActions}
          menus={workspace.menus}
          menuTree={workspace.menuTree}
          menuForm={workspace.menuForm}
          editingMenuId={workspace.editingMenuId}
          isLoading={workspace.isLoading}
          isSavingMenu={workspace.isSavingMenu}
          onRefresh={workspace.loadData}
          onMenuFormChange={workspace.setMenuForm}
          onSubmitMenu={workspace.handleSubmitMenu}
          onResetMenuForm={workspace.resetMenuForm}
          onEditMenu={workspace.handleEditMenu}
          onDeleteMenu={workspace.handleDeleteMenu}
        />
      )}

      {workspace.activePage === 'departments' && (
        <DepartmentsPage
          actions={departmentActions}
          departments={workspace.departments}
          users={workspace.users}
          menus={workspace.menus}
          isLoading={workspace.isLoading}
          isSaving={workspace.isSavingDepartment}
          isSavingPermissions={workspace.isSavingDepartmentPermission}
          onRefresh={workspace.loadData}
          onSave={workspace.handleSaveDepartment}
          onDelete={workspace.handleDeleteDepartment}
          onLoadPermissions={workspace.loadDepartmentPermissions}
          onLoadUsers={workspace.loadDepartmentUsers}
          onSavePermissions={workspace.handleSaveDepartmentPermissions}
        />
      )}

      {workspace.activePage === 'articles' && (
        <ArticlesPage
          actions={articleActions}
          filteredArticles={workspace.filteredArticles}
          articleForm={workspace.articleForm}
          editingArticleId={workspace.editingArticleId}
          articleKeyword={workspace.articleKeyword}
          articleStatus={workspace.articleStatus}
          isSavingArticle={workspace.isSavingArticle}
          onArticleFormChange={workspace.setArticleForm}
          onSubmitArticle={workspace.handleSubmitArticle}
          onResetArticleForm={workspace.resetArticleForm}
          onArticleKeywordChange={workspace.setArticleKeyword}
          onArticleStatusChange={workspace.setArticleStatus}
          onResetFilters={() => {
            workspace.setArticleKeyword('');
            workspace.setArticleStatus('全部');
          }}
          onEditArticle={workspace.handleEditArticle}
          onToggleArticleStatus={workspace.handleToggleArticleStatus}
          onDeleteArticle={workspace.handleDeleteArticle}
        />
      )}

      {workspace.activePage === 'files' && (
        <FilesPage
          actions={fileActions}
          filteredFiles={workspace.filteredFiles}
          recycleFiles={workspace.recycleFiles}
          fileForm={workspace.fileForm}
          selectedUploadFile={workspace.selectedUploadFile}
          editingFileId={workspace.editingFileId}
          fileKeyword={workspace.fileKeyword}
          isSavingFile={workspace.isSavingFile}
          onFileFormChange={workspace.setFileForm}
          onSelectUploadFile={workspace.handleSelectUploadFile}
          onSubmitFile={workspace.handleSubmitFile}
          onResetFileForm={workspace.resetFileForm}
          onFileKeywordChange={workspace.setFileKeyword}
          onEditFile={workspace.handleEditFile}
          onDownloadFile={workspace.handleDownloadFile}
          onDeleteFile={workspace.handleDeleteFile}
          onRestoreFile={workspace.handleRestoreFile}
          onLoadRecycleFiles={workspace.loadRecycleFiles}
          onRefreshFiles={workspace.loadData}
        />
      )}

      {workspace.activePage === 'profile' && (
        <ProfilePage
          authUser={authUser}
          onUpdated={workspace.handleAuthUserUpdate}
          onPasswordChanged={workspace.handleLogout}
        />
      )}
    </MainLayout>
  );
}
