type UnreadMessage = {
  id: number;
  senderId: number;
  recipientId?: number | null;
};

const STORAGE_PREFIX = 'collector:internal-chat-unread-by-peer';

function storageKey(userId: number) {
  return `${STORAGE_PREFIX}:${userId}`;
}

type StoredUnread = Record<string, number[]>;

function readStored(userId: number): StoredUnread {
  if (typeof window === 'undefined') return {};
  try {
    const parsed = JSON.parse(window.localStorage.getItem(storageKey(userId)) || '{}') as unknown;
    if (!parsed || typeof parsed !== 'object') return {};
    return Object.fromEntries(Object.entries(parsed as Record<string, unknown>).map(([key, value]) => [
      key,
      Array.isArray(value) ? value.filter((id): id is number => Number.isInteger(id) && id > 0) : [],
    ]));
  } catch {
    return {};
  }
}

function writeStored(userId: number, unread: StoredUnread) {
  if (typeof window === 'undefined') return;
  try {
    const compact = Object.fromEntries(Object.entries(unread).filter(([, ids]) => ids.length > 0));
    if (Object.keys(compact).length === 0) window.localStorage.removeItem(storageKey(userId));
    else window.localStorage.setItem(storageKey(userId), JSON.stringify(compact));
  } catch {
    return;
  }
}

export function readInternalChatUnreadCounts(userId: number): Record<number, number> {
  return Object.fromEntries(Object.entries(readStored(userId)).map(([peerId, ids]) => [Number(peerId), ids.length]));
}

export function getInternalChatUnreadTotal(userId: number) {
  return Object.values(readStored(userId)).reduce((total, ids) => total + ids.length, 0);
}

export function markInternalChatUnread(message: UnreadMessage, currentUserId: number) {
  const unread = readStored(currentUserId);
  const peerId = message.recipientId == null ? 0 : message.senderId;
  const key = String(peerId);
  unread[key] = Array.from(new Set([...(unread[key] || []), message.id]));
  writeStored(currentUserId, unread);
  return Object.fromEntries(Object.entries(unread).map(([id, ids]) => [Number(id), ids.length]));
}

export function clearInternalChatUnread(peerId: number, currentUserId: number) {
  const unread = readStored(currentUserId);
  delete unread[String(peerId)];
  writeStored(currentUserId, unread);
  return Object.fromEntries(Object.entries(unread).map(([id, ids]) => [Number(id), ids.length]));
}

export function internalChatUnreadStorageKey(userId: number) {
  return storageKey(userId);
}
