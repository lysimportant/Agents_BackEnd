import { API_BASE_URL } from '@/config/constants';

/** 登录用户摘要，来自后端会话接口。 */
export interface AuthUser {
  id: number;
  username: string;
  name: string;
}

/**
 * 调用后端登录接口，成功时后端会写入 HttpOnly 会话 Cookie。
 */
export async function loginRequest(username: string, password: string): Promise<boolean> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

/** 调用后端登出接口，清除会话 Cookie。 */
export async function logoutRequest(): Promise<void> {
  try {
    await fetch(API_BASE_URL + '/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    });
  } catch {
    // 登出失败不阻塞本地状态清理。
  }
}

/** 查询当前登录会话，未登录或请求失败返回 null。 */
export async function fetchSession(): Promise<AuthUser | null> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/session', {
      credentials: 'include',
    });
    if (!res.ok) {
      return null;
    }
    const body = (await res.json()) as { user?: AuthUser };
    return body.user ?? null;
  } catch {
    return null;
  }
}
