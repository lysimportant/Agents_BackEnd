/** DEFAULT_LOGIN_BACKGROUND_URL 保存模块使用的固定配置或共享状态。 */
export const DEFAULT_LOGIN_BACKGROUND_URL = '/images/login-anime-background-beach-v4.webp';

/** LOGIN_BACKGROUND_STORAGE_KEY 保存模块使用的固定配置或共享状态。 */
export const LOGIN_BACKGROUND_STORAGE_KEY = 'collector:login-background-url';

/** LOGIN_BACKGROUND_CHANGE_EVENT 保存模块使用的固定配置或共享状态。 */
export const LOGIN_BACKGROUND_CHANGE_EVENT = 'collector:login-background-change';

/** LOGIN_BACKGROUND_BOOTSTRAP_SCRIPT 保存模块使用的固定配置或共享状态。 */
export const LOGIN_BACKGROUND_BOOTSTRAP_SCRIPT = `
(function () {
  try {
    /** raw 保存变量 raw。 */
    var raw = window.localStorage.getItem('${LOGIN_BACKGROUND_STORAGE_KEY}');
    if (!raw) return;

    /** url 保存地址。 */
    var url = '';
    try {
      /** parsed 保存解析结果。 */
      var parsed = JSON.parse(raw);
      if (parsed && typeof parsed.url === 'string') {
        url = parsed.url.trim();
      }
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      url = raw.trim();
    }

    if (!url) return;
    document.documentElement.style.setProperty('--login-background-image', 'url(' + JSON.stringify(url) + ')');
  /** error 保存当前操作结果以及可能返回的错误状态。 */
  } catch (error) {
    document.documentElement.style.removeProperty('--login-background-image');
  }
})();
`;

/** LoginBackgroundPayload 定义对应业务的数据结构与调用契约。 */
export type LoginBackgroundPayload = {
  /** url 表示地址。 */
  url: string;
  /** name 表示名称。 */
  name?: string;
  /** source 表示来源。 */
  source?: 'file-manager' | 'default';
  /** mimeType 表示媒体类型类型。 */
  mimeType?: string;
  /** size 表示大小。 */
  size?: number;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** canUseBrowserStorage 校验对应业务条件。 */
function canUseBrowserStorage() {
  return typeof window !== 'undefined' && typeof document !== 'undefined';
}

/** parseStoredLoginBackground 解析对应业务数据。 */
function parseStoredLoginBackground(raw: string | null): LoginBackgroundPayload | null {
  if (!raw) return null;

  try {
    /** parsed 保存解析结果。 */
    const parsed = JSON.parse(raw) as Partial<LoginBackgroundPayload>;
    if (typeof parsed.url === 'string' && parsed.url.trim()) {
      return {
        url: parsed.url,
        name: parsed.name,
        source: parsed.source,
        mimeType: parsed.mimeType,
        size: parsed.size,
        updatedAt: parsed.updatedAt ?? new Date().toISOString(),
      };
    }
  } catch {
    if (raw.trim()) {
      return {
        url: raw,
        source: 'file-manager',
        updatedAt: new Date().toISOString(),
      };
    }
  }

  return null;
}

/** toCssUrl 实现对应业务逻辑。 */
function toCssUrl(url: string) {
  return `url(${JSON.stringify(url)})`;
}

/** emitLoginBackgroundChange 实现对应业务逻辑。 */
function emitLoginBackgroundChange(payload: LoginBackgroundPayload | null) {
  window.dispatchEvent(new CustomEvent(LOGIN_BACKGROUND_CHANGE_EVENT, { detail: payload }));
}

/** getStoredLoginBackground 获取对应业务记录。 */
export function getStoredLoginBackground() {
  if (!canUseBrowserStorage()) return null;
  return parseStoredLoginBackground(window.localStorage.getItem(LOGIN_BACKGROUND_STORAGE_KEY));
}

/** applyLoginBackground 执行对应业务流程。 */
export function applyLoginBackground(url?: string | null) {
  if (!canUseBrowserStorage()) return;

  if (url) {
    document.documentElement.style.setProperty('--login-background-image', toCssUrl(url));
    return;
  }

  document.documentElement.style.removeProperty('--login-background-image');
}

/** applyStoredLoginBackground 执行对应业务流程。 */
export function applyStoredLoginBackground() {
  /** stored 保存已存储。 */
  const stored = getStoredLoginBackground();
  applyLoginBackground(stored?.url);
  return stored;
}

/** setStoredLoginBackground 更新并保存对应业务状态。 */
export function setStoredLoginBackground(payload: Omit<LoginBackgroundPayload, 'updatedAt'> & { updatedAt?: string }) {
  if (!canUseBrowserStorage()) return null;
  /** url 保存地址。 */
  const url = payload.url.trim();
  if (!url) throw new Error('登录背景地址为空');

  /** stored、LoginBackgroundPayload 保存已存储、登录请求载荷。 */
  const stored: LoginBackgroundPayload = {
    ...payload,
    url,
    source: payload.source ?? 'file-manager',
    updatedAt: payload.updatedAt ?? new Date().toISOString(),
  };

  window.localStorage.setItem(LOGIN_BACKGROUND_STORAGE_KEY, JSON.stringify(stored));
  applyLoginBackground(stored.url);
  emitLoginBackgroundChange(stored);
  return stored;
}

/** clearStoredLoginBackground 删除或清理对应业务记录。 */
export function clearStoredLoginBackground() {
  if (!canUseBrowserStorage()) return;
  window.localStorage.removeItem(LOGIN_BACKGROUND_STORAGE_KEY);
  applyLoginBackground(null);
  emitLoginBackgroundChange(null);
}
