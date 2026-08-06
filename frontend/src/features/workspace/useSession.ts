import { useEffect, useState } from 'react';
import { API_BASE_URL } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';
import type { AuthUser } from '@/src/types/admin';

/** useSession 实现对应业务逻辑。 */
export function useSession() {
  /** currentUser、setCurrentUser 保存当前用户、当前用户。 */
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  /** checkingSession、setCheckingSession 分别保存登录会话状态及其更新函数。 */
  const [checkingSession, setCheckingSession] = useState(true);

  useEffect(() => {
    /** restoreSession 负责计算或维护登录会话。 */
    const restoreSession = async () => {
      try {
        /** response 保存接口响应及其关联状态。 */
        const response = await requestWithSession(`${API_BASE_URL}/api/auth/session`, {
          cache: 'no-store',
        });
        if (response.ok) {
          /** payload 保存请求载荷。 */
          const payload = (await response.json()) as { user: AuthUser };
          setCurrentUser(payload.user);
        }
      /** error 保存当前操作结果以及可能返回的错误状态。 */
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
