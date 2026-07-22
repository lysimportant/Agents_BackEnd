'use client';

import { useEffect, type Dispatch, type FormEvent, type SetStateAction } from 'react';
import { LOGIN_BACKGROUND_CHANGE_EVENT, applyStoredLoginBackground } from '@/src/utils/loginBackground';
import type { LoginForm } from '@/src/types/admin';

type AuthPageProps = {
  isCheckingSession: boolean;
  loginForm: LoginForm;
  loginError: string;
  isLoggingIn: boolean;
  onLoginFormChange: Dispatch<SetStateAction<LoginForm>>;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

export function AuthPage({ isCheckingSession, loginForm, loginError, isLoggingIn, onLoginFormChange, onSubmit }: AuthPageProps) {
  useEffect(() => {
    const refreshLoginBackground = () => {
      applyStoredLoginBackground();
    };

    refreshLoginBackground();
    window.addEventListener('storage', refreshLoginBackground);
    window.addEventListener(LOGIN_BACKGROUND_CHANGE_EVENT, refreshLoginBackground);

    return () => {
      window.removeEventListener('storage', refreshLoginBackground);
      window.removeEventListener(LOGIN_BACKGROUND_CHANGE_EVENT, refreshLoginBackground);
    };
  }, []);

  if (isCheckingSession) {
    return (
      <main className="auth-shell auth-shell-with-art">
        <section className="login-card login-card-on-art loading-card">
          <div className="login-mascot">🌈</div>
          <h1>正在恢复会话...</h1>
          <p>正在确认你的登录状态，请稍候。</p>
        </section>
      </main>
    );
  }

  return (
    <main className="auth-shell auth-shell-with-art">
      <form className="login-card login-card-on-art" onSubmit={onSubmit}>
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
            onChange={(event) => onLoginFormChange((current) => ({ ...current, username: event.target.value }))}
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
            onChange={(event) => onLoginFormChange((current) => ({ ...current, password: event.target.value }))}
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
