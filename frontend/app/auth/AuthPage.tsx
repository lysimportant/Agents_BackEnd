import type { FormEvent } from 'react';
import type { LoginForm } from '../types/admin';

type AuthPageProps = {
  isCheckingSession: boolean;
  loginForm: LoginForm;
  loginError: string;
  isLoggingIn: boolean;
  onLoginFormChange: (form: LoginForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

export function AuthPage({ isCheckingSession, loginForm, loginError, isLoggingIn, onLoginFormChange, onSubmit }: AuthPageProps) {
  if (isCheckingSession) {
    return (
      <main className="auth-shell">
        <section className="login-card loading-card">
          <div className="login-mascot">🌈</div>
          <h1>正在恢复会话...</h1>
          <p>正在确认你的登录状态，请稍候。</p>
        </section>
      </main>
    );
  }

  return (
    <main className="auth-shell">
      <div className="anime-orb orb-one" />
      <div className="anime-orb orb-two" />
      <div className="anime-orb orb-three" />
      <section className="login-hero">
        <p className="login-kicker">Kawaii Secure Console</p>
        <h1>多彩二次元登录页</h1>
        <p>登录后才能进入 MES 后台。默认管理员账号已预置为 MH / 123。</p>
        <div className="anime-scene" aria-hidden="true">
          <span className="anime-character">🧚‍♀️</span>
          <span className="anime-star star-a">✦</span>
          <span className="anime-star star-b">✧</span>
          <span className="anime-star star-c">✺</span>
        </div>
      </section>
      <form className="login-card" onSubmit={onSubmit}>
        <div>
          <p className="login-kicker">账号登录</p>
          <h2>欢迎回来</h2>
          <span>请输入后台账号和密码，成功后将自动恢复工作台数据。</span>
        </div>
        <label>
          账号
          <input
            required
            value={loginForm.username}
            onChange={(event) => onLoginFormChange({ ...loginForm, username: event.target.value })}
            placeholder="MH"
            autoComplete="username"
          />
        </label>
        <label>
          密码
          <input
            required
            type="password"
            value={loginForm.password}
            onChange={(event) => onLoginFormChange({ ...loginForm, password: event.target.value })}
            placeholder="123"
            autoComplete="current-password"
          />
        </label>
        {loginError && <p className="error-message">{loginError}</p>}
        <button className="primary-button login-button" type="submit" disabled={isLoggingIn}>
          {isLoggingIn ? '登录中...' : '进入后台'}
        </button>
        <p className="login-tip">密码通过 bcrypt 校验，接口不会返回 password 或 passwordHash。</p>
      </form>
    </main>
  );
}
