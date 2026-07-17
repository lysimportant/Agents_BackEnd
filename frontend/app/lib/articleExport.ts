import type { Article } from '../types/admin';
import { API_BASE_URL } from './constants';
import { buildArticleMarkdownDocument } from './articleMarkdown';

export type ArticleExportFormat = 'csv' | 'pdf' | 'print' | 'png' | 'word' | 'html' | 'markdown';

export const articleExportOptions: Array<{ key: ArticleExportFormat; label: string }> = [
  { key: 'pdf', label: 'PDF 文件 (.pdf)' },
  { key: 'print', label: '打印文章' },
  { key: 'word', label: 'Word 文档 (.doc)' },
  { key: 'html', label: 'HTML 网页 (.html)' },
  { key: 'markdown', label: 'Markdown 文档 (.md)' },
  { key: 'png', label: 'PNG 图片（自动分页）' },
  { key: 'csv', label: 'Excel 表格 (.csv)' },
];

const EXPORT_WIDTH = 1120;
const PNG_PAGE_HEIGHT = 1580;
const PNG_RENDER_SCALE = 1.5;

export async function exportArticle(article: Article, format: ArticleExportFormat) {
  assertExportableArticle(article);

  switch (format) {
    case 'csv':
      exportCsv(article);
      return `《${article.title}》CSV 已下载，正文与图片链接可使用 Excel 查看。`;
    case 'pdf': {
      const pages = await exportPdf(article);
      return `《${article.title}》PDF 已下载，共 ${pages} 页。`;
    }
    case 'print':
      await openPrintView(article);
      return `《${article.title}》打印窗口已打开。`;
    case 'png': {
      const pages = await exportPngPages(article);
      return pages > 1
        ? `《${article.title}》完整内容已导出为 ${pages} 张连续 PNG。`
        : `《${article.title}》完整内容 PNG 已下载。`;
    }
    case 'html':
      downloadBlob(buildArticleHtml(article), 'text/html;charset=utf-8', buildFilename(article, 'html'));
      return `《${article.title}》HTML 网页已下载。`;
    case 'word':
      downloadBlob(`\uFEFF${buildWordDocument(article)}`, 'application/msword;charset=utf-8', buildFilename(article, 'doc'));
      return `《${article.title}》Word 文档已下载。`;
    case 'markdown':
      downloadBlob(`\uFEFF${buildArticleMarkdownDocument(article, sanitizeArticleContent(article.content || ''))}`, 'text/markdown;charset=utf-8', buildFilename(article, 'md'));
      return `《${article.title}》Markdown 文档已下载。`;
    default:
      throw new Error('不支持所选的文章导出格式。');
  }
}

function assertExportableArticle(article: Article) {
  const contentText = htmlToText(article?.content || '');
  if (!article || (!article.title?.trim() && !article.summary?.trim() && !contentText)) {
    throw new Error('当前文章没有可导出的标题、摘要或正文内容。');
  }
}

function exportCsv(article: Article) {
  const safeContent = sanitizeArticleContent(article.content || '');
  const images = extractImageSources(safeContent);
  const header = ['ID', '标题', '摘要', '作者', '分类', '正文', '图片链接', '状态', '可见范围', '归属人', '浏览量', '创建时间', '更新时间'];
  const row = [
    article.id,
    article.title,
    article.summary,
    article.author,
    article.category,
    htmlToText(safeContent) || '暂无正文内容',
    images.join('\n'),
    article.status,
    article.isPrivate ? '私密' : '公开',
    article.ownerName || '',
    article.views,
    formatDate(article.createdAt),
    formatDate(article.updatedAt),
  ];
  const csv = `\uFEFF${[header, row].map((values) => values.map(escapeCsvCell).join(',')).join('\r\n')}`;
  downloadBlob(csv, 'text/csv;charset=utf-8', buildFilename(article, 'csv'));
}

async function openPrintView(article: Article) {
  const printWindow = window.open('', '_blank', 'width=1100,height=820');
  if (!printWindow) {
    throw new Error('浏览器阻止了打印窗口，请允许本站弹出窗口后重新导出。');
  }

  try {
    printWindow.opener = null;
    printWindow.document.open();
    printWindow.document.write(buildArticleHtml(article, true));
    printWindow.document.close();

    const failedImages = await waitForDocumentImages(printWindow.document);
    if (failedImages > 0) {
      throw new Error(`《${article.title}》有 ${failedImages} 张正文图片加载失败，请检查图片地址或登录状态后重试。`);
    }
    if (printWindow.document.fonts) {
      await Promise.race([printWindow.document.fonts.ready, delay(2000)]);
    }
    if (printWindow.closed) {
      throw new Error('打印窗口已关闭，文章尚未导出。');
    }
    printWindow.focus();
    printWindow.print();
  } catch (error) {
    if (!printWindow.closed) printWindow.close();
    throw error;
  }
}

async function renderArticlePages(article: Article) {
  const exportDocument = new DOMParser().parseFromString(buildArticleHtml(article), 'text/html');
  const articlePage = exportDocument.querySelector<HTMLElement>('.article-export-page');
  if (!articlePage) throw new Error('文章图片版式生成失败，请重试。');

  const stage = document.createElement('div');
  stage.setAttribute('aria-hidden', 'true');
  // Render behind the application so html2canvas can still paint the content.
  // A hidden or transparent ancestor would produce an empty PDF/PNG.
  stage.style.cssText = `position:fixed;left:0;top:0;width:${EXPORT_WIDTH}px;background:#fff;pointer-events:none;z-index:-2147483648;`;
  stage.innerHTML = `<style>${ARTICLE_EXPORT_STYLES}</style>${articlePage.outerHTML}`;
  document.body.appendChild(stage);

  try {
    const renderedPage = stage.querySelector<HTMLElement>('.article-export-page');
    if (!renderedPage) throw new Error('文章图片版式生成失败，请重试。');
    const failedImages = await inlineImages(renderedPage);
    if (failedImages > 0) {
      throw new Error(`《${article.title}》有 ${failedImages} 张图片受跨域或权限限制，无法生成完整 PDF / PNG；请检查图片后重试，或改用 Word / HTML 导出。`);
    }
    if (document.fonts) await Promise.race([document.fonts.ready, delay(2000)]);
    await nextPaint();

    const contentHeight = Math.max(1, Math.ceil(renderedPage.scrollHeight));
    const pages = Math.max(1, Math.ceil(contentHeight / PNG_PAGE_HEIGHT));
    const { default: html2canvas } = await import('html2canvas');
    const fullCanvas = await html2canvas(renderedPage, {
      backgroundColor: '#ffffff',
      logging: false,
      scale: PNG_RENDER_SCALE,
      useCORS: false,
      width: EXPORT_WIDTH,
      height: contentHeight,
      windowWidth: EXPORT_WIDTH,
      windowHeight: contentHeight,
    });
    assertCanvasHasVisibleContent(fullCanvas);

    return splitCanvasIntoPages(fullCanvas, contentHeight, pages);
  } finally {
    stage.remove();
  }
}

async function exportPngPages(article: Article) {
  const pages = await renderArticlePages(article);
  for (let pageIndex = 0; pageIndex < pages.length; pageIndex += 1) {
    const blob = await canvasToBlob(pages[pageIndex]);
    const suffix = pages.length > 1 ? `page-${String(pageIndex + 1).padStart(2, '0')}.png` : 'png';
    downloadBlob(blob, 'image/png', buildFilename(article, suffix));
    await delay(30);
  }
  return pages.length;
}

async function exportPdf(article: Article) {
  const [{ jsPDF }, pages] = await Promise.all([
    import('jspdf'),
    renderArticlePages(article),
  ]);
  const pdf = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4', compress: true });
  const pageWidth = 210;
  const pageHeight = 297;
  const margin = 8;
  const availableWidth = pageWidth - margin * 2;
  const availableHeight = pageHeight - margin * 2;

  pages.forEach((canvas, index) => {
    if (index > 0) pdf.addPage('a4', 'portrait');
    const scale = Math.min(availableWidth / canvas.width, availableHeight / canvas.height);
    const width = canvas.width * scale;
    const height = canvas.height * scale;
    const x = (pageWidth - width) / 2;
    pdf.addImage(canvas, 'PNG', x, margin, width, height, undefined, 'FAST');
  });
  pdf.save(buildFilename(article, 'pdf'));
  return pages.length;
}

function splitCanvasIntoPages(fullCanvas: HTMLCanvasElement, contentHeight: number, pages: number) {
  const renderedPages: HTMLCanvasElement[] = [];
  for (let pageIndex = 0; pageIndex < pages; pageIndex += 1) {
    const offset = pageIndex * PNG_PAGE_HEIGHT;
    const visibleHeight = Math.min(PNG_PAGE_HEIGHT, contentHeight - offset);
    const canvas = document.createElement('canvas');
    canvas.width = fullCanvas.width;
    canvas.height = Math.ceil(visibleHeight * PNG_RENDER_SCALE);
    const context = canvas.getContext('2d');
    if (!context) throw new Error('当前浏览器无法创建文章导出画布。');
    context.fillStyle = '#ffffff';
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.drawImage(
      fullCanvas,
      0,
      Math.floor(offset * PNG_RENDER_SCALE),
      fullCanvas.width,
      canvas.height,
      0,
      0,
      canvas.width,
      canvas.height,
    );
    renderedPages.push(canvas);
  }
  return renderedPages;
}

function assertCanvasHasVisibleContent(canvas: HTMLCanvasElement) {
  const context = canvas.getContext('2d', { willReadFrequently: true });
  if (!context) throw new Error('当前浏览器无法检查文章导出内容。');
  const step = Math.max(1, Math.floor(Math.min(canvas.width, canvas.height) / 180));
  const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data;
  let visibleSamples = 0;
  let totalSamples = 0;
  for (let y = 0; y < canvas.height; y += step) {
    for (let x = 0; x < canvas.width; x += step) {
      const offset = (y * canvas.width + x) * 4;
      totalSamples += 1;
      if (pixels[offset + 3] > 16 && (pixels[offset] < 248 || pixels[offset + 1] < 248 || pixels[offset + 2] < 248)) {
        visibleSamples += 1;
      }
    }
  }
  if (visibleSamples < Math.max(12, Math.floor(totalSamples * 0.001))) {
    throw new Error('PDF / PNG 渲染结果为空白，请刷新页面后重试。');
  }
}

function buildArticleHtml(article: Article, print = false) {
  const safeContent = sanitizeArticleContent(article.content || '');
  const bodyText = htmlToText(safeContent);
  const images = extractImageSources(safeContent);
  const isIndexable = article.status === '已发布' && !article.isPrivate;
  const structuredData = isIndexable ? {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: article.summary || bodyText.slice(0, 180),
    articleSection: article.category,
    articleBody: bodyText,
    author: { '@type': 'Person', name: article.author },
    datePublished: article.createdAt,
    dateModified: article.updatedAt,
    image: images.length > 0 ? images : undefined,
    isAccessibleForFree: true,
  } : null;
  const jsonLd = structuredData ? JSON.stringify(structuredData).replace(/</g, '\\u003c') : '';
  const title = (article.title || '').trim() || `文章 ${article.id}`;
  const summary = (article.summary || '').trim() || '暂无摘要';
  const content = safeContent.trim() || '<p class="article-empty">暂无正文内容。</p>';
  const articleScope = isIndexable ? ' itemscope itemtype="https://schema.org/Article"' : '';
  const printHint = print ? '<p class="print-hint no-print">正文和图片加载完成后将打开打印对话框，请选择打印机及打印设置。</p>' : '';

  return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escapeHtml(title)} - HuaJian_AI</title><meta name="description" content="${escapeHtml(article.summary || bodyText.slice(0, 160) || title)}">${isIndexable ? '' : '<meta name="robots" content="noindex,nofollow">'}${jsonLd ? `<script type="application/ld+json">${jsonLd}</script>` : ''}<style>${ARTICLE_EXPORT_STYLES}</style></head><body><main class="article-export-page"><article${articleScope}>
    <header class="article-export-header"><p class="brand">HuaJian_AI · 文章内容</p><h1${isIndexable ? ' itemprop="headline"' : ''}>${escapeHtml(title)}</h1><p class="summary"${isIndexable ? ' itemprop="description"' : ''}>${escapeHtml(summary)}</p></header>
    <dl class="article-export-meta"><div><dt>作者</dt><dd${isIndexable ? ' itemprop="author"' : ''}>${escapeHtml(article.author || '未知')}</dd></div><div><dt>分类</dt><dd${isIndexable ? ' itemprop="articleSection"' : ''}>${escapeHtml(article.category || '未分类')}</dd></div><div><dt>状态</dt><dd>${escapeHtml(article.status || '未知')}</dd></div><div><dt>可见范围</dt><dd>${article.isPrivate ? '私密' : '公开'}</dd></div><div><dt>创建时间</dt><dd><time${isIndexable ? ' itemprop="datePublished"' : ''} datetime="${escapeHtml(article.createdAt)}">${escapeHtml(formatDate(article.createdAt))}</time></dd></div><div><dt>更新时间</dt><dd><time${isIndexable ? ' itemprop="dateModified"' : ''} datetime="${escapeHtml(article.updatedAt)}">${escapeHtml(formatDate(article.updatedAt))}</time></dd></div><div><dt>归属人</dt><dd>${escapeHtml(article.ownerName || '未知')}</dd></div><div><dt>浏览量</dt><dd>${escapeHtml(article.views)}</dd></div></dl>
    ${printHint}<section class="article-export-content"${isIndexable ? ' itemprop="articleBody"' : ''}>${content}</section>
    <footer class="article-export-footer">导出时间：${escapeHtml(new Date().toLocaleString('zh-CN'))}</footer>
  </article></main></body></html>`;
}

function buildWordDocument(article: Article) {
  return buildArticleHtml(article)
    .replace(
      '<html lang="zh-CN">',
      '<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:w="urn:schemas-microsoft-com:office:word" lang="zh-CN">',
    )
    .replace(
      '<head>',
      '<head><meta name="ProgId" content="Word.Document"><meta name="Generator" content="HuaJian_AI"><xml><w:WordDocument><w:View>Print</w:View><w:Zoom>100</w:Zoom></w:WordDocument></xml>',
    );
}


function sanitizeArticleContent(input: string) {
  if (!input) return '';
  const parsed = new DOMParser().parseFromString(input, 'text/html');
  const allowedTags = new Set(['P', 'BR', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'STRONG', 'B', 'EM', 'I', 'U', 'S', 'DEL', 'BLOCKQUOTE', 'UL', 'OL', 'LI', 'A', 'FIGURE', 'FIGCAPTION', 'IMG', 'VIDEO', 'SOURCE', 'DIV', 'SPAN', 'PRE', 'CODE', 'HR', 'TABLE', 'THEAD', 'TBODY', 'TFOOT', 'TR', 'TH', 'TD', 'TIME']);
  const blockedTags = new Set(['SCRIPT', 'STYLE', 'IFRAME', 'OBJECT', 'EMBED', 'FORM', 'INPUT', 'BUTTON', 'TEXTAREA', 'SELECT', 'OPTION', 'SVG', 'MATH', 'LINK', 'META']);

  Array.from(parsed.body.querySelectorAll('*')).forEach((node) => {
    if (blockedTags.has(node.tagName)) {
      node.remove();
      return;
    }
    if (!allowedTags.has(node.tagName)) {
      node.replaceWith(...Array.from(node.childNodes));
      return;
    }

    const attributes = new Map(Array.from(node.attributes).map((attribute) => [attribute.name.toLowerCase(), attribute.value]));
    Array.from(node.attributes).forEach((attribute) => node.removeAttribute(attribute.name));

    if (node.tagName === 'A') {
      const href = normalizeLink(attributes.get('href') || '');
      if (href) node.setAttribute('href', href);
      node.setAttribute('target', '_blank');
      node.setAttribute('rel', 'noopener noreferrer');
    }
    if (node.tagName === 'IMG') {
      const source = normalizeMediaSource(attributes.get('src') || '', 'image');
      const alt = (attributes.get('alt') || attributes.get('title') || '文章内容图片').trim();
      if (!source) {
        node.replaceWith(parsed.createTextNode(`[图片：${alt}]`));
        return;
      }
      node.setAttribute('src', source);
      node.setAttribute('alt', alt);
      node.setAttribute('title', (attributes.get('title') || alt).trim());
      node.setAttribute('loading', 'eager');
      node.setAttribute('decoding', 'async');
      copyNumericAttribute(node, attributes, 'width');
      copyNumericAttribute(node, attributes, 'height');
    }
    if (node.tagName === 'VIDEO') {
      const source = normalizeMediaSource(attributes.get('src') || '', 'video');
      if (source) node.setAttribute('src', source);
      const poster = normalizeMediaSource(attributes.get('poster') || '', 'image');
      if (poster) node.setAttribute('poster', poster);
      node.setAttribute('controls', '');
      node.setAttribute('preload', 'metadata');
    }
    if (node.tagName === 'SOURCE') {
      const source = normalizeMediaSource(attributes.get('src') || '', 'video');
      if (!source) {
        node.remove();
        return;
      }
      node.setAttribute('src', source);
      const type = attributes.get('type');
      if (type && /^video\/[a-z0-9.+-]+$/i.test(type)) node.setAttribute('type', type);
    }
    if (node.tagName === 'FIGURE' || node.tagName === 'DIV') {
      const safeClasses = (attributes.get('class') || '').split(/\s+/).filter((name) => ['article-media', 'image-media', 'video-media'].includes(name));
      if (safeClasses.length > 0) node.setAttribute('class', safeClasses.join(' '));
    }
    if (node.tagName === 'TH' || node.tagName === 'TD') {
      copyNumericAttribute(node, attributes, 'colspan');
      copyNumericAttribute(node, attributes, 'rowspan');
    }
    if (node.tagName === 'OL') copyNumericAttribute(node, attributes, 'start');
    if (node.tagName === 'TIME' && attributes.get('datetime')) node.setAttribute('datetime', attributes.get('datetime')!.slice(0, 64));
    if (attributes.get('title') && node.tagName !== 'IMG') node.setAttribute('title', attributes.get('title')!.slice(0, 300));
  });
  return parsed.body.innerHTML;
}

function copyNumericAttribute(node: Element, attributes: Map<string, string>, name: string) {
  const value = attributes.get(name);
  if (value && /^\d{1,5}$/.test(value)) node.setAttribute(name, value);
}

function normalizeLink(value: string) {
  const source = value.trim();
  if (/^(mailto:|#)/i.test(source)) return source;
  try {
    const url = new URL(source, window.location.origin);
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : '';
  } catch {
    return '';
  }
}

function normalizeMediaSource(value: string, kind: 'image' | 'video') {
  const source = value.trim();
  if (!source) return '';
  if (kind === 'image' && /^data:image\/(?:png|jpe?g|gif|webp|svg\+xml);base64,/i.test(source)) return source;
  if (source.startsWith('/api/files/')) return `${API_BASE_URL.replace(/\/$/, '')}${source}`;
  try {
    const url = new URL(source, window.location.origin);
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : '';
  } catch {
    return '';
  }
}

function extractImageSources(html: string) {
  if (!html) return [];
  const parsed = new DOMParser().parseFromString(html, 'text/html');
  return Array.from(new Set(Array.from(parsed.images).map((image) => image.getAttribute('src') || '').filter(Boolean)));
}

async function inlineImages(root: HTMLElement) {
  const images = Array.from(root.querySelectorAll('img'));
  const results = await Promise.all(images.map(async (image) => {
    const source = image.getAttribute('src') || '';
    if (!source) return false;
    if (source.startsWith('data:image/')) return waitForImage(image, 8000);
    try {
      const url = new URL(source, window.location.href);
      const apiOrigin = new URL(API_BASE_URL, window.location.href).origin;
      const credentials: RequestCredentials = url.origin === window.location.origin || url.origin === apiOrigin ? 'include' : 'omit';
      const response = await fetch(url.href, { credentials, cache: 'force-cache' });
      if (!response.ok) return false;
      const blob = await response.blob();
      if (!blob.type.startsWith('image/')) return false;
      image.removeAttribute('srcset');
      image.src = await blobToDataUrl(blob);
      return waitForImage(image, 8000);
    } catch {
      return false;
    }
  }));
  return results.filter((loaded) => !loaded).length;
}

async function waitForDocumentImages(targetDocument: Document) {
  const results = await Promise.all(Array.from(targetDocument.images).map((image) => waitForImage(image, 10000)));
  return results.filter((loaded) => !loaded).length;
}

function waitForImage(image: HTMLImageElement, timeoutMs: number) {
  return new Promise<boolean>((resolve) => {
    if (image.complete) {
      resolve(image.naturalWidth > 0);
      return;
    }
    let settled = false;
    const finish = (loaded: boolean) => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timeout);
      image.removeEventListener('load', onLoad);
      image.removeEventListener('error', onError);
      resolve(loaded);
    };
    const onLoad = () => finish(image.naturalWidth > 0);
    const onError = () => finish(false);
    const timeout = window.setTimeout(() => finish(false), timeoutMs);
    image.addEventListener('load', onLoad, { once: true });
    image.addEventListener('error', onError, { once: true });
  });
}

function htmlToText(html: string) {
  if (!html) return '';
  const parsed = new DOMParser().parseFromString(html, 'text/html');
  return (parsed.body.textContent || '').replace(/\s+/g, ' ').trim();
}

function escapeCsvCell(value: string | number) {
  const raw = String(value ?? '');
  const trimmed = raw.trimStart();
  const text = trimmed && '=+-@'.includes(trimmed[0]) ? `'${raw}` : raw;
  return /[",\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
}

function escapeHtml(value: string | number) {
  return String(value ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN');
}

function buildFilename(article: Article, suffix: string) {
  const title = (article.title || `article-${article.id}`).replace(/[<>:"/\\|?*\u0000-\u001F]/g, '-').replace(/\s+/g, '-').replace(/-+/g, '-').slice(0, 72) || `article-${article.id}`;
  const stamp = new Date().toISOString().slice(0, 10);
  return `HuaJian_AI-${title}-${stamp}.${suffix}`;
}

function downloadBlob(content: BlobPart | Blob, type: string, filename: string) {
  const blob = content instanceof Blob ? content : new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';
  document.body.appendChild(link);
  try {
    link.click();
  } finally {
    link.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
}

function blobToDataUrl(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => typeof reader.result === 'string' ? resolve(reader.result) : reject(new Error('图片读取失败。'));
    reader.onerror = () => reject(reader.error || new Error('图片读取失败。'));
    reader.readAsDataURL(blob);
  });
}

function canvasToBlob(canvas: HTMLCanvasElement) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error('图片生成失败，请重试。')), 'image/png');
  });
}

function nextPaint() {
  return new Promise<void>((resolve) => window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve())));
}

function delay(milliseconds: number) {
  return new Promise<void>((resolve) => window.setTimeout(resolve, milliseconds));
}

const ARTICLE_EXPORT_STYLES = `
  *{box-sizing:border-box}html,body{margin:0;padding:0;background:#fff;color:#172033;font-family:"Microsoft YaHei","PingFang SC",Arial,sans-serif}.article-export-page{width:${EXPORT_WIDTH}px;min-height:100%;margin:0 auto;padding:64px 72px;background:#fff}.article-export-header{padding:0 0 28px;border-bottom:3px solid #1761c7}.brand{margin:0 0 14px;color:#1761c7;font-size:15px;font-weight:700}.article-export-header h1{margin:0;color:#102a5e;font-size:42px;line-height:1.28;word-break:break-word}.summary{margin:18px 0 0;color:#52647f;font-size:18px;line-height:1.75}.article-export-meta{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0;margin:28px 0 34px;border:1px solid #dbe4f0;border-radius:8px;overflow:hidden}.article-export-meta>div{display:grid;grid-template-columns:108px 1fr;min-height:54px;border-bottom:1px solid #e7edf5}.article-export-meta>div:nth-child(odd){border-right:1px solid #e7edf5}.article-export-meta>div:nth-last-child(-n+2){border-bottom:0}.article-export-meta dt,.article-export-meta dd{margin:0;padding:15px 16px}.article-export-meta dt{color:#52647f;background:#f5f8fc;font-weight:700}.article-export-meta dd{word-break:break-word}.print-hint{margin:0 0 22px;padding:12px 16px;color:#0b4da2;border:1px solid #bfdbfe;border-radius:6px;background:#eff6ff}.article-export-content{font-size:17px;line-height:1.9;word-break:break-word}.article-export-content h1,.article-export-content h2,.article-export-content h3,.article-export-content h4{margin:1.4em 0 .65em;color:#172b4d;line-height:1.4}.article-export-content p{margin:.8em 0}.article-export-content ul,.article-export-content ol{padding-left:30px}.article-export-content blockquote{margin:1.3em 0;padding:12px 20px;color:#52647f;border-left:4px solid #1761c7;background:#f5f8fc}.article-export-content a{color:#0958d9;text-decoration:underline}.article-export-content pre{overflow-wrap:anywhere;padding:16px;border-radius:6px;background:#f1f5f9;white-space:pre-wrap}.article-export-content table{width:100%;border-collapse:collapse}.article-export-content th,.article-export-content td{padding:10px;border:1px solid #cbd5e1;text-align:left}.article-export-content figure{break-inside:avoid;margin:28px 0;padding:12px;border:1px solid #dbe4f0;border-radius:8px;background:#f8fafc}.article-export-content img,.article-export-content video{display:block;width:auto;max-width:100%;height:auto;max-height:900px;margin:0 auto;object-fit:contain}.article-export-content figcaption{padding:10px 4px 0;color:#64748b;font-size:14px;text-align:center}.article-empty{padding:36px;color:#64748b;border:1px dashed #cbd5e1;text-align:center}.article-export-footer{margin-top:48px;padding-top:16px;color:#94a3b8;border-top:1px solid #e2e8f0;font-size:13px;text-align:right}@page{size:A4;margin:14mm}@media print{html,body{background:#fff}.article-export-page{width:auto;padding:0}.no-print{display:none}.article-export-header h1{font-size:30px}.article-export-meta{break-inside:avoid}.article-export-content figure{break-inside:avoid}.article-export-footer{position:static}}
`;
