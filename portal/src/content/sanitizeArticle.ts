import sanitizeHtml from 'sanitize-html';
import { load } from 'cheerio';
import type { PublicTocEntry } from '@/types/publicContent';
import { API_BASE_URL } from '@/config/constants';

/**
 * 允许保留的正文标签白名单，与后端 content.SanitizeArticleContent 保持一致，不额外放宽。
 */
const ALLOWED_TAGS = [
  'a', 'b', 'strong', 'i', 'em', 'u', 's', 'p', 'br', 'hr', 'blockquote',
  'pre', 'code', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li',
  'div', 'span', 'figure', 'figcaption', 'img', 'video', 'source',
  'table', 'thead', 'tbody', 'tr', 'th', 'td',
];

/**
 * 允许保留的属性白名单，仅保留后端已确认的安全属性，避免脚本、事件与任意 class。
 */
const ALLOWED_ATTRIBUTES: Record<string, string[]> = {
  a: ['href'],
  img: ['src', 'alt', 'title', 'width', 'height'],
  video: ['src', 'controls', 'preload'],
  source: ['src'],
};

/** 允许保留的链接协议，与后端 allowedProtocols 对齐，并拒绝协议相对地址。 */
const ALLOWED_SCHEMES = ['http', 'https', 'mailto'];

/** 清洗并结构化后的文章正文结果。 */
export interface SanitizedArticle {
  /** 清洗、重写媒体并注入标题锚点后的正文 HTML。 */
  html: string;
  /** 从正文标题生成的目录条目，与正文锚点一一对应。 */
  tableOfContents: PublicTocEntry[];
}

/**
 * 将标题文本转换为 ASCII 友好的锚点基础，纯中文标题回退为 section。
 * 重复标题通过递增后缀保证唯一，空标题直接跳过。
 */
function buildUniqueAnchor(
  text: string,
  usedAnchors: Set<string>,
): string {
  const slug = text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  const base = slug || 'section';
  let anchor = base;
  let counter = 2;
  while (usedAnchors.has(anchor)) {
    anchor = base + '-' + String(counter);
    counter += 1;
  }
  usedAnchors.add(anchor);
  return anchor;
}

/**
 * 清洗文章正文：按白名单移除危险标签、属性与协议，重写本地媒体为绝对公开地址，
 * 并提取标题生成唯一锚点与目录。任何解析异常都回退为空内容，避免泄漏原始 HTML。
 */
export function sanitizeArticle(rawHtml: string): SanitizedArticle {
  const empty: SanitizedArticle = { html: '', tableOfContents: [] };
  if (!rawHtml) {
    return empty;
  }

  try {
    const cleaned = sanitizeHtml(rawHtml, {
      allowedTags: ALLOWED_TAGS,
      allowedAttributes: ALLOWED_ATTRIBUTES,
      allowedSchemes: ALLOWED_SCHEMES,
      allowProtocolRelative: false,
      transformTags: {
        // 正文图片默认懒加载与异步解码，避免大图阻塞首屏。
        img: (tagName, attribs) => ({
          tagName,
          attribs: {
            ...attribs,
            loading: attribs.loading ?? 'lazy',
            decoding: attribs.decoding ?? 'async',
          },
        }),
        // 视频默认 metadata 预加载，保证不自动播放。
        video: (tagName, attribs) => ({
          tagName,
          attribs: {
            ...attribs,
            preload: attribs.preload ?? 'metadata',
            controls: '',
          },
        }),
      },
    });

    const $ = load(cleaned, null, false);

    // 将后端返回的相对 /api/ 媒体地址重写为浏览器可访问的后端绝对地址。
    $('img, video, source').each((_index, element) => {
      const source = $(element).attr('src');
      if (source && source.startsWith('/api/')) {
        $(element).attr('src', API_BASE_URL + source);
      }
    });

    // 提取标题并注入稳定唯一的锚点 ID，构建目录。
    const tableOfContents: PublicTocEntry[] = [];
    const usedAnchors = new Set<string>();
    $('h1, h2, h3, h4').each((_index, element) => {
      const heading = $(element);
      const text = heading.text().replace(/\s+/g, ' ').trim();
      if (!text) {
        return;
      }
      const level = Number(element.tagName.slice(1));
      const anchor = buildUniqueAnchor(text, usedAnchors);
      heading.attr('id', anchor);
      tableOfContents.push({ id: anchor, level, text });
    });

    const bodyHtml = $('body').html();
    const html = bodyHtml !== null ? bodyHtml : $.html();
    return { html, tableOfContents };
  } catch {
    return empty;
  }
}
