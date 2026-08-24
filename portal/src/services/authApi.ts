import { API_BASE_URL } from '@/config/constants';

/** 登录用户摘要，来自后端会话接口。 */
export interface AuthUser {
  id: number;
  username: string;
  name: string;
}

/** 登录会话恢复结果，包含后端域持久化的 18R 内容偏好。 */
export interface AuthSession {
  user: AuthUser;
  r18Enabled: boolean;
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

/** 请求公开注册邮箱验证码，后端不会创建临时用户记录。 */
export async function registerCodeRequest(username: string, email: string): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/register/code', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, email }),
    });
    if (res.ok) return { ok: true };
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    return { ok: false, error: body?.error };
  } catch {
    return { ok: false };
  }
}

/** 提交邮箱验证码并创建最低权限普通用户。 */
export async function registerRequest(
  username: string,
  password: string,
  email: string,
  code: string,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, password, email, code }),
    });
    if (res.ok) return { ok: true };
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    return { ok: false, error: body?.error };
  } catch {
    return { ok: false };
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
export async function fetchSession(): Promise<AuthSession | null> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/session', {
      credentials: 'include',
    });
    if (!res.ok) {
      return null;
    }
    const body = (await res.json()) as { user?: AuthUser; r18Enabled?: boolean };
    if (!body.user) {
      return null;
    }
    return { user: body.user, r18Enabled: body.r18Enabled === true };
  } catch {
    return null;
  }
}

/** 更新当前登录会话在后端域中的 18R 内容可见性偏好。 */
export async function setPortalR18Preference(enabled: boolean): Promise<boolean> {
  try {
    const res = await fetch(API_BASE_URL + '/api/auth/portal-r18', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ enabled }),
    });
    return res.ok;
  } catch {
    return false;
  }
}
