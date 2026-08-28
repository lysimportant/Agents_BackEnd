'use client';

import { useEffect, useRef, useState, type ChangeEvent, type FormEvent, type MouseEvent as ReactMouseEvent, type ReactNode } from 'react';
import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  BoldOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  ExportOutlined,
  FileAddOutlined,
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
  /** filteredArticles 表示筛选后。 */
  filteredArticles: Article[];
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** articleForm 表示文章表单。 */
  articleForm: ArticleForm;
  /** articleKeyword 表示文章搜索关键词。 */
  articleKeyword: string;
  /** articleStatus 表示文章状态。 */
  articleStatus: string;
  /** isSavingArticle 表示文章。 */
  isSavingArticle: boolean;
  /** onArticleFormChange 表示文章表单。 */
  onArticleFormChange: (form: ArticleForm) => void;
  /** onSubmitArticle 表示文章。 */
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => Promise<boolean>;
  /** onResetArticleForm 表示文章表单。 */
  onResetArticleForm: () => void;
  /** onArticleKeywordChange 表示文章搜索关键词。 */
  onArticleKeywordChange: (keyword: string) => void;
  /** onArticleStatusChange 表示文章状态。 */
  onArticleStatusChange: (status: string) => void;
  /** onResetFilters 表示筛选条件。 */
  onResetFilters: () => void;
  /** onEditArticle 表示文章。 */
  onEditArticle: (article: Article) => void;
  /** onToggleArticleStatus 表示文章状态。 */
  onToggleArticleStatus: (article: Article) => void;
  /** onDeleteArticle 表示文章。 */
  onDeleteArticle: (articleId: number) => void;
};

/** ArticlesPage 实现对应业务逻辑。 */
export function ArticlesPage(props: ArticlesPageProps) {
  /** message、feedbackMessage 保存消息、消息。 */
  const { message: feedbackMessage } = App.useApp();
  const {
    filteredArticles, actions, articleForm, articleKeyword, articleStatus, isSavingArticle,
    onArticleFormChange, onSubmitArticle, onResetArticleForm, onArticleKeywordChange,
    onArticleStatusChange, onResetFilters, onEditArticle, onToggleArticleStatus, onDeleteArticle,
  } = props;
  /** previewArticle、setPreviewArticle 保存预览文章、预览文章。 */
  const [previewArticle, setPreviewArticle] = useState<Article | null>(null);
  /** isEditorOpen、setIsEditorOpen 分别保存编辑器打开状态状态及其更新函数。 */
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  /** isCreating、setIsCreating 保存文章管理主内容区是否切换为创建视图。 */
  const [isCreating, setIsCreating] = useState(false);
  /** exportingArticle、setExportingArticle 保存文章、文章。 */
  const [exportingArticle, setExportingArticle] = useState<{ articleId: number; format: ArticleExportFormat } | null>(null);
  /** exportFeedback、setExportFeedback 保存导出反馈、导出操作。 */
  const [exportFeedback, setExportFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  /** published 负责计算或维护发布状态。 */
  const published = filteredArticles.filter((article) => article.status === '已发布').length;

  /** openNew 清空旧表单并在文章管理主内容区打开创建视图。 */
  const openNew = () => {
    if (!actions.create) return;
    onResetArticleForm();
    setIsCreating(true);
  };
  /** openEdit 负责计算或维护打开状态。 */
  const openEdit = (article: Article) => { if (!actions.update) return; setIsEditorOpen(true); onEditArticle(article); };
  /** closeEditor 负责删除或清理对应业务状态。 */
  const closeEditor = () => { onResetArticleForm(); setIsEditorOpen(false); };
  /** submit 负责执行对应业务操作。 */
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    if (!actions.update) return;
    if (await onSubmitArticle(event)) {
      setIsEditorOpen(false);
    }
  };
  /** cancelCreate 清空未提交内容并回到同一文章管理列表。 */
  const cancelCreate = () => {
    onResetArticleForm();
    setIsCreating(false);
  };
  /** submitCreate 创建成功后留在当前工作台并显示更新后的文章列表。 */
  const submitCreate = async (event: FormEvent<HTMLFormElement>) => {
    if (!actions.create) return;
    if (await onSubmitArticle(event)) {
      setIsCreating(false);
    }
  };
  /** handleExport 负责处理对应的界面事件和状态变化。 */
  const handleExport = async (article: Article, format: ArticleExportFormat) => {
    setExportFeedback(null);
    setExportingArticle({ articleId: article.id, format });
    try {
      /** exportedMessage 保存消息。 */
      const exportedMessage = await exportArticle(article, format);
      setExportFeedback({ type: 'success', message: exportedMessage });
      void feedbackMessage.success(exportedMessage);
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = error instanceof Error ? error.message : `《${article.title}》导出失败，请重试。`;
      setExportFeedback({ type: 'error', message: errorMessage });
      void feedbackMessage.error(errorMessage);
    } finally {
      setExportingArticle(null);
    }
  };

  return <section className="page-stack article-workspace" aria-labelledby={isCreating ? 'article-create-title' : 'articles-page-title'}>
    {isCreating ? <ArticleCreateView
      articleForm={articleForm}
      isSavingArticle={isSavingArticle}
      onArticleFormChange={onArticleFormChange}
      onSubmitArticle={submitCreate}
      onCancel={cancelCreate}
    /> : <>
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
    </>}

    {actions.update && <Modal open={isEditorOpen} title="编辑文章" footer={null} width="min(1160px, 96vw)" destroyOnHidden onCancel={closeEditor}>
      <ArticleEditorForm
        mode="edit"
        articleForm={articleForm}
        isSavingArticle={isSavingArticle}
        onArticleFormChange={onArticleFormChange}
        onSubmitArticle={submit}
        onCancel={closeEditor}
      />
    </Modal>}

    <ArticlePreview article={previewArticle} onClose={() => setPreviewArticle(null)} />
  </section>;
}

type ArticleCreateViewProps = {
  /** articleForm 保存待创建文章的完整表单内容。 */
  articleForm: ArticleForm;
  /** isSavingArticle 表示创建请求是否正在进行。 */
  isSavingArticle: boolean;
  /** onArticleFormChange 在字段变化时同步待创建文章。 */
  onArticleFormChange: (form: ArticleForm) => void;
  /** onSubmitArticle 创建文章，并在成功后回到列表视图。 */
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  /** onCancel 放弃本次创建并回到列表视图。 */
  onCancel: () => void;
};

/** ArticleCreateView 在文章管理主内容区承载文章创建表单，不改变工作台路由。 */
function ArticleCreateView({
  articleForm,
  isSavingArticle,
  onArticleFormChange,
  onSubmitArticle,
  onCancel,
}: ArticleCreateViewProps) {
  return <div className="article-create-page">
    <header className="article-create-page-header">
      <Button className="article-create-back" type="text" icon={<ArrowLeftOutlined />} onClick={onCancel}>返回文章管理</Button>
      <div className="article-create-heading">
        <span className="article-create-heading-icon" aria-hidden="true"><FileAddOutlined /></span>
        <div>
          <p className="page-kicker">内容管理 / 文章管理</p>
          <h1 id="article-create-title">创建文章</h1>
        </div>
      </div>
    </header>

    <div className="article-create-surface" data-tilt-disabled="true">
      <ArticleEditorForm
        mode="create"
        articleForm={articleForm}
        isSavingArticle={isSavingArticle}
        onArticleFormChange={onArticleFormChange}
        onSubmitArticle={onSubmitArticle}
        onCancel={onCancel}
      />
    </div>
  </div>;
}

type ArticleEditorFormProps = {
  /** mode 决定提交按钮显示创建还是编辑语义。 */
  mode: 'create' | 'edit';
  /** articleForm 保存当前文章的标题、正文和可见性设置。 */
  articleForm: ArticleForm;
  /** isSavingArticle 表示文章写请求是否正在进行。 */
  isSavingArticle: boolean;
  /** onArticleFormChange 在任一字段变化时同步完整表单。 */
  onArticleFormChange: (form: ArticleForm) => void;
  /** onSubmitArticle 提交当前文章表单，并由上层决定后续导航。 */
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  /** onCancel 放弃当前编辑并返回上层页面。 */
  onCancel: () => void;
};

type ArticleSettingKey = 'title' | 'category' | 'author' | 'summary' | 'status' | 'visibility' | 'is18r' | 'contentLocale';
type ArticleSettingDraft = string | boolean;

/** 文章设置字段名称用于按钮和 Dialog 标题，保持创建与编辑语义一致。 */
const articleSettingLabels: Record<ArticleSettingKey, string> = {
  title: '标题',
  category: '分类',
  author: '作者',
  summary: '摘要',
  status: '状态',
  visibility: '可见性',
  is18r: '18R 限制',
  contentLocale: '正文语言',
};

/** 文章正文语言选项同时服务于设置 Dialog 和提交前的当前值展示。 */
const articleLocaleOptions: Array<{ value: ArticleForm['contentLocale']; label: string }> = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
  { value: 'ja-JP', label: '日本語' },
];

/** 文章设置按钮的固定顺序，避免正文编辑区域上方出现大量输入框。 */
const articleSettingKeys: ArticleSettingKey[] = ['title', 'category', 'author', 'summary', 'status', 'visibility', 'is18r', 'contentLocale'];

/** ArticleEditorForm 复用新建页面与编辑弹窗的完整文章编辑能力。 */
export function ArticleEditorForm({
  mode,
  articleForm,
  isSavingArticle,
  onArticleFormChange,
  onSubmitArticle,
  onCancel,
}: ArticleEditorFormProps) {
  /** activeSettingKey 表示当前正在 Dialog 中编辑的文章字段。 */
  const [activeSettingKey, setActiveSettingKey] = useState<ArticleSettingKey | null>(null);
  /** settingDraft 保存当前 Dialog 的临时输入，取消时不会写回文章表单。 */
  const [settingDraft, setSettingDraft] = useState<ArticleSettingDraft>('');
  /** settingError 保存当前字段的校验反馈。 */
  const [settingError, setSettingError] = useState('');
  /** feedbackMessage 提供继承当前主题的表单校验反馈。 */
  const { message: feedbackMessage } = App.useApp();
  /** updateArticleForm 在正文或设置 Dialog 确认后同步完整文章表单。 */
  const updateArticleForm = (changes: Partial<ArticleForm>) => onArticleFormChange({ ...articleForm, ...changes });

  /** openArticleSettingDialog 打开指定字段的 Dialog，并将当前值复制到临时草稿。 */
  const openArticleSettingDialog = (settingKey: ArticleSettingKey) => {
    setActiveSettingKey(settingKey);
    setSettingError('');
    switch (settingKey) {
      case 'title': setSettingDraft(articleForm.title); break;
      case 'category': setSettingDraft(articleForm.category); break;
      case 'author': setSettingDraft(articleForm.author); break;
      case 'summary': setSettingDraft(articleForm.summary); break;
      case 'status': setSettingDraft(articleForm.status); break;
      case 'visibility': setSettingDraft(Boolean(articleForm.isPrivate)); break;
      case 'is18r': setSettingDraft(Boolean(articleForm.is18r)); break;
      case 'contentLocale': setSettingDraft(articleForm.contentLocale); break;
    }
  };

  /** closeArticleSettingDialog 关闭字段 Dialog，并丢弃未确认的临时输入。 */
  const closeArticleSettingDialog = () => {
    setActiveSettingKey(null);
    setSettingDraft('');
    setSettingError('');
  };

  /** confirmArticleSetting 校验并将当前字段的 Dialog 草稿写回文章表单。 */
  const confirmArticleSetting = () => {
    if (!activeSettingKey) return;
    if (['title', 'category', 'author'].includes(activeSettingKey)) {
      const normalizedValue = String(settingDraft).trim();
      if (!normalizedValue) {
        setSettingError(`请输入${articleSettingLabels[activeSettingKey]}。`);
        return;
      }
      if (activeSettingKey === 'title') updateArticleForm({ title: normalizedValue });
      if (activeSettingKey === 'category') updateArticleForm({ category: normalizedValue });
      if (activeSettingKey === 'author') updateArticleForm({ author: normalizedValue });
    } else if (activeSettingKey === 'summary') {
      updateArticleForm({ summary: String(settingDraft) });
    } else if (activeSettingKey === 'status') {
      updateArticleForm({ status: String(settingDraft) as ArticleForm['status'] });
    } else if (activeSettingKey === 'visibility') {
      updateArticleForm({ isPrivate: settingDraft === true });
    } else if (activeSettingKey === 'is18r') {
      updateArticleForm({ is18r: settingDraft === true });
    } else if (activeSettingKey === 'contentLocale') {
      updateArticleForm({ contentLocale: String(settingDraft) as ArticleForm['contentLocale'] });
    }
    closeArticleSettingDialog();
  };

  /** handleSubmitArticle 在调用工作台提交前校验已移入 Dialog 的必填字段。 */
  const handleSubmitArticle = (event: FormEvent<HTMLFormElement>) => {
    const missingField = [
      { value: articleForm.title, key: 'title' as const },
      { value: articleForm.category, key: 'category' as const },
      { value: articleForm.author, key: 'author' as const },
    ].find(({ value }) => !value.trim());
    if (missingField) {
      event.preventDefault();
      openArticleSettingDialog(missingField.key);
      void feedbackMessage.error(`请先填写${articleSettingLabels[missingField.key]}。`);
      return;
    }
    void onSubmitArticle(event);
  };

  /** getArticleSettingValue 生成按钮上的当前值摘要，避免打开 Dialog 才能确认状态。 */
  const getArticleSettingValue = (settingKey: ArticleSettingKey) => {
    switch (settingKey) {
      case 'title': return articleForm.title.trim() || '未填写';
      case 'category': return articleForm.category.trim() || '未填写';
      case 'author': return articleForm.author.trim() || '未填写';
      case 'summary': return articleForm.summary.trim() || '未填写';
      case 'status': return articleForm.status || '草稿';
      case 'visibility': return articleForm.isPrivate ? '仅自己可见' : '公开可见';
      case 'is18r': return articleForm.is18r ? '已开启' : '未开启';
      case 'contentLocale': return articleLocaleOptions.find((option) => option.value === articleForm.contentLocale)?.label || '简体中文';
    }
  };

  /** renderArticleSettingControl 按字段类型渲染 Dialog 内的受控输入控件。 */
  const renderArticleSettingControl = () => {
    if (!activeSettingKey) return null;
    const stringDraft = typeof settingDraft === 'string' ? settingDraft : '';
    if (activeSettingKey === 'title' || activeSettingKey === 'category' || activeSettingKey === 'author') {
      const placeholders: Record<'title' | 'category' | 'author', string> = {
        title: '请输入清晰、有辨识度的文章标题',
        category: '例如：通知公告',
        author: '作者姓名',
      };
      return <label className="article-dialog-field"><span>{articleSettingLabels[activeSettingKey]}</span><Input autoFocus size={activeSettingKey === 'title' ? 'large' : 'middle'} value={stringDraft} placeholder={placeholders[activeSettingKey]} onChange={(event) => setSettingDraft(event.target.value)} /></label>;
    }
    if (activeSettingKey === 'summary') {
      return <label className="article-dialog-field"><span>摘要</span><Input.TextArea autoFocus value={stringDraft} rows={5} placeholder="一句话概括文章价值，便于列表展示" onChange={(event) => setSettingDraft(event.target.value)} /></label>;
    }
    if (activeSettingKey === 'status') {
      return <label className="article-dialog-field"><span>状态</span><Select autoFocus value={stringDraft} options={articleStatusOptions.map((status) => ({ value: status, label: status }))} onChange={(status) => setSettingDraft(status)} /></label>;
    }
    if (activeSettingKey === 'contentLocale') {
      return <label className="article-dialog-field"><span>正文语言</span><Select autoFocus value={stringDraft} options={articleLocaleOptions} onChange={(locale) => setSettingDraft(locale)} /></label>;
    }
    if (activeSettingKey === 'visibility') {
      return <div className="privacy-switch-row"><div><strong>仅自己可见</strong><small>开启后仅归属人和管理员可查看，其他登录用户不会在列表中看到此文章。</small></div><Switch autoFocus checked={settingDraft === true} checkedChildren="私密" unCheckedChildren="公开" onChange={(checked) => setSettingDraft(checked)} /></div>;
    }
    return <div className="privacy-switch-row"><div><strong>18R 限制</strong><small>开启后仅登录且开启 18R 内容的用户可见。</small></div><Switch autoFocus checked={settingDraft === true} checkedChildren="开启" unCheckedChildren="关闭" onChange={(checked) => setSettingDraft(checked)} /></div>;
  };

  return <form className="rich-editor-form" onSubmit={handleSubmitArticle}>
    <RichTextEditor value={articleForm.content} onChange={(content) => updateArticleForm({ content })} />
    <section className="article-settings-section" aria-labelledby="article-settings-title">
      <div className="article-settings-heading"><div><strong id="article-settings-title">文章设置</strong><span>点击字段按钮，在 Dialog 中编辑文章信息。</span></div><Tag color="blue">{articleSettingKeys.length} 项设置</Tag></div>
      <div className="article-settings-actions">
        {articleSettingKeys.map((settingKey) => {
          const settingValue = getArticleSettingValue(settingKey);
          return <Button key={settingKey} className="article-setting-button" htmlType="button" aria-label={`${articleSettingLabels[settingKey]}：${settingValue}`} onClick={() => openArticleSettingDialog(settingKey)}>
            <span className="article-setting-copy"><span className="article-setting-label">{articleSettingLabels[settingKey]}</span><span className="article-setting-value" title={settingValue}>{settingValue}</span></span>
            <EditOutlined aria-hidden="true" />
          </Button>;
        })}
      </div>
    </section>
    <div className="rich-editor-actions"><Button htmlType="button" onClick={onCancel}>取消</Button><Button htmlType="submit" type="primary" loading={isSavingArticle} icon={<SaveOutlined />}>{mode === 'edit' ? '保存修改' : '创建文章'}</Button></div>
    <Modal
      open={Boolean(activeSettingKey)}
      title={activeSettingKey ? articleSettingLabels[activeSettingKey] : '文章设置'}
      okText="保存"
      cancelText="取消"
      width="min(560px, calc(100vw - 32px))"
      destroyOnHidden
      onOk={confirmArticleSetting}
      onCancel={closeArticleSettingDialog}
    >
      {settingError && <Alert type="error" showIcon title={settingError} />}
      <div className="article-setting-dialog-content">{renderArticleSettingControl()}</div>
    </Modal>
  </form>;
}

type ArticleCardProps = {
  /** article 表示文章。 */
  article: Article;
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** exportingFormat 表示导出格式。 */
  exportingFormat: ArticleExportFormat | null;
  /** exportDisabled 表示导出操作。 */
  exportDisabled: boolean;
  /** onExport 表示导出回调。 */
  onExport: (article: Article, format: ArticleExportFormat) => Promise<void>;
  /** onPreview 表示预览。 */
  onPreview: (article: Article) => void;
  /** onEdit 表示编辑回调。 */
  onEdit: (article: Article) => void;
  /** onToggle 表示变量 onToggle。 */
  onToggle: (article: Article) => void;
  /** onDelete 表示删除回调。 */
  onDelete: (articleId: number) => void;
};

/** ArticleCard 保存模块使用的固定配置或共享状态。 */
function ArticleCard({ article, actions, exportingFormat, exportDisabled, onExport, onPreview, onEdit, onToggle, onDelete }: ArticleCardProps) {
  /** isPublished 保存发布状态。 */
  const isPublished = article.status === '已发布';
  /** isIndexable 保存可索引状态。 */
  const isIndexable = isPublished && !article.isPrivate;
  /** titleId 保存标题标识。 */
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

/** RichTextEditor 实现对应业务逻辑。 */
function RichTextEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  /** editorRef 保存跨渲染周期使用的编辑器引用。 */
  const editorRef = useRef<HTMLDivElement>(null);
  /** imageInputRef 保存跨渲染周期使用的图片输入值引用。 */
  const imageInputRef = useRef<HTMLInputElement>(null);
  /** videoInputRef 保存跨渲染周期使用的输入值引用。 */
  const videoInputRef = useRef<HTMLInputElement>(null);
  /** selectionRef 保存跨渲染周期使用的文本选区引用。 */
  const selectionRef = useRef<Range | null>(null);
  /** isUploading、setIsUploading 分别保存上传状态状态及其更新函数。 */
  const [isUploading, setIsUploading] = useState(false);
  /** uploadError、setUploadError 分别保存上传错误状态状态及其更新函数。 */
  const [uploadError, setUploadError] = useState('');
  /** externalMedia、setExternalMedia 保存外部媒体、变量 setExternalMedia。 */
  const [externalMedia, setExternalMedia] = useState<{ kind: 'link' | 'image' | 'video'; url: string; description: string } | null>(null);
  /** externalMediaError、setExternalMediaError 分别保存错误状态状态及其更新函数。 */
  const [externalMediaError, setExternalMediaError] = useState('');

  useEffect(() => {
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value || '';
    }
  }, [value]);

  /** sync 负责更新并保存对应业务状态。 */
  const sync = () => onChange(editorRef.current?.innerHTML ?? '');
  /** restoreSelection 负责计算或维护文本选区。 */
  const restoreSelection = () => {
    /** selection 保存文本选区。 */
    const selection = window.getSelection();
    if (!selectionRef.current || !selection) return;
    selection.removeAllRanges();
    selection.addRange(selectionRef.current);
  };
  /** command 负责计算或维护变量 command。 */
  const command = (name: string, commandValue?: string) => {
    editorRef.current?.focus();
    restoreSelection();
    document.execCommand(name, false, commandValue);
    sync();
  };
  /** insertHTML 负责创建或追加对应业务记录。 */
  const insertHTML = (html: string) => {
    editorRef.current?.focus();
    restoreSelection();
    document.execCommand('insertHTML', false, html);
    sync();
  };
  /** openExternalMediaDialog 负责计算或维护对话框。 */
  const openExternalMediaDialog = (kind: 'link' | 'image' | 'video') => {
    /** selection 保存文本选区。 */
    const selection = window.getSelection();
    if (selection?.rangeCount) selectionRef.current = selection.getRangeAt(0).cloneRange();
    setExternalMediaError('');
    setExternalMedia({ kind, url: '', description: kind === 'image' ? '文章配图' : '' });
  };
  /** confirmExternalMedia 负责计算或维护变量 confirmExternalMedia。 */
  const confirmExternalMedia = () => {
    if (!externalMedia) return;
    /** url 保存地址。 */
    const url = normalizeExternalUrl(externalMedia.url);
    if (!url) {
      setExternalMediaError('请输入以 http:// 或 https:// 开头的有效地址。');
      return;
    }
    if (externalMedia.kind === 'link') command('createLink', url);
    if (externalMedia.kind === 'image') {
      /** safeDescription 保存说明。 */
      const safeDescription = escapeHtmlAttribute(externalMedia.description.trim() || '文章配图');
      insertHTML(`<figure class="article-media image-media"><img src="${escapeHtmlAttribute(url)}" alt="${safeDescription}" title="${safeDescription}" loading="lazy" decoding="async" /><figcaption>${safeDescription}</figcaption></figure><p><br /></p>`);
    }
    if (externalMedia.kind === 'video') {
      insertHTML(`<figure class="article-media video-media"><video controls preload="metadata" src="${escapeHtmlAttribute(url)}">当前浏览器不支持视频播放。</video><figcaption>视频</figcaption></figure><p><br /></p>`);
    }
    setExternalMedia(null);
    setExternalMediaError('');
  };
  /** uploadMedia 负责执行对应业务操作。 */
  const uploadMedia = async (event: ChangeEvent<HTMLInputElement>, kind: 'image' | 'video') => {
    /** file 保存文件。 */
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    /** expectedPrefix 保存变量 expectedPrefix。 */
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
      /** formData 保存表单业务数据。 */
      const formData = new FormData();
      formData.set('file', file);
      formData.set('displayName', file.name);
      formData.set('category', kind === 'image' ? '文章图片' : '文章视频');
      formData.set('description', `文章富文本本地${kind === 'image' ? '图片' : '视频'}资源`);
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/files`, { method: 'POST', body: formData });
      if (!response.ok) throw new Error(await readMediaUploadError(response));
      /** uploaded 保存变量 uploaded。 */
      const uploaded = (await response.json()) as { id: number; displayName?: string };
      /** source 保存来源。 */
      const source = `${API_BASE_URL}/api/files/${uploaded.id}/preview`;
      /** label 保存显示标签。 */
      const label = escapeHtmlAttribute(uploaded.displayName || file.name);
      if (kind === 'image') {
        insertHTML(`<figure class="article-media image-media"><img src="${source}" alt="${label}" title="${label}" loading="lazy" decoding="async" /><figcaption>${label}</figcaption></figure><p><br /></p>`);
      } else {
        insertHTML(`<figure class="article-media video-media"><video controls preload="metadata" src="${source}">当前浏览器不支持视频播放。</video><figcaption>${label}</figcaption></figure><p><br /></p>`);
      }
    /** error 保存当前操作结果以及可能返回的错误状态。 */
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

/** readMediaUploadError 加载对应业务数据。 */
async function readMediaUploadError(response: Response) {
  try { const payload = (await response.json()) as { error?: string }; return payload.error || '媒体上传失败。'; } catch { return '媒体上传失败。'; }
}

/** normalizeExternalUrl 实现对应业务逻辑。 */
function normalizeExternalUrl(value: string) {
  /** url 保存地址。 */
  const url = value.trim();
  return /^https?:\/\//i.test(url) ? url : '';
}

/** escapeHtmlAttribute 实现对应业务逻辑。 */
function escapeHtmlAttribute(value: string) {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/** formatUploadSize 转换并生成对应业务结果。 */
function formatUploadSize(size: number) { return size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(0)} MB` : `${Math.ceil(size / 1024)} KB`; }

/** ToolbarButton 定义对应业务的数据结构与调用契约。 */
function ToolbarButton({ label, icon, text, onClick }: { label: string; icon?: ReactNode; text?: string; onClick: () => void }) { return <Tooltip title={label}><Button aria-label={label} type="text" onMouseDown={(event) => event.preventDefault()} onClick={onClick}>{icon ?? text}</Button></Tooltip>; }

/** ArticlePreview 实现对应业务逻辑。 */
function ArticlePreview({ article, onClose }: { article: Article | null; onClose: () => void }) {
  /** safeContent 保存内容。 */
  const safeContent = article ? sanitizeArticleHtml(article.content || '<p>暂无正文内容。</p>') : '';
  /** imageSource、setImageSource 保存图片来源、图片来源。 */
  const [imageSource, setImageSource] = useState<string | null>(null);
  /** isIndexable 保存可索引状态。 */
  const isIndexable = Boolean(article && article.status === '已发布' && !article.isPrivate);
  /** structuredData 保存业务数据。 */
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
  /** handlePreviewContentClick 负责处理对应的界面事件和状态变化。 */
  const handlePreviewContentClick = (event: ReactMouseEvent<HTMLDivElement>) => {
    /** image 保存图片。 */
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

/** ImageZoomPreview 实现对应业务逻辑。 */
function ImageZoomPreview({ source, onClose }: { source: string | null; onClose: () => void }) {
  /** scale、setScale 分别保存缩放比例状态及其更新函数。 */
  const [scale, setScale] = useState(1);
  /** offset、setOffset 分别保存偏移量状态及其更新函数。 */
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  /** dragRef 保存跨渲染周期使用的变量 dragRef引用。 */
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  useEffect(() => { setScale(1); setOffset({ x: 0, y: 0 }); }, [source]);
  // 保留最小缩放下限，确保预览始终可恢复；不设置上限，
  // 因为高分辨率原图可能需要放大数倍才能正常检查。
  const adjustScale = (amount: number) => setScale((current) => Math.max(0.1, Number((current + amount).toFixed(2))));
  /** reset 负责计算或维护变量 reset。 */
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
      /** drag 保存变量 drag。 */
      const drag = dragRef.current;
      if (!drag) return;
      setOffset({ x: drag.originX + event.clientX - drag.x, y: drag.originY + event.clientY - drag.y });
    }} onPointerUp={() => { dragRef.current = null; }} onPointerLeave={() => { dragRef.current = null; }}>
      {source && <img src={source} alt="文章原图预览" draggable={false} onPointerDown={(event) => { dragRef.current = { x: event.clientX, y: event.clientY, originX: offset.x, originY: offset.y }; event.currentTarget.setPointerCapture(event.pointerId); }} style={{ transform: `translate(${offset.x}px, ${offset.y}px) scale(${scale})` }} />}
    </div>
  </Modal>;
}

/** sanitizeArticleHtml 实现对应业务逻辑。 */
function sanitizeArticleHtml(input: string) {
  if (typeof window === 'undefined') return '';
  /** template 保存变量 template。 */
  const template = document.createElement('template');
  template.innerHTML = input;
  /** allowedTags 保存允许范围标签。 */
  const allowedTags = new Set(['A', 'B', 'BR', 'BLOCKQUOTE', 'CODE', 'DIV', 'EM', 'FIGCAPTION', 'FIGURE', 'H1', 'H2', 'H3', 'H4', 'HR', 'I', 'IMG', 'LI', 'OL', 'P', 'PRE', 'S', 'SOURCE', 'SPAN', 'STRONG', 'U', 'UL', 'VIDEO']);
  template.content.querySelectorAll('*').forEach((node) => {
    if (!allowedTags.has(node.tagName)) { node.replaceWith(...Array.from(node.childNodes)); return; }
    Array.from(node.attributes).forEach((attribute) => {
      /** name 保存名称。 */
      const name = attribute.name.toLowerCase();
      /** value 保存值。 */
      const value = attribute.value.trim();
      /** isSafeMediaSource 保存来源。 */
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
