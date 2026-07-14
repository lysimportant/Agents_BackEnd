import { API_BASE_URL } from '../lib/constants';

type DashboardPageProps = {
  usersCount: number;
  activeUsers: number;
  menusCount: number;
  enabledMenus: number;
  publishedArticles: number;
  isLoading: boolean;
  onRefresh: () => void;
  onNavigateUsers: () => void;
  onNavigateMenus: () => void;
  onNavigateArticles: () => void;
};

export function DashboardPage({
  usersCount,
  activeUsers,
  menusCount,
  enabledMenus,
  publishedArticles,
  isLoading,
  onRefresh,
  onNavigateUsers,
  onNavigateMenus,
  onNavigateArticles,
}: DashboardPageProps) {
  return (
    <div className="page-stack">
      <section className="welcome-card">
        <div>
          <p className="page-kicker">后台首页</p>
          <h1>欢迎使用若依式管理后台</h1>
          <p>当前后台已接入真实登录、会话恢复与退出，未认证用户无法访问管理接口。</p>
        </div>
        <button className="primary-button" type="button" onClick={onRefresh} disabled={isLoading}>
          {isLoading ? '刷新中...' : '刷新数据'}
        </button>
      </section>

      <section className="stat-grid">
        <article className="stat-card blue"><span>用户总数</span><strong>{usersCount}</strong><small>系统账号</small></article>
        <article className="stat-card green"><span>在线/可调度</span><strong>{activeUsers}</strong><small>非离线用户</small></article>
        <article className="stat-card orange"><span>菜单节点</span><strong>{menusCount}</strong><small>{enabledMenus} 个启用</small></article>
        <article className="stat-card purple"><span>文章发布</span><strong>{publishedArticles}</strong><small>模拟内容</small></article>
      </section>

      <section className="content-grid two-columns">
        <article className="panel-card">
          <div className="panel-heading"><div><p className="page-kicker">快捷入口</p><h2>常用功能</h2></div></div>
          <div className="quick-grid">
            <button type="button" onClick={onNavigateUsers}>用户管理</button>
            <button type="button" onClick={onNavigateMenus}>菜单管理</button>
            <button type="button" onClick={onNavigateArticles}>文章管理</button>
          </div>
        </article>
        <article className="panel-card">
          <div className="panel-heading"><div><p className="page-kicker">接口状态</p><h2>联调信息</h2></div></div>
          <ul className="info-list">
            <li><span>后端地址</span><strong>{API_BASE_URL}</strong></li>
            <li><span>认证接口</span><strong>POST /api/auth/login</strong></li>
            <li><span>会话模式</span><strong>HttpOnly Cookie</strong></li>
          </ul>
        </article>
      </section>
    </div>
  );
}
