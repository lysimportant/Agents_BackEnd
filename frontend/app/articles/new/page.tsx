'use client';

import { useEffect, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Result } from 'antd';
import { ArticleCreatePage } from '@/src/admin-pages/articles/ArticleCreatePage';
import { AuthPage } from '@/src/admin-pages/auth/AuthPage';
import { MainLayout } from '@/src/components/layout/MainLayout';
import { useAdminWorkspace } from '@/src/features/workspace/useAdminWorkspace';
import type { PageKey } from '@/src/types/admin';
import { isAdministratorRoleCode } from '@/src/utils/roleAccess';

/** NewArticleRoute 为 B 端文章创建提供可直接访问的独立路由。 */
export default function NewArticleRoute() {
  /** router 负责在创建页与根工作台之间切换真实路由。 */
  const router = useRouter();
  /** workspace 提供会话恢复、文章表单和写入操作。 */
  const workspace = useAdminWorkspace();

  useEffect(() => {
    if (workspace.authUser) workspace.handleNavigate('articles');
  }, [workspace.authUser]);

  if (!workspace.authUser) {
    return <AuthPage
      isCheckingSession={workspace.isCheckingSession}
      loginForm={workspace.loginForm}
      loginError={workspace.loginError}
      isLoggingIn={workspace.isLoggingIn}
      onLoginFormChange={workspace.setLoginForm}
      onSubmit={workspace.handleLogin}
    />;
  }

  /** authUser 保存当前已恢复的后台登录用户。 */
  const authUser = workspace.authUser;
  /** canCreateArticle 表示用户是否拥有后端对应的文章创建动作。 */
  const canCreateArticle = isAdministratorRoleCode(authUser.roleCode)
    || authUser.actionPermissions?.includes('articles.create') === true;

  /** returnToWorkspace 记录目标菜单后返回根工作台。 */
  const returnToWorkspace = (page: PageKey = 'articles') => {
    workspace.handleNavigate(page);
    router.push('/');
  };

  /** cancelCreate 清空未提交表单并返回文章列表。 */
  const cancelCreate = () => {
    workspace.resetArticleForm();
    returnToWorkspace();
  };

  /** submitCreate 创建成功后替换当前历史记录，避免返回键再次进入空表单。 */
  const submitCreate = async (event: FormEvent<HTMLFormElement>) => {
    if (!canCreateArticle) {
      event.preventDefault();
      return;
    }
    if (await workspace.handleSubmitArticle(event)) {
      workspace.handleNavigate('articles');
      router.replace('/');
    }
  };

  /** navigateFromCreatePage 让侧栏导航离开独立路由并打开目标工作台页面。 */
  const navigateFromCreatePage = (page: PageKey) => {
    workspace.resetArticleForm();
    returnToWorkspace(page);
  };

  return <MainLayout
    authUser={authUser}
    menus={workspace.menus}
    activePage="articles"
    sidebarCollapsed={workspace.sidebarCollapsed}
    mobileSidebarOpen={workspace.mobileSidebarOpen}
    error={workspace.error}
    onToggleSidebar={() => workspace.setSidebarCollapsed((current) => !current)}
    onOpenMobileSidebar={() => workspace.setMobileSidebarOpen(true)}
    onCloseMobileSidebar={() => workspace.setMobileSidebarOpen(false)}
    onNavigate={navigateFromCreatePage}
    onLogout={workspace.handleLogout}
  >
    {canCreateArticle
      ? <ArticleCreatePage
          articleForm={workspace.articleForm}
          isSavingArticle={workspace.isSavingArticle}
          onArticleFormChange={workspace.setArticleForm}
          onSubmitArticle={submitCreate}
          onCancel={cancelCreate}
        />
      : <Result
          status="403"
          title="无权创建文章"
          subTitle="当前账号没有 articles.create 操作权限。"
          extra={<Button type="primary" onClick={() => returnToWorkspace()}>返回文章管理</Button>}
        />}
  </MainLayout>;
}
