'use client';

import { useEffect, useRef, useState, type ChangeEvent, type FormEvent, type MouseEvent as ReactMouseEvent, type ReactNode } from 'react';
import {
  AppstoreOutlined,
  BoldOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FileTextOutlined,
  PictureOutlined,
  PlayCircleOutlined,
  ItalicOutlined,
  LinkOutlined,
  MinusOutlined,
  OrderedListOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  SendOutlined,
  StrikethroughOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { Button, Card, Empty, Input, Modal, Popconfirm, Select, Space, Statistic, Switch, Tag, Tooltip } from 'antd';
import type { Article, ArticleForm } from '../types/admin';
import { API_BASE_URL, articleStatusOptions, MAX_UPLOAD_SIZE } from '../lib/constants';
import { requestWithSession } from '../lib/api';

type ArticlesPageProps = {
  filteredArticles: Article[];
  articleForm: ArticleForm;
  editingArticleId: number | null;
  articleKeyword: string;
  articleStatus: string;
  isSavingArticle: boolean;
  onArticleFormChange: (form: ArticleForm) => void;
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => void;
  onResetArticleForm: () => void;
  onArticleKeywordChange: (keyword: string) => void;
  onArticleStatusChange: (status: string) => void;
  onResetFilters: () => void;
  onEditArticle: (article: Article) => void;
  onToggleArticleStatus: (article: Article) => void;
  onDeleteArticle: (articleId: number) => void;
};

export function ArticlesPage(props: ArticlesPageProps) {
  const {
    filteredArticles, articleForm, editingArticleId, articleKeyword, articleStatus, isSavingArticle,
    onArticleFormChange, onSubmitArticle, onResetArticleForm, onArticleKeywordChange,
    onArticleStatusChange, onResetFilters, onEditArticle, onToggleArticleStatus, onDeleteArticle,
  } = props;
  const [previewArticle, setPreviewArticle] = useState<Article | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editorArticleId, setEditorArticleId] = useState<number | null>(null);
  const isEditing = editorArticleId !== null;
  const published = filteredArticles.filter((article) => article.status === '已发布').length;

  const openNew = () => { onResetArticleForm(); setEditorArticleId(null); setIsEditorOpen(true); };
  const openEdit = (article: Article) => { setEditorArticleId(article.id); setIsEditorOpen(true); onEditArticle(article); };
  const closeEditor = () => { onResetArticleForm(); setEditorArticleId(null); setIsEditorOpen(false); };
  const submit = (event: FormEvent<HTMLFormElement>) => { onSubmitArticle(event); };

  return <div className="page-stack article-workspace">
    <Card className="article-hero" bordered={false}>
      <div><p className="page-kicker">内容管理 / 写作中心</p><h1>文章管理</h1><span>在富文本编辑器中创作、预览、保存和发布文章；所有操作均直连后端接口。</span></div>
      <Space><Button type="primary" icon={<PlusOutlined />} onClick={openNew}>写文章</Button><Button icon={<AppstoreOutlined />} onClick={onResetFilters}>重置筛选</Button></Space>
    </Card>

    <div className="article-stat-grid"><Card><Statistic title="当前结果" value={filteredArticles.length} prefix={<FileTextOutlined />} /></Card><Card><Statistic title="已发布" value={published} suffix="篇" prefix={<SendOutlined />} /></Card><Card><Statistic title="草稿 / 下架" value={filteredArticles.length - published} suffix="篇" prefix={<EditOutlined />} /></Card></div>

    <Card className="article-browser-card" title="文章库" extra={<Space className="article-filter-bar"><Input allowClear value={articleKeyword} onChange={(event) => onArticleKeywordChange(event.target.value)} placeholder="标题、分类、作者、摘要" prefix={<FileTextOutlined />} /><Select value={articleStatus} onChange={onArticleStatusChange} options={[{ value: '全部', label: '全部状态' }, ...articleStatusOptions.map((status) => ({ value: status, label: status }))]} /><Button onClick={onResetFilters}>重置</Button></Space>}>
      {filteredArticles.length === 0 ? <Empty description="暂无匹配文章"><Button type="primary" onClick={openNew}>创建第一篇文章</Button></Empty> : <div className="article-card-list">{filteredArticles.map((article) => <ArticleCard key={article.id} article={article} onPreview={setPreviewArticle} onEdit={openEdit} onToggle={onToggleArticleStatus} onDelete={onDeleteArticle} />)}</div>}
    </Card>

    <Modal open={isEditorOpen} title={isEditing ? '编辑文章' : '新建文章'} footer={null} width="min(1160px, 96vw)" destroyOnClose onCancel={closeEditor}>
      <form className="rich-editor-form" onSubmit={submit}>
        <div className="rich-editor-meta">
          <label>标题<Input required size="large" value={articleForm.title} onChange={(event) => onArticleFormChange({ ...articleForm, title: event.target.value })} placeholder="请输入清晰、有辨识度的文章标题" /></label>
          <label>分类<Input required value={articleForm.category} onChange={(event) => onArticleFormChange({ ...articleForm, category: event.target.value })} placeholder="例如：通知公告" /></label>
          <label>作者<Input required value={articleForm.author} onChange={(event) => onArticleFormChange({ ...articleForm, author: event.target.value })} placeholder="作者姓名" /></label>
          <label>状态<Select value={articleForm.status} options={articleStatusOptions.map((status) => ({ value: status, label: status }))} onChange={(status) => onArticleFormChange({ ...articleForm, status: status as ArticleForm['status'] })} /></label>
        </div>
        <label className="article-summary-field">摘要<Input.TextArea value={articleForm.summary} rows={2} onChange={(event) => onArticleFormChange({ ...articleForm, summary: event.target.value })} placeholder="一句话概括文章价值，便于列表展示" /></label>
        <div className="privacy-switch-row">
          <div>
            <strong>仅自己可见</strong>
            <small>开启后仅归属人和系统管理员可查看；其他登录用户不会在列表中看到此文章。</small>
          </div>
          <Switch checked={Boolean(articleForm.isPrivate)} onChange={(checked) => onArticleFormChange({ ...articleForm, isPrivate: checked })} checkedChildren="私密" unCheckedChildren="公开" />
        </div>
        <RichTextEditor value={articleForm.content} onChange={(content) => onArticleFormChange({ ...articleForm, content })} />
        <div className="rich-editor-actions"><Button onClick={closeEditor}>取消</Button><Button htmlType="submit" type="primary" loading={isSavingArticle} icon={<SaveOutlined />}>{isEditing ? '保存修改' : '保存文章'}</Button></div>
      </form>
    </Modal>

    <ArticlePreview article={previewArticle} onClose={() => setPreviewArticle(null)} />
  </div>;
}

type ArticleCardProps = { article: Article; onPreview: (article: Article) => void; onEdit: (article: Article) => void; onToggle: (article: Article) => void; onDelete: (articleId: number) => void };
function ArticleCard({ article, onPreview, onEdit, onToggle, onDelete }: ArticleCardProps) {
  const isPublished = article.status === '已发布';
  return <article className="article-library-card">
    <div className="article-library-main"><div className="article-library-title"><h3>{article.title}</h3><Space size={6} wrap><Tag color={isPublished ? 'success' : article.status === '下架' ? 'default' : 'processing'}>{article.status}</Tag><Tag color={article.isPrivate ? 'warning' : 'blue'}>{article.isPrivate ? '私密' : '公开'}</Tag></Space></div><p>{article.summary || '暂无摘要，打开文章后可补充内容概览。'}</p><div className="article-library-meta"><span>{article.category}</span><span>作者：{article.author}</span><span>归属：{article.ownerName || '未知'}</span><span>浏览 {article.views}</span><span>{new Date(article.updatedAt).toLocaleString()}</span></div></div>
    <Space wrap className="article-library-actions"><Tooltip title="在安全预览窗口中查看排版"><Button icon={<EyeOutlined />} onClick={() => onPreview(article)}>预览</Button></Tooltip><Button icon={<EditOutlined />} onClick={() => onEdit(article)}>编辑</Button><Button type={isPublished ? 'default' : 'primary'} icon={<SendOutlined />} onClick={() => onToggle(article)}>{isPublished ? '下架' : '发布'}</Button><Popconfirm title="确认删除此文章？此操作不可恢复。" okText="删除" cancelText="取消" onConfirm={() => onDelete(article.id)}><Button danger icon={<DeleteOutlined />}>删除</Button></Popconfirm></Space>
  </article>;
}

function RichTextEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const editorRef = useRef<HTMLDivElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const videoInputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState('');

  useEffect(() => {
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value || '';
    }
  }, [value]);

  const sync = () => onChange(editorRef.current?.innerHTML ?? '');
  const command = (name: string, commandValue?: string) => {
    editorRef.current?.focus();
    document.execCommand(name, false, commandValue);
    sync();
  };
  const insertHTML = (html: string) => {
    editorRef.current?.focus();
    document.execCommand('insertHTML', false, html);
    sync();
  };
  const createLink = () => {
    const url = normalizeExternalUrl(window.prompt('请输入链接地址（https://…）') ?? '');
    if (url) command('createLink', url);
  };
  const insertExternalImage = () => {
    const url = normalizeExternalUrl(window.prompt('请输入图片 URL（https://…）') ?? '');
    if (url) insertHTML(`<figure class="article-media image-media"><img src="${escapeHtmlAttribute(url)}" alt="文章图片" loading="lazy" /><figcaption>图片</figcaption></figure><p><br /></p>`);
  };
  const insertExternalVideo = () => {
    const url = normalizeExternalUrl(window.prompt('请输入视频 URL（mp4 / webm / ogg，https://…）') ?? '');
    if (url) insertHTML(`<figure class="article-media video-media"><video controls preload="metadata" src="${escapeHtmlAttribute(url)}">当前浏览器不支持视频播放。</video><figcaption>视频</figcaption></figure><p><br /></p>`);
  };
  const uploadMedia = async (event: ChangeEvent<HTMLInputElement>, kind: 'image' | 'video') => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    const expectedPrefix = `${kind}/`;
    if (!file.type.startsWith(expectedPrefix)) {
      setUploadError(kind === 'image' ? '请选择图片文件。' : '请选择视频文件。');
      return;
    }
    if (file.size > MAX_UPLOAD_SIZE) {
      setUploadError(`文件不能超过 ${formatUploadSize(MAX_UPLOAD_SIZE)}。`);
      return;
    }
    setUploadError('');
    setIsUploading(true);
    try {
      const formData = new FormData();
      formData.set('file', file);
      formData.set('displayName', file.name);
      formData.set('category', kind === 'image' ? '文章图片' : '文章视频');
      formData.set('description', `文章富文本本地${kind === 'image' ? '图片' : '视频'}资源`);
      const response = await requestWithSession(`${API_BASE_URL}/api/files`, { method: 'POST', body: formData });
      if (!response.ok) throw new Error(await readMediaUploadError(response));
      const uploaded = (await response.json()) as { id: number; displayName?: string };
      const source = `${API_BASE_URL}/api/files/${uploaded.id}/preview`;
      const label = escapeHtmlAttribute(uploaded.displayName || file.name);
      if (kind === 'image') {
        insertHTML(`<figure class="article-media image-media"><img src="${source}" alt="${label}" loading="lazy" /><figcaption>${label}</figcaption></figure><p><br /></p>`);
      } else {
        insertHTML(`<figure class="article-media video-media"><video controls preload="metadata" src="${source}">当前浏览器不支持视频播放。</video><figcaption>${label}</figcaption></figure><p><br /></p>`);
      }
    } catch (error) {
      setUploadError(error instanceof Error ? error.message : '媒体上传失败，请重试。');
    } finally {
      setIsUploading(false);
    }
  };
  return <section className="rich-editor-section">
    <div className="rich-editor-heading"><strong>正文</strong><span>支持本地上传或 URL 插入图片、视频；本地媒体保存至文件库。</span></div>
    <div className="rich-editor-toolbar" role="toolbar" aria-label="富文本工具栏">
      <ToolbarButton label="加粗" icon={<BoldOutlined />} onClick={() => command('bold')} />
      <ToolbarButton label="斜体" icon={<ItalicOutlined />} onClick={() => command('italic')} />
      <ToolbarButton label="删除线" icon={<StrikethroughOutlined />} onClick={() => command('strikeThrough')} />
      <ToolbarButton label="一级标题" text="H1" onClick={() => command('formatBlock', 'h1')} />
      <ToolbarButton label="二级标题" text="H2" onClick={() => command('formatBlock', 'h2')} />
      <ToolbarButton label="无序列表" icon={<UnorderedListOutlined />} onClick={() => command('insertUnorderedList')} />
      <ToolbarButton label="有序列表" icon={<OrderedListOutlined />} onClick={() => command('insertOrderedList')} />
      <ToolbarButton label="插入链接" icon={<LinkOutlined />} onClick={createLink} />
      <ToolbarButton label="插入图片 URL" icon={<PictureOutlined />} onClick={insertExternalImage} />
      <ToolbarButton label="上传本地图片" text="本地图" onClick={() => imageInputRef.current?.click()} />
      <ToolbarButton label="插入视频 URL" icon={<PlayCircleOutlined />} onClick={insertExternalVideo} />
      <ToolbarButton label="上传本地视频" text="本地视频" onClick={() => videoInputRef.current?.click()} />
      <ToolbarButton label="清除格式" text="Tx" onClick={() => command('removeFormat')} />
    </div>
    <input ref={imageInputRef} className="media-upload-input" type="file" accept="image/*" onChange={(event) => uploadMedia(event, 'image')} />
    <input ref={videoInputRef} className="media-upload-input" type="file" accept="video/mp4,video/webm,video/ogg,video/quicktime" onChange={(event) => uploadMedia(event, 'video')} />
    {isUploading && <div className="rich-editor-media-state">正在上传并插入媒体…</div>}
    {uploadError && <div className="rich-editor-media-error">{uploadError}</div>}
    <div ref={editorRef} className="rich-editor-content" contentEditable suppressContentEditableWarning data-placeholder="从这里开始写作……" onInput={(event) => onChange(event.currentTarget.innerHTML)} />
  </section>;
}

async function readMediaUploadError(response: Response) {
  try { const payload = (await response.json()) as { error?: string }; return payload.error || '媒体上传失败。'; } catch { return '媒体上传失败。'; }
}
function normalizeExternalUrl(value: string) {
  const url = value.trim();
  return /^https?:\/\//i.test(url) ? url : '';
}
function escapeHtmlAttribute(value: string) {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
function formatUploadSize(size: number) { return size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(0)} MB` : `${Math.ceil(size / 1024)} KB`; }
function ToolbarButton({ label, icon, text, onClick }: { label: string; icon?: ReactNode; text?: string; onClick: () => void }) { return <Tooltip title={label}><Button aria-label={label} type="text" onMouseDown={(event) => event.preventDefault()} onClick={onClick}>{icon ?? text}</Button></Tooltip>; }
function ArticlePreview({ article, onClose }: { article: Article | null; onClose: () => void }) {
  const safeContent = article ? sanitizeArticleHtml(article.content || '<p>暂无正文内容。</p>') : '';
  const [imageSource, setImageSource] = useState<string | null>(null);
  const handlePreviewContentClick = (event: ReactMouseEvent<HTMLDivElement>) => {
    const image = (event.target as HTMLElement).closest('img');
    if (image instanceof HTMLImageElement && image.currentSrc) {
      setImageSource(image.currentSrc);
    }
  };

  return <>
    <Modal className="article-preview-modal" open={Boolean(article)} title={article?.title} footer={<Button onClick={onClose}>关闭预览</Button>} width="min(1320px, 97vw)" onCancel={onClose} destroyOnClose>
      <article className="article-preview">
        <div className="article-preview-meta"><Tag color="blue">{article?.category}</Tag><Tag color={article?.isPrivate ? 'warning' : 'default'}>{article?.isPrivate ? '私密' : '公开'}</Tag><span>作者：{article?.author}</span><span>归属：{article?.ownerName || '未知'}</span><span>{article && new Date(article.updatedAt).toLocaleString()}</span><span>点击图片可放大、缩放和拖动查看</span></div>
        {article?.summary && <p className="article-preview-summary">{article.summary}</p>}
        <div className="article-preview-content" onClick={handlePreviewContentClick} dangerouslySetInnerHTML={{ __html: safeContent }} />
      </article>
    </Modal>
    <ImageZoomPreview source={imageSource} onClose={() => setImageSource(null)} />
  </>;
}

function ImageZoomPreview({ source, onClose }: { source: string | null; onClose: () => void }) {
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  useEffect(() => { setScale(1); setOffset({ x: 0, y: 0 }); }, [source]);
  const adjustScale = (amount: number) => setScale((current) => Math.max(0.35, Math.min(5, Number((current + amount).toFixed(2)))));
  const reset = () => { setScale(1); setOffset({ x: 0, y: 0 }); };
  return <Modal className="article-image-zoom-modal" open={Boolean(source)} title="图片放大预览" footer={null} width="min(1500px, 98vw)" onCancel={onClose} destroyOnClose>
    <div className="image-zoom-toolbar">
      <Space>
        <Button aria-label="缩小图片" icon={<MinusOutlined />} onClick={() => adjustScale(-0.25)}>缩小</Button>
        <Button aria-label="放大图片" icon={<PlusOutlined />} onClick={() => adjustScale(0.25)}>放大</Button>
        <Button aria-label="适配图片" icon={<ReloadOutlined />} onClick={reset}>适配</Button>
      </Space>
      <span>{Math.round(scale * 100)}% · 滚轮缩放，按住图片拖动</span>
    </div>
    <div className="image-zoom-stage" onWheelCapture={(event) => { event.preventDefault(); adjustScale(event.deltaY < 0 ? 0.15 : -0.15); }} onPointerMove={(event) => {
      const drag = dragRef.current;
      if (!drag) return;
      setOffset({ x: drag.originX + event.clientX - drag.x, y: drag.originY + event.clientY - drag.y });
    }} onPointerUp={() => { dragRef.current = null; }} onPointerLeave={() => { dragRef.current = null; }}>
      {source && <img src={source} alt="文章原图预览" draggable={false} onPointerDown={(event) => { dragRef.current = { x: event.clientX, y: event.clientY, originX: offset.x, originY: offset.y }; event.currentTarget.setPointerCapture(event.pointerId); }} style={{ transform: `translate(${offset.x}px, ${offset.y}px) scale(${scale})` }} />}
    </div>
  </Modal>;
}

function sanitizeArticleHtml(input: string) {
  if (typeof window === 'undefined') return '';
  const template = document.createElement('template');
  template.innerHTML = input;
  const allowedTags = new Set(['A', 'B', 'BR', 'BLOCKQUOTE', 'CODE', 'DIV', 'EM', 'FIGCAPTION', 'FIGURE', 'H1', 'H2', 'H3', 'H4', 'HR', 'I', 'IMG', 'LI', 'OL', 'P', 'PRE', 'S', 'SOURCE', 'SPAN', 'STRONG', 'U', 'UL', 'VIDEO']);
  template.content.querySelectorAll('*').forEach((node) => {
    if (!allowedTags.has(node.tagName)) { node.replaceWith(...Array.from(node.childNodes)); return; }
    Array.from(node.attributes).forEach((attribute) => {
      const name = attribute.name.toLowerCase();
      const value = attribute.value.trim();
      const isSafeMediaSource = /^(https?:\/\/|\/api\/files\/)/i.test(value) || value.startsWith(`${API_BASE_URL}/api/files/`);
      if (node.tagName === 'A' && name === 'href' && /^(https?:|mailto:|#)/i.test(value)) return;
      if ((node.tagName === 'IMG' || node.tagName === 'VIDEO' || node.tagName === 'SOURCE') && name === 'src' && isSafeMediaSource) return;
      if (node.tagName === 'IMG' && name === 'alt') return;
      if (node.tagName === 'VIDEO' && (name === 'controls' || name === 'preload')) return;
      if ((node.tagName === 'FIGURE' || node.tagName === 'DIV') && name === 'class' && /^article-media\s+(image-media|video-media)$/.test(value)) return;
      node.removeAttribute(attribute.name);
    });
    if (node.tagName === 'A') { node.setAttribute('target', '_blank'); node.setAttribute('rel', 'noopener noreferrer'); }
  });
  return template.innerHTML;
}
