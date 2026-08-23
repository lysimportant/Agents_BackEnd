'use client';

import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';
import { registerCodeRequest, registerRequest } from '@/services/authApi';

/** RegisterModal 提供账号、密码和邮箱验证码注册流程。 */
export function RegisterModal({ onClose, onRegistered }: { onClose: () => void; onRegistered: () => void }) {
  const t = useTranslations('auth');
  const common = useTranslations('common');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    const handleKeyDown = (event: KeyboardEvent) => event.key === 'Escape' && onClose();
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = window.setInterval(() => setCooldown((value) => Math.max(0, value - 1)), 1000);
    return () => window.clearInterval(timer);
  }, [cooldown]);

  const sendCode = async () => {
    setError('');
    const result = await registerCodeRequest(username.trim(), email.trim());
    if (!result.ok) {
      setError(result.error ?? t('registerFailed'));
      return;
    }
    setSent(true);
    setCooldown(60);
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError('');
    const result = await registerRequest(username.trim(), password, email.trim(), code.trim());
    setSubmitting(false);
    if (result.ok) {
      onRegistered();
      onClose();
    } else {
      setError(result.error ?? t('registerFailed'));
    }
  };

  return createPortal(
    <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-overlay backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <form onSubmit={submit} role="dialog" aria-modal="true" className="relative w-full max-w-sm rounded-2xl border border-border bg-surface p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t('registerTitle')}</h2>
          <button type="button" onClick={onClose} aria-label={common('close')} className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted">
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
        <div className="space-y-3">
          <label className="block"><span className="mb-1 block text-sm text-muted-foreground">{t('username')}</span><input ref={inputRef} value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" /></label>
          <label className="block"><span className="mb-1 block text-sm text-muted-foreground">{t('email')}</span><input type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" /></label>
          <label className="block"><span className="mb-1 block text-sm text-muted-foreground">{t('password')}</span><input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" /></label>
          <div className="flex gap-2"><input value={code} onChange={(event) => setCode(event.target.value)} inputMode="numeric" placeholder={t('verificationCode')} className="h-11 min-w-0 flex-1 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" /><button type="button" onClick={sendCode} disabled={cooldown > 0 || !username.trim() || !email.trim()} className="h-11 shrink-0 rounded-lg border border-border px-3 text-sm transition-colors hover:bg-muted disabled:opacity-50">{cooldown > 0 ? `${cooldown}s` : t('sendCode')}</button></div>
          {sent ? <p className="text-xs text-muted-foreground">{t('codeSent')}</p> : null}
          {error ? <p className="text-sm text-red-500">{error}</p> : null}
          <button type="submit" disabled={submitting || !username.trim() || !email.trim() || !password || !code.trim()} className="h-11 w-full rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50">{t('register')}</button>
        </div>
      </form>
    </div>,
    document.body,
  );
}
