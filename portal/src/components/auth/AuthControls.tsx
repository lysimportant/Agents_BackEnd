'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Eye, EyeOff, LogIn, LogOut, UserPlus } from 'lucide-react';
import { useAuth } from '@/features/auth/AuthProvider';
import { LoginModal } from './LoginModal';
import { RegisterModal } from './RegisterModal';
import { cn } from '@/utils/cn';

/** AuthControls 在导航中提供登录/退出与 18R 内容开关。 */
export function AuthControls() {
  const t = useTranslations('auth');
  const { isLoggedIn, isR18Enabled, username, logout, toggleR18 } = useAuth();
  const [loginOpen, setLoginOpen] = useState(false);
  const [registerOpen, setRegisterOpen] = useState(false);

  if (!isLoggedIn) {
    return (
      <>
        <button
          type="button"
          onClick={() => setLoginOpen(true)}
          aria-label={t('login')}
          className="flex h-11 items-center gap-1.5 rounded-full px-3 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <LogIn className="h-4 w-4" aria-hidden="true" />
          <span className="text-sm font-medium">{t('login')}</span>
        </button>
        <button
          type="button"
          onClick={() => setRegisterOpen(true)}
          aria-label={t('register')}
          className="flex h-11 items-center gap-1.5 rounded-full px-3 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <UserPlus className="h-4 w-4" aria-hidden="true" />
          <span className="text-sm font-medium">{t('register')}</span>
        </button>
        {loginOpen ? <LoginModal onClose={() => setLoginOpen(false)} /> : null}
        {registerOpen ? <RegisterModal onClose={() => setRegisterOpen(false)} onRegistered={() => setLoginOpen(true)} /> : null}
      </>
    );
  }

  return (
    <div className="flex items-center gap-1">
      <span className="hidden max-w-[8rem] truncate text-sm text-muted-foreground sm:inline">
        {username}
      </span>
      <button
        type="button"
        onClick={() => void toggleR18()}
        aria-pressed={isR18Enabled}
        aria-label={isR18Enabled ? t('r18DisableLabel') : t('r18EnableLabel')}
        title={isR18Enabled ? t('r18DisableLabel') : t('r18EnableLabel')}
        className={cn(
          'flex h-11 w-11 items-center justify-center rounded-full transition-colors',
          isR18Enabled
            ? 'bg-primary text-primary-foreground'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground',
        )}
      >
        {isR18Enabled ? (
          <Eye className="h-5 w-5" aria-hidden="true" />
        ) : (
          <EyeOff className="h-5 w-5" aria-hidden="true" />
        )}
      </button>
      <button
        type="button"
        onClick={() => logout()}
        aria-label={t('logout')}
        className="flex h-11 items-center gap-1.5 rounded-full px-3 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <LogOut className="h-4 w-4" aria-hidden="true" />
        <span className="text-sm font-medium">{t('logout')}</span>
      </button>
    </div>
  );
}
