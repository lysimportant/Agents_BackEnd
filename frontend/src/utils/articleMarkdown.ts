import TurndownService from 'turndown';
import { gfm } from 'turndown-plugin-gfm';
import type { Article } from '@/src/types/admin';

type MarkdownHeading = {
  /** level 表示变量 level。 */
  level: number;
  /** text 表示文本内容。 */
  text: string;
  /** anchor 表示锚点。 */
  anchor: string;
};

/** buildArticleMarkdownDocument 转换并生成对应业务结果。 */
export function buildArticleMarkdownDocument(article: Article, safeContent: string) {
  /** headings、MarkdownHeading 保存变量 headings、变量 MarkdownHeading。 */
  const headings: MarkdownHeading[] = [];
  /** anchorCounts 保存锚点数量。 */
  const anchorCounts = new Map<string, number>();
  /** turndown 保存变量 turndown。 */
  const turndown = new TurndownService({
    bulletListMarker: '-',
    codeBlockStyle: 'fenced',
    emDelimiter: '*',
    headingStyle: 'atx',
    strongDelimiter: '**',
  });
  turndown.use(gfm);
  turndown.addRule('anchoredHeadings', {
    filter: (node) => /^H[1-6]$/.test(node.nodeName),
    replacement: (content, node) => {
      /** level 保存变量 level。 */
      const level = Number(node.nodeName.slice(1));
      /** text 保存文本内容。 */
      const text = (node.textContent || '').replace(/\s+/g, ' ').trim();
      if (!text) return '';
      /** baseAnchor 保存锚点。 */
      const baseAnchor = createMarkdownAnchor(text);
      /** count 保存数量。 */
      const count = (anchorCounts.get(baseAnchor) || 0) + 1;
      anchorCounts.set(baseAnchor, count);
      /** anchor 保存锚点。 */
      const anchor = count === 1 ? baseAnchor : `${baseAnchor}-${count}`;
      headings.push({ level, text, anchor });
      return `\n\n<a id="${anchor}"></a>\n\n${'#'.repeat(level)} ${content.trim()}\n\n`;
    },
  });

  /** content 保存内容。 */
  const content = turndown.turndown(safeContent).trim();
  /** sections、string 保存变量 sections、变量 string。 */
  const sections: string[] = [];
  /** title 保存标题。 */
  const title = article.title.trim();
  if (title) sections.push(`# ${escapeMarkdownText(title)}`);
  if (article.summary.trim()) sections.push(`> ${escapeMarkdownText(article.summary.trim()).replace(/\n/g, '\n> ')}`);
  sections.push([
    `- 作者：${escapeMarkdownText(article.author || '未知')}`,
    `- 分类：${escapeMarkdownText(article.category || '未分类')}`,
    `- 状态：${escapeMarkdownText(article.status || '未知')}`,
    `- 可见范围：${article.isPrivate ? '私密' : '公开'}`,
    `- 创建时间：${escapeMarkdownText(formatMarkdownDate(article.createdAt))}`,
    `- 更新时间：${escapeMarkdownText(formatMarkdownDate(article.updatedAt))}`,
    `- 归属人：${escapeMarkdownText(article.ownerName || '未知')}`,
  ].join('\n'));

  if (headings.length > 0) sections.push(`## 目录\n\n${buildMarkdownTableOfContents(headings)}`);
  if (content) sections.push(content);
  return `${sections.filter(Boolean).join('\n\n')}\n`;
}

/** buildMarkdownTableOfContents 转换并生成对应业务结果。 */
function buildMarkdownTableOfContents(headings: MarkdownHeading[]) {
  /** baseLevel 负责计算或维护变量 baseLevel。 */
  const baseLevel = Math.min(...headings.map((heading) => heading.level));
  return headings.map((heading) => {
    /** indentation 保存变量 indentation。 */
    const indentation = '  '.repeat(Math.max(0, heading.level - baseLevel));
    return `${indentation}- [${escapeMarkdownLinkText(heading.text)}](#${heading.anchor})`;
  }).join('\n');
}

/** createMarkdownAnchor 创建或追加对应业务记录。 */
function createMarkdownAnchor(value: string) {
  /** anchor 保存锚点。 */
  const anchor = value.normalize('NFKC').toLocaleLowerCase('zh-CN')
    .replace(/[\s_]+/g, '-')
    .replace(/[^\p{L}\p{N}-]+/gu, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
  return anchor || 'section';
}

/** escapeMarkdownText 实现对应业务逻辑。 */
function escapeMarkdownText(value: string) {
  return value.replace(/([\\`*_{}\[\]()#+\-.!|>])/g, '\\$1');
}

/** escapeMarkdownLinkText 实现对应业务逻辑。 */
function escapeMarkdownLinkText(value: string) {
  return value.replace(/([\\\[\]])/g, '\\$1');
}

/** formatMarkdownDate 转换并生成对应业务结果。 */
function formatMarkdownDate(value: string) {
  /** date 保存日期。 */
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN');
}
