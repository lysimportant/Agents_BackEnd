export const DEFAULT_LOGIN_BACKGROUND_URL = '/images/login-anime-background-beach-v4.webp';
export const LOGIN_BACKGROUND_STORAGE_KEY = 'collector:login-background-url';
export const LOGIN_BACKGROUND_CHANGE_EVENT = 'collector:login-background-change';
export const LOGIN_BACKGROUND_BOOTSTRAP_SCRIPT = `
(function () {
  try {
    var raw = window.localStorage.getItem('${LOGIN_BACKGROUND_STORAGE_KEY}');
    if (!raw) return;

    var url = '';
    try {
      var parsed = JSON.parse(raw);
      if (parsed && typeof parsed.url === 'string') {
        url = parsed.url.trim();
      }
    } catch (error) {
      url = raw.trim();
    }

    if (!url) return;
    document.documentElement.style.setProperty('--login-background-image', 'url(' + JSON.stringify(url) + ')');
  } catch (error) {
    document.documentElement.style.removeProperty('--login-background-image');
  }
})();
`;

export type LoginBackgroundPayload = {
  url: string;
  name?: string;
  source?: 'file-manager' | 'default';
  mimeType?: string;
  size?: number;
  updatedAt: string;
};

function canUseBrowserStorage() {
  return typeof window !== 'undefined' && typeof document !== 'undefined';
}

function parseStoredLoginBackground(raw: string | null): LoginBackgroundPayload | null {
  if (!raw) return null;

  try {
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

function toCssUrl(url: string) {
  return `url(${JSON.stringify(url)})`;
}

function emitLoginBackgroundChange(payload: LoginBackgroundPayload | null) {
  window.dispatchEvent(new CustomEvent(LOGIN_BACKGROUND_CHANGE_EVENT, { detail: payload }));
}

export function getStoredLoginBackground() {
  if (!canUseBrowserStorage()) return null;
  return parseStoredLoginBackground(window.localStorage.getItem(LOGIN_BACKGROUND_STORAGE_KEY));
}

export function applyLoginBackground(url?: string | null) {
  if (!canUseBrowserStorage()) return;

  if (url) {
    document.documentElement.style.setProperty('--login-background-image', toCssUrl(url));
    return;
  }

  document.documentElement.style.removeProperty('--login-background-image');
}

export function applyStoredLoginBackground() {
  const stored = getStoredLoginBackground();
  applyLoginBackground(stored?.url);
  return stored;
}

export function setStoredLoginBackground(payload: Omit<LoginBackgroundPayload, 'updatedAt'> & { updatedAt?: string }) {
  if (!canUseBrowserStorage()) return null;
  const url = payload.url.trim();
  if (!url) throw new Error('登录背景地址为空');

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

export function clearStoredLoginBackground() {
  if (!canUseBrowserStorage()) return;
  window.localStorage.removeItem(LOGIN_BACKGROUND_STORAGE_KEY);
  applyLoginBackground(null);
  emitLoginBackgroundChange(null);
}
