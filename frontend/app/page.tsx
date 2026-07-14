'use client';

import { ArticlesPage } from './articles/ArticlesPage';
import { AuthPage } from './auth/AuthPage';
import { DashboardPage } from './dashboard/DashboardPage';
import { FilesPage } from './files/FilesPage';
import { useAdminWorkspace } from './hooks/useAdminWorkspace';
import { MainLayout } from './layout/MainLayout';
import { MenusPage } from './menus/MenusPage';
import { UsersPage } from './users/UsersPage';

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
          activeUsers={workspace.users.filter((user) => user.status !== '离线').length}
          menusCount={workspace.menus.length}
          enabledMenus={workspace.menus.filter((menu) => menu.status === '启用').length}
          publishedArticles={workspace.articles.filter((article) => article.status === '已发布').length}
          isLoading={workspace.isLoading}
          onRefresh={workspace.loadData}
          onNavigateUsers={() => workspace.handleNavigate('users')}
          onNavigateMenus={() => workspace.handleNavigate('menus')}
          onNavigateArticles={() => workspace.handleNavigate('articles')}
        />
      )}

      {workspace.activePage === 'users' && (
        <UsersPage
          users={workspace.users}
          menus={workspace.menus}
          userForm={workspace.userForm}
          editingUserId={workspace.editingUserId}
          selectedUserId={workspace.selectedUserId}
          selectedMenuIds={workspace.selectedMenuIds}
          isLoading={workspace.isLoading}
          isSavingUser={workspace.isSavingUser}
          isSavingPermission={workspace.isSavingPermission}
          onRefresh={workspace.loadData}
          onUserFormChange={workspace.setUserForm}
          onSubmitUser={workspace.handleSubmitUser}
          onResetUserForm={workspace.resetUserForm}
          onSelectUser={workspace.handleSelectUser}
          onEditUser={workspace.handleEditUser}
          onDeleteUser={workspace.handleDeleteUser}
          onToggleMenuPermission={workspace.handleToggleMenuPermission}
          onSavePermissions={workspace.handleSavePermissions}
        />
      )}

      {workspace.activePage === 'menus' && (
        <MenusPage
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

      {workspace.activePage === 'articles' && (
        <ArticlesPage
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
    </MainLayout>
  );
}
