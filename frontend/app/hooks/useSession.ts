import { useEffect, useState } from 'react';
import { API_BASE_URL } from '../lib/constants';
import { requestWithSession } from '../lib/api';
import type { AuthUser } from '../types/admin';

export function useSession() {
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [checkingSession, setCheckingSession] = useState(true);

  useEffect(() => {
    const restoreSession = async () => {
      try {
        const response = await requestWithSession(`${API_BASE_URL}/api/auth/session`, {
          cache: 'no-store',
        });
        if (response.ok) {
          const payload = (await response.json()) as { user: AuthUser };
          setCurrentUser(payload.user);
        }
      } catch (error) {
        console.error('恢复登录会话失败', error);
      } finally {
        setCheckingSession(false);
      }
    };

    restoreSession();
  }, []);

  return {
    currentUser,
    setCurrentUser,
    checkingSession,
  };
}
