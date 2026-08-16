'use client';

import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';
import { useAuth } from '@/features/auth/AuthProvider';

/** LoginModal 提供 C 端登录表单，复用后端 /api/auth/login。 */
export function LoginModal({ onClose }: { onClose: () => void }) {
  const t = useTranslations('auth');
  const common = useTranslations('common');
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // 打开时聚焦账号输入，Esc 关闭。
  useEffect(() => {
    inputRef.current?.focus();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError(false);
    const ok = await login(username.trim(), password);
    setSubmitting(false);
    if (ok) {
      onClose();
    } else {
      setError(true);
    }
  };

  // 通过 Portal 渲染到 body，避免被带 backdrop-filter 的导航栏影响 fixed 定位。
  return createPortal(
    <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-overlay backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <form
        onSubmit={submit}
        role="dialog"
        aria-modal="true"
        className="relative w-full max-w-sm rounded-2xl border border-border bg-surface p-6 shadow-2xl"
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t('loginTitle')}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={common('close')}
            className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
        <div className="space-y-3">
          <label className="block">
            <span className="mb-1 block text-sm text-muted-foreground">{t('username')}</span>
            <input
              ref={inputRef}
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              autoComplete="username"
              className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none transition-colors focus:border-primary"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-muted-foreground">{t('password')}</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none transition-colors focus:border-primary"
            />
          </label>
          {error ? <p className="text-sm text-red-500">{t('loginFailed')}</p> : null}
          <button
            type="submit"
            disabled={submitting || !username.trim() || !password}
            className="h-11 w-full rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50"
          >
            {t('login')}
          </button>
        </div>
      </form>
    </div>,
    document.body,
  );
}
