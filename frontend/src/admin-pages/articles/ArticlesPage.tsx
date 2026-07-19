'use client';

import { useEffect, useRef, useState, type ChangeEvent, type FormEvent, type MouseEvent as ReactMouseEvent, type ReactNode } from 'react';
import {
  AppstoreOutlined,
  BoldOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  ExportOutlined,
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
import { Alert, App, Button, Card, Dropdown, Empty, Input, Modal, Popconfirm, Select, Space, Statistic, Switch, Tag, Tooltip } from 'antd';
import type { Article, ArticleForm } from '@/src/types/admin';
import type { ResourceActionAccess } from '@/src/utils/actionPermissions';
import { API_BASE_URL, articleStatusOptions, MAX_UPLOAD_SIZE } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';
import { articleExportOptions, exportArticle, type ArticleExportFormat } from '@/src/utils/articleExport';

type ArticlesPageProps = {
  filteredArticles: Article[];
  actions: ResourceActionAccess;
  articleForm: ArticleForm;
  editingArticleId: number | null;
  articleKeyword: string;
  articleStatus: string;
  isSavingArticle: boolean;
  onArticleFormChange: (form: ArticleForm) => void;
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => Promise<boolean>;
  onResetArticleForm: () => void;
  onArticleKeywordChange: (keyword: string) => void;
  onArticleStatusChange: (status: string) => void;
  onResetFilters: () => void;
  onEditArticle: (article: Article) => void;
  onToggleArticleStatus: (article: Article) => void;
  onDeleteArticle: (articleId: number) => void;
};

export function ArticlesPage(props: ArticlesPageProps) {
  const { message: feedbackMessage } = App.useApp();
  const {
    filteredArticles, actions, articleForm, editingArticleId, articleKeyword, articleStatus, isSavingArticle,
    onArticleFormChange, onSubmitArticle, onResetArticleForm, onArticleKeywordChange,
    onArticleStatusChange, onResetFilters, onEditArticle, onToggleArticleStatus, onDeleteArticle,
  } = props;
  const [previewArticle, setPreviewArticle] = useState<Article | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editorArticleId, setEditorArticleId] = useState<number | null>(null);
  const [exportingArticle, setExportingArticle] = useState<{ articleId: number; format: ArticleExportFormat } | null>(null);
  const [exportFeedback, setExportFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const isEditing = editorArticleId !== null;
  const published = filteredArticles.filter((article) => article.status === '已发布').length;

  const openNew = () => { if (!actions.create) return; onResetArticleForm(); setEditorArticleId(null); setIsEditorOpen(true); };
  const openEdit = (article: Article) => { if (!actions.update) return; setEditorArticleId(article.id); setIsEditorOpen(true); onEditArticle(article); };
  const closeEditor = () => { onResetArticleForm(); setEditorArticleId(null); setIsEditorOpen(false); };
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    if (isEditing ? !actions.update : !actions.create) return;
    if (await onSubmitArticle(event)) {
      setEditorArticleId(null);
      setIsEditorOpen(false);
    }
  };
  const handleExport = async (article: Article, format: ArticleExportFormat) => {
    setExportFeedback(null);
    setExportingArticle({ articleId: article.id, format });
    try {
      const exportedMessage = await exportArticle(article, format);
      setExportFeedback({ type: 'success', message: exportedMessage });
      void feedbackMessage.success(exportedMessage);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : `《${article.title}》导出失败，请重试。`;
      setExportFeedback({ type: 'error', message: errorMessage });
      void feedbackMessage.error(errorMessage);
    } finally {
      setExportingArticle(null);
    }
  };

  return <section className="page-stack article-workspace" aria-labelledby="articles-page-title">
    <Card className="article-hero" variant="borderless">
      <div><p className="page-kicker">内容管理 / 写作中心</p><h1 id="articles-page-title">文章管理</h1><span>在富文本编辑器中创作、预览、保存和发布文章；所有操作均直连后端接口。</span></div>
      <Space className="article-hero-actions" size={12} wrap>
        {actions.create && <Button className="article-create-button" icon={<PlusOutlined />} onClick={openNew}>新建文章</Button>}
        <Button className="article-clear-filter-button" icon={<AppstoreOutlined />} onClick={onResetFilters}>清空筛选条件</Button>
      </Space>
    </Card>

    {exportFeedback && <Alert className="article-export-feedback" type={exportFeedback.type} title={exportFeedback.message} showIcon closable={{ onClose: () => setExportFeedback(null) }} />}

    <div className="article-stat-grid"><Card><Statistic title="当前结果" value={filteredArticles.length} prefix={<FileTextOutlined />} /></Card><Card><Statistic title="已发布" value={published} suffix="篇" prefix={<SendOutlined />} /></Card><Card><Statistic title="草稿 / 下架" value={filteredArticles.length - published} suffix="篇" prefix={<EditOutlined />} /></Card></div>

    <Card className="article-browser-card" title="文章库" extra={<Space className="article-filter-bar"><Input allowClear value={articleKeyword} onChange={(event) => onArticleKeywordChange(event.target.value)} placeholder="标题、分类、作者、摘要" prefix={<FileTextOutlined />} /><Select value={articleStatus} onChange={onArticleStatusChange} options={[{ value: '全部', label: '全部状态' }, ...articleStatusOptions.map((status) => ({ value: status, label: status }))]} /><Button onClick={onResetFilters}>重置</Button></Space>}>
      {filteredArticles.length === 0 ? <Empty description="暂无匹配文章">{actions.create && <Button type="primary" onClick={openNew}>创建第一篇文章</Button>}</Empty> : <div className="article-card-list" aria-label="文章列表">{filteredArticles.map((article) => <ArticleCard key={article.id} article={article} actions={actions} exportingFormat={exportingArticle?.articleId === article.id ? exportingArticle.format : null} exportDisabled={Boolean(exportingArticle)} onExport={handleExport} onPreview={setPreviewArticle} onEdit={openEdit} onToggle={onToggleArticleStatus} onDelete={onDeleteArticle} />)}</div>}
    </Card>

    {(actions.create || actions.update) && <Modal open={isEditorOpen} title={isEditing ? '编辑文章' : '新建文章'} footer={null} width="min(1160px, 96vw)" destroyOnHidden onCancel={closeEditor}>
      <form className="rich-editor-form" onSubmit={(event) => void submit(event)}>
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
            <small>开启后仅归属人和管理员可查看；其他登录用户不会在列表中看到此文章。</small>
          </div>
          <Switch checked={Boolean(articleForm.isPrivate)} onChange={(checked) => onArticleFormChange({ ...articleForm, isPrivate: checked })} checkedChildren="私密" unCheckedChildren="公开" />
        </div>
        <RichTextEditor value={articleForm.content} onChange={(content) => onArticleFormChange({ ...articleForm, content })} />
        <div className="rich-editor-actions"><Button onClick={closeEditor}>取消</Button><Button htmlType="submit" type="primary" loading={isSavingArticle} icon={<SaveOutlined />}>{isEditing ? '保存修改' : '保存文章'}</Button></div>
      </form>
    </Modal>}

    <ArticlePreview article={previewArticle} onClose={() => setPreviewArticle(null)} />
  </section>;
}

type ArticleCardProps = {
  article: Article;
  actions: ResourceActionAccess;
  exportingFormat: ArticleExportFormat | null;
  exportDisabled: boolean;
  onExport: (article: Article, format: ArticleExportFormat) => Promise<void>;
  onPreview: (article: Article) => void;
  onEdit: (article: Article) => void;
  onToggle: (article: Article) => void;
  onDelete: (articleId: number) => void;
};
function ArticleCard({ article, actions, exportingFormat, exportDisabled, onExport, onPreview, onEdit, onToggle, onDelete }: ArticleCardProps) {
  const isPublished = article.status === '已发布';
  const isIndexable = isPublished && !article.isPrivate;
  const titleId = `article-title-${article.id}`;
  return <article className="article-library-card" aria-labelledby={titleId} itemScope={isIndexable} itemType={isIndexable ? 'https://schema.org/Article' : undefined}>
    <div className="article-library-main"><div className="article-library-title"><h3 id={titleId} itemProp={isIndexable ? 'headline' : undefined}>{article.title}</h3><Space size={6} wrap><Tag color={isPublished ? 'success' : article.status === '下架' ? 'default' : 'processing'}>{article.status}</Tag><Tag color={article.isPrivate ? 'warning' : 'blue'}>{article.isPrivate ? '私密' : '公开'}</Tag></Space></div><p itemProp={isIndexable ? 'description' : undefined}>{article.summary || '暂无摘要，打开文章后可补充内容概览。'}</p><div className="article-library-meta"><span itemProp={isIndexable ? 'articleSection' : undefined}>{article.category}</span><span itemProp={isIndexable ? 'author' : undefined}>作者：{article.author}</span><span>归属：{article.ownerName || '未知'}</span><span>浏览 {article.views}</span><time itemProp={isIndexable ? 'dateModified' : undefined} dateTime={article.updatedAt}>{new Date(article.updatedAt).toLocaleString()}</time></div></div>
    <Space wrap className="article-library-actions">
      <Tooltip title="在安全预览窗口中查看排版"><Button icon={<EyeOutlined />} onClick={() => onPreview(article)}>预览</Button></Tooltip>
      <Dropdown
        rootClassName="article-export-dropdown"
        trigger={['click']}
        disabled={exportDisabled}
        menu={{
          items: articleExportOptions.map((option) => ({ key: option.key, label: option.label })),
          onClick: ({ key }) => void onExport(article, key as ArticleExportFormat),
        }}
      >
        <Button type="primary" className="article-export-button" icon={<ExportOutlined />} loading={Boolean(exportingFormat)} aria-label={`导出《${article.title}》完整内容`}>
          {exportingFormat ? '正在导出' : '导出全文'} {!exportingFormat && <DownOutlined />}
        </Button>
      </Dropdown>
      {actions.update && <Button icon={<EditOutlined />} onClick={() => onEdit(article)}>编辑</Button>}
      {actions.update && <Button type={isPublished ? 'default' : 'primary'} icon={<SendOutlined />} onClick={() => onToggle(article)}>{isPublished ? '下架' : '发布'}</Button>}
      {actions.delete && <Popconfirm title="确认删除此文章？此操作不可恢复。" okText="删除" cancelText="取消" onConfirm={() => onDelete(article.id)}><Button danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>}
    </Space>
  </article>;
}

function RichTextEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const editorRef = useRef<HTMLDivElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const videoInputRef = useRef<HTMLInputElement>(null);
  const selectionRef = useRef<Range | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState('');
  const [externalMedia, setExternalMedia] = useState<{ kind: 'link' | 'image' | 'video'; url: string; description: string } | null>(null);
  const [externalMediaError, setExternalMediaError] = useState('');

  useEffect(() => {
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value || '';
    }
  }, [value]);

  const sync = () => onChange(editorRef.current?.innerHTML ?? '');
  const restoreSelection = () => {
    const selection = window.getSelection();
    if (!selectionRef.current || !selection) return;
    selection.removeAllRanges();
    selection.addRange(selectionRef.current);
  };
  const command = (name: string, commandValue?: string) => {
    editorRef.current?.focus();
    restoreSelection();
    document.execCommand(name, false, commandValue);
    sync();
  };
  const insertHTML = (html: string) => {
    editorRef.current?.focus();
    restoreSelection();
    document.execCommand('insertHTML', false, html);
    sync();
  };
  const openExternalMediaDialog = (kind: 'link' | 'image' | 'video') => {
    const selection = window.getSelection();
    if (selection?.rangeCount) selectionRef.current = selection.getRangeAt(0).cloneRange();
    setExternalMediaError('');
    setExternalMedia({ kind, url: '', description: kind === 'image' ? '文章配图' : '' });
  };
  const confirmExternalMedia = () => {
    if (!externalMedia) return;
    const url = normalizeExternalUrl(externalMedia.url);
    if (!url) {
      setExternalMediaError('请输入以 http:// 或 https:// 开头的有效地址。');
      return;
    }
    if (externalMedia.kind === 'link') command('createLink', url);
    if (externalMedia.kind === 'image') {
      const safeDescription = escapeHtmlAttribute(externalMedia.description.trim() || '文章配图');
      insertHTML(`<figure class="article-media image-media"><img src="${escapeHtmlAttribute(url)}" alt="${safeDescription}" title="${safeDescription}" loading="lazy" decoding="async" /><figcaption>${safeDescription}</figcaption></figure><p><br /></p>`);
    }
    if (externalMedia.kind === 'video') {
      insertHTML(`<figure class="article-media video-media"><video controls preload="metadata" src="${escapeHtmlAttribute(url)}">当前浏览器不支持视频播放。</video><figcaption>视频</figcaption></figure><p><br /></p>`);
    }
    setExternalMedia(null);
    setExternalMediaError('');
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
        insertHTML(`<figure class="article-media image-media"><img src="${source}" alt="${label}" title="${label}" loading="lazy" decoding="async" /><figcaption>${label}</figcaption></figure><p><br /></p>`);
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
      <ToolbarButton label="插入链接" icon={<LinkOutlined />} onClick={() => openExternalMediaDialog('link')} />
      <ToolbarButton label="插入图片 URL" icon={<PictureOutlined />} onClick={() => openExternalMediaDialog('image')} />
      <ToolbarButton label="上传本地图片" text="本地图" onClick={() => imageInputRef.current?.click()} />
      <ToolbarButton label="插入视频 URL" icon={<PlayCircleOutlined />} onClick={() => openExternalMediaDialog('video')} />
      <ToolbarButton label="上传本地视频" text="本地视频" onClick={() => videoInputRef.current?.click()} />
      <ToolbarButton label="清除格式" text="Tx" onClick={() => command('removeFormat')} />
    </div>
    <input ref={imageInputRef} className="media-upload-input" type="file" accept="image/*" onChange={(event) => uploadMedia(event, 'image')} />
    <input ref={videoInputRef} className="media-upload-input" type="file" accept="video/mp4,video/webm,video/ogg,video/quicktime" onChange={(event) => uploadMedia(event, 'video')} />
    {isUploading && <div className="rich-editor-media-state">正在上传并插入媒体…</div>}
    {uploadError && <div className="rich-editor-media-error">{uploadError}</div>}
    <div ref={editorRef} className="rich-editor-content" contentEditable suppressContentEditableWarning data-placeholder="从这里开始写作……" onInput={(event) => onChange(event.currentTarget.innerHTML)} />
    <Modal
      open={Boolean(externalMedia)}
      title={externalMedia?.kind === 'link' ? '插入链接' : externalMedia?.kind === 'image' ? '插入图片 URL' : '插入视频 URL'}
      okText="插入"
      cancelText="取消"
      onOk={confirmExternalMedia}
      onCancel={() => { setExternalMedia(null); setExternalMediaError(''); }}
      destroyOnHidden
    >
      <Space direction="vertical" size={14} style={{ width: '100%' }}>
        {externalMediaError && <Alert type="error" showIcon title={externalMediaError} />}
        <label className="article-dialog-field">
          地址
          <Input value={externalMedia?.url ?? ''} placeholder="https://example.com/resource" onChange={(event) => externalMedia && setExternalMedia({ ...externalMedia, url: event.target.value })} onPressEnter={confirmExternalMedia} />
        </label>
        {externalMedia?.kind === 'image' && (
          <label className="article-dialog-field">
            图片说明
            <Input value={externalMedia.description} placeholder="用于替代文本和内容检索" onChange={(event) => setExternalMedia({ ...externalMedia, description: event.target.value })} />
          </label>
        )}
      </Space>
    </Modal>
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
  const isIndexable = Boolean(article && article.status === '已发布' && !article.isPrivate);
  const structuredData = article && isIndexable ? JSON.stringify({
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: article.summary || undefined,
    articleSection: article.category,
    author: { '@type': 'Person', name: article.author },
    datePublished: article.createdAt,
    dateModified: article.updatedAt,
    isAccessibleForFree: true,
  }).replace(/</g, '\\u003c') : '';
  const handlePreviewContentClick = (event: ReactMouseEvent<HTMLDivElement>) => {
    const image = (event.target as HTMLElement).closest('img');
    if (image instanceof HTMLImageElement && image.currentSrc) {
      setImageSource(image.currentSrc);
    }
  };

  return <>
    <Modal className="article-preview-modal" open={Boolean(article)} title="文章预览" footer={<Button onClick={onClose}>关闭预览</Button>} width="min(1320px, 97vw)" onCancel={onClose} destroyOnHidden>
      <article className="article-preview" itemScope={isIndexable} itemType={isIndexable ? 'https://schema.org/Article' : undefined}>
        {structuredData && <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: structuredData }} />}
        <h1 className="article-preview-title" itemProp={isIndexable ? 'headline' : undefined}>{article?.title}</h1>
        <div className="article-preview-meta"><Tag color="blue"><span itemProp={isIndexable ? 'articleSection' : undefined}>{article?.category}</span></Tag><Tag color={article?.isPrivate ? 'warning' : 'default'}>{article?.isPrivate ? '私密' : '公开'}</Tag><span itemProp={isIndexable ? 'author' : undefined}>作者：{article?.author}</span><span>归属：{article?.ownerName || '未知'}</span>{article && <time itemProp={isIndexable ? 'dateModified' : undefined} dateTime={article.updatedAt}>{new Date(article.updatedAt).toLocaleString()}</time>}<span>点击图片可放大、缩放和拖动查看</span></div>
        {article?.summary && <p className="article-preview-summary">{article.summary}</p>}
        <div className="article-preview-content" itemProp={isIndexable ? 'articleBody' : undefined} onClick={handlePreviewContentClick} dangerouslySetInnerHTML={{ __html: safeContent }} />
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
  // Keep a lower bound so the preview remains recoverable, but do not cap
  // the upper bound: large source images may need more than a few hundred %.
  const adjustScale = (amount: number) => setScale((current) => Math.max(0.1, Number((current + amount).toFixed(2))));
  const reset = () => { setScale(1); setOffset({ x: 0, y: 0 }); };
  return <Modal className="article-image-zoom-modal" open={Boolean(source)} title="图片放大预览" footer={null} width="min(1500px, 98vw)" onCancel={onClose} destroyOnHidden>
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
      if (node.tagName === 'IMG' && ['alt', 'title', 'loading', 'decoding'].includes(name)) return;
      if (node.tagName === 'VIDEO' && (name === 'controls' || name === 'preload')) return;
      if ((node.tagName === 'FIGURE' || node.tagName === 'DIV') && name === 'class' && /^article-media\s+(image-media|video-media)$/.test(value)) return;
      node.removeAttribute(attribute.name);
    });
    if (node.tagName === 'A') { node.setAttribute('target', '_blank'); node.setAttribute('rel', 'noopener noreferrer'); }
    if (node.tagName === 'IMG') {
      if (!node.getAttribute('alt')?.trim()) node.setAttribute('alt', '文章内容图片');
      node.setAttribute('loading', 'lazy');
      node.setAttribute('decoding', 'async');
    }
  });
  return template.innerHTML;
}
