import TurndownService from 'turndown';
import { gfm } from 'turndown-plugin-gfm';
import type { Article } from '@/src/types/admin';

type MarkdownHeading = {
  level: number;
  text: string;
  anchor: string;
};

export function buildArticleMarkdownDocument(article: Article, safeContent: string) {
  const headings: MarkdownHeading[] = [];
  const anchorCounts = new Map<string, number>();
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
      const level = Number(node.nodeName.slice(1));
      const text = (node.textContent || '').replace(/\s+/g, ' ').trim();
      if (!text) return '';
      const baseAnchor = createMarkdownAnchor(text);
      const count = (anchorCounts.get(baseAnchor) || 0) + 1;
      anchorCounts.set(baseAnchor, count);
      const anchor = count === 1 ? baseAnchor : `${baseAnchor}-${count}`;
      headings.push({ level, text, anchor });
      return `\n\n<a id="${anchor}"></a>\n\n${'#'.repeat(level)} ${content.trim()}\n\n`;
    },
  });

  const content = turndown.turndown(safeContent).trim();
  const sections: string[] = [];
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

function buildMarkdownTableOfContents(headings: MarkdownHeading[]) {
  const baseLevel = Math.min(...headings.map((heading) => heading.level));
  return headings.map((heading) => {
    const indentation = '  '.repeat(Math.max(0, heading.level - baseLevel));
    return `${indentation}- [${escapeMarkdownLinkText(heading.text)}](#${heading.anchor})`;
  }).join('\n');
}

function createMarkdownAnchor(value: string) {
  const anchor = value.normalize('NFKC').toLocaleLowerCase('zh-CN')
    .replace(/[\s_]+/g, '-')
    .replace(/[^\p{L}\p{N}-]+/gu, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
  return anchor || 'section';
}

function escapeMarkdownText(value: string) {
  return value.replace(/([\\`*_{}\[\]()#+\-.!|>])/g, '\\$1');
}

function escapeMarkdownLinkText(value: string) {
  return value.replace(/([\\\[\]])/g, '\\$1');
}

function formatMarkdownDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN');
}
