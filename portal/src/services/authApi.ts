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
