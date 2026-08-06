/** 未读计数存储识别一条新消息所需的最小字段集合。 */
export type UnreadMessage = {
  /** id 表示标识。 */
  id: number;
  /** senderId 表示标识。 */
  senderId: number;
  /** recipientId 表示标识。 */
  recipientId?: number | null;
};

/** STORAGE_PREFIX 保存模块使用的固定配置或共享状态。 */
const STORAGE_PREFIX = 'collector:internal-chat-unread-by-peer';

/** storageKey 实现对应业务逻辑。 */
function storageKey(userId: number) {
  return `${STORAGE_PREFIX}:${userId}`;
}

type StoredUnread = Record<string, number[]>;

/** readStored 加载对应业务数据。 */
function readStored(userId: number): StoredUnread {
  if (typeof window === 'undefined') return {};
  try {
    /** parsed 保存解析结果。 */
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

/** writeStored 实现对应业务逻辑。 */
function writeStored(userId: number, unread: StoredUnread) {
  if (typeof window === 'undefined') return;
  try {
    /** compact 负责计算或维护变量 compact。 */
    const compact = Object.fromEntries(Object.entries(unread).filter(([, ids]) => ids.length > 0));
    if (Object.keys(compact).length === 0) window.localStorage.removeItem(storageKey(userId));
    else window.localStorage.setItem(storageKey(userId), JSON.stringify(compact));
  } catch {
    return;
  }
}

/** 读取指定登录用户按聊天对象持久化的未读数量。 */
export function readInternalChatUnreadCounts(userId: number): Record<number, number> {
  return Object.fromEntries(Object.entries(readStored(userId)).map(([peerId, ids]) => [Number(peerId), ids.length]));
}

/** 返回当前用户的内部聊天未读总数。 */
export function getInternalChatUnreadTotal(userId: number) {
  return Object.values(readStored(userId)).reduce((total, ids) => total + ids.length, 0);
}

/** 将一条收到的消息加入接收者对应聊天对象的未读计数。 */
export function markInternalChatUnread(message: UnreadMessage, currentUserId: number) {
  /** unread 保存变量 unread。 */
  const unread = readStored(currentUserId);
  /** peerId 保存标识。 */
  const peerId = message.recipientId == null ? 0 : message.senderId;
  /** key 保存存储键。 */
  const key = String(peerId);
  unread[key] = Array.from(new Set([...(unread[key] || []), message.id]));
  writeStored(currentUserId, unread);
  return Object.fromEntries(Object.entries(unread).map(([id, ids]) => [Number(id), ids.length]));
}

/** 用户查看会话后清除对应聊天对象的未读消息。 */
export function clearInternalChatUnread(peerId: number, currentUserId: number) {
  /** unread 保存变量 unread。 */
  const unread = readStored(currentUserId);
  delete unread[String(peerId)];
  writeStored(currentUserId, unread);
  return Object.fromEntries(Object.entries(unread).map(([id, ids]) => [Number(id), ids.length]));
}

/** 返回指定用户用于跨标签页同步未读事件的存储键。 */
export function internalChatUnreadStorageKey(userId: number) {
  return storageKey(userId);
}
