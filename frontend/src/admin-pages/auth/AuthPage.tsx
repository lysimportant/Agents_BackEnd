'use client';

import { useEffect, useRef, type Dispatch, type FormEvent, type SetStateAction } from 'react';
import { LOGIN_BACKGROUND_CHANGE_EVENT, applyStoredLoginBackground } from '@/src/utils/loginBackground';
import type { LoginForm } from '@/src/types/admin';

type AuthPageProps = {
  /** isCheckingSession 表示登录会话。 */
  isCheckingSession: boolean;
  /** loginForm 表示登录表单。 */
  loginForm: LoginForm;
  /** loginError 表示登录错误状态。 */
  loginError: string;
  /** isLoggingIn 表示登录状态。 */
  isLoggingIn: boolean;
  /** onLoginFormChange 表示登录表单。 */
  onLoginFormChange: Dispatch<SetStateAction<LoginForm>>;
  /** onSubmit 表示提交回调。 */
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

/** AuthPage 实现对应业务逻辑。 */
export function AuthPage({ isCheckingSession, loginForm, loginError, isLoggingIn, onLoginFormChange, onSubmit }: AuthPageProps) {
  /** formRef 保存跨渲染周期使用的表单引用。 */
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    /** refreshLoginBackground 负责计算或维护登录。 */
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

  // 未聚焦输入框时按 Enter 也触发与点击登录相同的 form submit
  useEffect(() => {
    if (isCheckingSession) return;

    /** onKeyDown 负责处理对应的界面事件和状态变化。 */
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Enter' || event.defaultPrevented || event.isComposing || isLoggingIn) return;

      /** target 保存目标。 */
      const target = event.target as HTMLElement | null;
      if (target && formRef.current?.contains(target)) {
        // 焦点已在表单内时交给原生 submit（input/button 的 Enter）
        return;
      }

      /** form 保存表单。 */
      const form = formRef.current;
      if (!form) return;

      event.preventDefault();
      if (typeof form.requestSubmit === 'function') {
        form.requestSubmit();
      } else {
        form.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }));
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isCheckingSession, isLoggingIn]);

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
      <form ref={formRef} className="login-card login-card-on-art" onSubmit={onSubmit}>
        <div>
          <p className="login-kicker">账号登录</p>
          <h2>欢迎回来</h2>
          <span>请输入后台账号和密码，成功后将自动恢复工作台数据。</span>
        </div>
        <label>
          账号
          <input
            required
            autoFocus
            value={loginForm.username}
            onChange={(event) => onLoginFormChange((current) => ({ ...current, username: event.target.value }))}
            placeholder="请输入账号"
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
            placeholder="请输入密码"
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
