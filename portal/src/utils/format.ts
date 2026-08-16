/**
 * 纯展示格式化工具：日期、数字与文件大小，统一使用 Intl 以保证多语言一致性。
 */

/** 将 RFC 3339 时间字符串格式化为本地化的日期时间，非法值返回空字符串。 */
export function formatDateTime(iso: string, locale: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

/** 将 RFC 3339 时间字符串格式化为本地化的日期，非法值返回空字符串。 */
export function formatDate(iso: string, locale: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(date);
}

/** 将字节数格式化为可读的文件大小，按 1024 进位。 */
export function formatFileSize(bytes: number, locale: string): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  const value = bytes / Math.pow(1024, index);
  const fractionDigits = index === 0 ? 0 : 1;
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: fractionDigits,
  }).format(value);
  return formatted + ' ' + units[index];
}

/** 将数字格式化为本地化计数。 */
export function formatCount(count: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(count);
}
