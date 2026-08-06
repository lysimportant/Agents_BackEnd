'use client';

import { useEffect, useMemo, useRef, useState, type ChangeEvent, type FormEvent, type MouseEvent, type WheelEvent } from 'react';
import {
  Button,
  App,
  Card,
  Drawer,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tag,
  Tooltip,
} from 'antd';
import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeOutlined,
  FileImageOutlined,
  FilePdfOutlined,
  FileTextOutlined,
  InboxOutlined,
  LoadingOutlined,
  ReloadOutlined,
  PictureOutlined,
  HistoryOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
} from '@ant-design/icons';
import { API_BASE_URL, MAX_UPLOAD_SIZE } from '@/src/config/constants';
import { permanentlyDeleteFile, readFilePreviewBlob, readTextFileContent, updateFileMetadata, updateTextFileContent } from '@/src/services/fileApi';
import type { ResourceActionAccess } from '@/src/utils/actionPermissions';
import { clearStoredLoginBackground, setStoredLoginBackground } from '@/src/utils/loginBackground';
import { RichTextEditor } from '@/src/components/shared/RichTextEditor';
import type { FileForm, ManagedFile } from '@/src/types/admin';

type FilesPageProps = {
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** filteredFiles 表示筛选后。 */
  filteredFiles: ManagedFile[];
  /** recycleFiles 表示文件。 */
  recycleFiles: ManagedFile[];
  /** fileForm 表示文件表单。 */
  fileForm: FileForm;
  /** selectedUploadFile 表示已选择上传文件。 */
  selectedUploadFile: File | null;
  /** editingFileId 表示文件标识。 */
  editingFileId: number | null;
  /** fileKeyword 表示文件搜索关键词。 */
  fileKeyword: string;
  /** isSavingFile 表示文件。 */
  isSavingFile: boolean;
  /** onFileFormChange 表示文件表单。 */
  onFileFormChange: (form: FileForm) => void;
  /** onSelectUploadFile 表示上传文件。 */
  onSelectUploadFile: (event: ChangeEvent<HTMLInputElement>) => void;
  /** onSubmitFile 表示文件。 */
  onSubmitFile: (event: FormEvent<HTMLFormElement>) => Promise<boolean>;
  /** onResetFileForm 表示文件表单。 */
  onResetFileForm: () => void;
  /** onFileKeywordChange 表示文件搜索关键词。 */
  onFileKeywordChange: (keyword: string) => void;
  /** onEditFile 表示文件。 */
  onEditFile: (file: ManagedFile) => void;
  /** onDownloadFile 表示文件。 */
  onDownloadFile: (file: ManagedFile) => void;
  /** onDeleteFile 表示文件。 */
  onDeleteFile: (fileId: number) => void;
  /** onRestoreFile 表示文件。 */
  onRestoreFile: (fileId: number) => void;
  /** onLoadRecycleFiles 表示文件。 */
  onLoadRecycleFiles: () => Promise<ManagedFile[]>;
  /** onRefreshFiles 表示文件。 */
  onRefreshFiles: () => Promise<unknown>;
};

type FileKind = 'all' | 'image' | 'pdf' | 'word' | 'spreadsheet' | 'presentation' | 'archive' | 'executable' | 'text' | 'other';
type FileKindMeta = { key: FileKind; label: string; icon: string; tone: string; description: string };

/** FILE_KIND_OPTIONS 保存模块使用的固定配置或共享状态。 */
const FILE_KIND_OPTIONS: FileKindMeta[] = [
  { key: 'all', label: '全部', icon: '🗂️', tone: 'slate', description: '所有文件' },
  { key: 'image', label: '图片', icon: '🖼️', tone: 'green', description: 'JPG / PNG / GIF / SVG' },
  { key: 'pdf', label: 'PDF', icon: '📕', tone: 'red', description: '浏览器在线阅读' },
  { key: 'word', label: 'Word', icon: '📘', tone: 'blue', description: 'DOC / DOCX / WPS' },
  { key: 'spreadsheet', label: '表格', icon: '📗', tone: 'emerald', description: 'XLS / XLSX / CSV' },
  { key: 'presentation', label: '演示', icon: '📙', tone: 'orange', description: 'PPT / PPTX' },
  { key: 'archive', label: '压缩包', icon: '🗜️', tone: 'amber', description: 'ZIP / RAR / 7Z' },
  { key: 'executable', label: '程序', icon: '⚙️', tone: 'purple', description: 'EXE / MSI / BAT' },
  { key: 'text', label: '文本', icon: '📄', tone: 'cyan', description: 'TXT / MD / JSON' },
  { key: 'other', label: '其它', icon: '📦', tone: 'gray', description: '无法归类的文件' },
];

/** CATEGORY_PRESETS 保存模块使用的固定配置或共享状态。 */
const CATEGORY_PRESETS = ['制度文档', '图片素材', '合同资料', '报表台账', '安装包', '培训资料', '其它'];

/** FilesPage 实现对应业务逻辑。 */
export function FilesPage(props: FilesPageProps) {
  /** message 保存消息。 */
  const { message } = App.useApp();
  const {
    actions,
    filteredFiles, recycleFiles, fileForm, selectedUploadFile, editingFileId, fileKeyword, isSavingFile,
    onFileFormChange, onSelectUploadFile, onSubmitFile, onResetFileForm, onFileKeywordChange,
    onEditFile, onDownloadFile, onDeleteFile, onRestoreFile, onLoadRecycleFiles, onRefreshFiles,
  } = props;
  /** activeKind、setActiveKind 保存当前激活、当前激活。 */
  const [activeKind, setActiveKind] = useState<FileKind>('all');
  /** isUploadOpen、setIsUploadOpen 分别保存上传状态及其更新函数。 */
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  /** isEditOpen、setIsEditOpen 分别保存打开状态状态及其更新函数。 */
  const [isEditOpen, setIsEditOpen] = useState(false);
  /** isRecycleOpen、setIsRecycleOpen 分别保存打开状态状态及其更新函数。 */
  const [isRecycleOpen, setIsRecycleOpen] = useState(false);
  /** isRecycleLoading、setIsRecycleLoading 分别保存加载状态状态及其更新函数。 */
  const [isRecycleLoading, setIsRecycleLoading] = useState(false);
  /** textEditorFile、setTextEditorFile 保存文件、文件。 */
  const [textEditorFile, setTextEditorFile] = useState<ManagedFile | null>(null);
  /** textEditorContent、setTextEditorContent 分别保存内容状态及其更新函数。 */
  const [textEditorContent, setTextEditorContent] = useState('');
  /** isTextLoading、setIsTextLoading 分别保存加载状态状态及其更新函数。 */
  const [isTextLoading, setIsTextLoading] = useState(false);
  /** isTextSaving、setIsTextSaving 分别保存文本内容状态及其更新函数。 */
  const [isTextSaving, setIsTextSaving] = useState(false);
  /** deletingPermanentId、setDeletingPermanentId 保存标识、标识。 */
  const [deletingPermanentId, setDeletingPermanentId] = useState<number | null>(null);
  /** recycleError、setRecycleError 分别保存错误状态状态及其更新函数。 */
  const [recycleError, setRecycleError] = useState('');
  /** textEditorError、setTextEditorError 分别保存错误状态状态及其更新函数。 */
  const [textEditorError, setTextEditorError] = useState('');
  /** previewFile、setPreviewFile 保存预览文件、预览文件。 */
  const [previewFile, setPreviewFile] = useState<ManagedFile | null>(null);
  /** originalLoading、setOriginalLoading 分别保存加载状态状态及其更新函数。 */
  const [originalLoading, setOriginalLoading] = useState(true);
  /** imageScale、setImageScale 分别保存图片状态及其更新函数。 */
  const [imageScale, setImageScale] = useState(1);
  /** imageOffset、setImageOffset 分别保存图片状态及其更新函数。 */
  const [imageOffset, setImageOffset] = useState({ x: 0, y: 0 });
  /** isDraggingImage、setIsDraggingImage 分别保存图片状态及其更新函数。 */
  const [isDraggingImage, setIsDraggingImage] = useState(false);
  /** settingLoginBackgroundKey、setSettingLoginBackgroundKey 保存登录存储键、登录存储键。 */
  const [settingLoginBackgroundKey, setSettingLoginBackgroundKey] = useState<string | null>(null);
  /** isRefreshingFiles、setIsRefreshingFiles 分别保存文件状态及其更新函数。 */
  const [isRefreshingFiles, setIsRefreshingFiles] = useState(false);
  /** dragStartRef 保存跨渲染周期使用的开始位置引用。 */
  const dragStartRef = useRef({ pointerX: 0, pointerY: 0, offsetX: 0, offsetY: 0 });
  /** files 保存文件。 */
  const files = Array.isArray(filteredFiles) ? filteredFiles : [];
  /** kindCounts 缓存计算得到的数量。 */
  const kindCounts = useMemo(() => {
    /** counts 负责计算或维护数量。 */
    const counts = FILE_KIND_OPTIONS.reduce<Record<FileKind, number>>((all, item) => ({ ...all, [item.key]: 0 }), {} as Record<FileKind, number>);
    counts.all = files.length;
    files.forEach((file) => { counts[getFileKind(file).key] += 1; });
    return counts;
  }, [files]);
  /** categoryOptions 缓存计算得到的分类。 */
  const categoryOptions = useMemo(() => Array.from(new Set([...CATEGORY_PRESETS, ...files.map((f) => f.category), fileForm.category].filter(Boolean))), [files, fileForm.category]);
  /** visibleFiles 负责计算或维护可见状态。 */
  const visibleFiles = activeKind === 'all' ? files : files.filter((file) => getFileKind(file).key === activeKind);
  /** clampScale 负责计算或维护缩放比例。 */
  const clampScale = (next: number) => Math.max(0.35, Number(next.toFixed(2)));
  /** resetImageTransform 负责计算或维护图片。 */
  const resetImageTransform = () => { setImageScale(1); setImageOffset({ x: 0, y: 0 }); };
  /** openImage 负责计算或维护图片。 */
  const openImage = (file: ManagedFile) => { setOriginalLoading(true); resetImageTransform(); setPreviewFile(file); };
  /** closeUploadDialog 负责删除或清理对应业务状态。 */
  const closeUploadDialog = () => { setIsUploadOpen(false); onResetFileForm(); };
  /** closeEditDialog 负责删除或清理对应业务状态。 */
  const closeEditDialog = () => { setIsEditOpen(false); onResetFileForm(); };
  /** openTextEditor 负责计算或维护打开状态文本内容编辑器。 */
  const openTextEditor = async (file: ManagedFile) => {
    if (!actions.update || file.readOnly) return;
    onEditFile(file);
    setTextEditorFile(file);
    setTextEditorContent('');
    setTextEditorError('');
    setIsTextLoading(true);
    try {
      /** content 保存内容。 */
      const content = await readTextFileContent(file.id);
      setTextEditorContent(content);
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = error instanceof Error ? error.message : '读取文本内容失败';
      setTextEditorError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setIsTextLoading(false);
    }
  };
  /** saveTextContent 负责更新并保存对应业务状态。 */
  const saveTextContent = async () => {
    if (!actions.update || !textEditorFile) return;
    setTextEditorError('');
    setIsTextSaving(true);
    try {
      await updateFileMetadata(textEditorFile.id, {
        displayName: fileForm.displayName,
        category: fileForm.category,
        description: fileForm.description,
        isPrivate: Boolean(fileForm.isPrivate),
      });
      await updateTextFileContent(textEditorFile.id, textEditorContent);
      await onRefreshFiles();
      onResetFileForm();
      setTextEditorFile(null);
      void message.success('文件内容保存完成');
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = error instanceof Error ? error.message : '保存文件失败';
      setTextEditorError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setIsTextSaving(false);
    }
  };
  /** permanentlyDeleteFromRecycle 负责计算或维护起始时间。 */
  const permanentlyDeleteFromRecycle = async (fileId: number) => {
    if (!actions.permanentDelete) return;
    setRecycleError('');
    setDeletingPermanentId(fileId);
    try {
      await permanentlyDeleteFile(fileId);
      await onLoadRecycleFiles();
      await onRefreshFiles();
      void message.success('文件永久删除完成');
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = error instanceof Error ? error.message : '永久删除文件失败';
      setRecycleError(errorMessage);
      void message.error(errorMessage);
    } finally {
      setDeletingPermanentId(null);
    }
  };
  /** openEditDialog 负责计算或维护对话框。 */
  const openEditDialog = (file: ManagedFile) => {
    if (!actions.update || file.readOnly) return;
    if (getFileKind(file).key === 'text') {
      void openTextEditor(file);
      return;
    }
    onEditFile(file);
    setIsEditOpen(true);
  };
  /** openRecycleBin 负责计算或维护打开状态。 */
  const openRecycleBin = async () => {
    if (!actions.restore && !actions.permanentDelete) return;
    setIsRecycleOpen(true);
    setIsRecycleLoading(true);
    try {
      await onLoadRecycleFiles();
    } finally {
      setIsRecycleLoading(false);
    }
  };
  /** setAsLoginBackground 负责更新并保存对应业务状态。 */
  const setAsLoginBackground = async (file: ManagedFile) => {
    if (getFileKind(file).key !== 'image') {
      void message.warning('只有图片文件可以设置为登录背景');
      return;
    }

    setSettingLoginBackgroundKey(getFileIdentity(file));
    try {
      /** blob 保存二进制内容。 */
      const blob = await readFilePreviewBlob(file);
      /** dataUrl 保存业务数据地址。 */
      const dataUrl = await createLoginBackgroundDataUrl(blob, file.contentType);
      setStoredLoginBackground({
        url: dataUrl,
        name: file.displayName || file.originalName,
        source: 'file-manager',
        mimeType: blob.type || file.contentType,
        size: blob.size || file.size,
      });
      void message.success('已设为登录背景，退出到登录页后会使用这张本地背景');
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      /** errorMessage 保存错误状态消息。 */
      const errorMessage = error instanceof Error ? error.message : '设置登录背景失败';
      void message.error(errorMessage);
    } finally {
      setSettingLoginBackgroundKey(null);
    }
  };
  /** refreshFiles 负责计算或维护文件。 */
  const refreshFiles = async () => {
    setIsRefreshingFiles(true);
    try {
      await onRefreshFiles();
      void message.success('文件列表已刷新');
    /** error 保存当前操作结果以及可能返回的错误状态。 */
    } catch (error) {
      void message.error(error instanceof Error ? error.message : '刷新文件列表失败');
    } finally {
      setIsRefreshingFiles(false);
    }
  };
  /** resetLoginBackground 负责计算或维护登录。 */
  const resetLoginBackground = () => {
    clearStoredLoginBackground();
    void message.success('已恢复默认登录背景');
  };
  /** onPreviewWheel 负责处理对应的界面事件和状态变化。 */
  const onPreviewWheel = (event: WheelEvent<HTMLDivElement>) => {
    event.preventDefault();
    setImageScale((current) => clampScale(current + (event.deltaY < 0 ? 0.12 : -0.12)));
  };
  /** startImageDrag 负责计算或维护图片。 */
  const startImageDrag = (event: MouseEvent<HTMLDivElement>) => {
    if (imageScale <= 1) return;
    setIsDraggingImage(true);
    dragStartRef.current = { pointerX: event.clientX, pointerY: event.clientY, offsetX: imageOffset.x, offsetY: imageOffset.y };
  };
  /** moveImageDrag 负责计算或维护图片。 */
  const moveImageDrag = (event: MouseEvent<HTMLDivElement>) => {
    if (!isDraggingImage) return;
    setImageOffset({
      x: dragStartRef.current.offsetX + event.clientX - dragStartRef.current.pointerX,
      y: dragStartRef.current.offsetY + event.clientY - dragStartRef.current.pointerY,
    });
  };
  /** stopImageDrag 负责计算或维护图片。 */
  const stopImageDrag = () => setIsDraggingImage(false);

  useEffect(() => {
    if (!editingFileId) setIsEditOpen(false);
  }, [editingFileId]);

  /** submitFileForm 负责执行对应业务操作。 */
  const submitFileForm = async (event: FormEvent<HTMLFormElement>, mode: 'upload' | 'edit') => {
    if (mode === 'upload' ? !actions.create : !actions.update) return;
    if (!(await onSubmitFile(event))) return;
    if (mode === 'upload') setIsUploadOpen(false);
    else setIsEditOpen(false);
  };

  /** fileFormContent 负责计算或维护文件表单内容。 */
  const fileFormContent = (mode: 'upload' | 'edit', asForm = true) => {
    /** fields 保存表单字段。 */
    const fields = <>
      {mode === 'upload' && <label className="file-dropzone antd-dropzone"><input required type="file" onChange={onSelectUploadFile} /><InboxOutlined /><strong>{selectedUploadFile?.name ?? '点击选择文件上传'}</strong><small>{selectedUploadFile ? `${formatFileSize(selectedUploadFile.size)} · ${getFileKindFromName(selectedUploadFile.name, selectedUploadFile.type).label}` : `图片、PDF、Office、程序等，单文件最大 ${formatFileSize(MAX_UPLOAD_SIZE)}`}</small></label>}
      <label>显示名称<Input required value={fileForm.displayName} onChange={(event) => onFileFormChange({ ...fileForm, displayName: event.target.value })} placeholder="请输入文件显示名称" /></label>
      <label>业务分类<Select value={fileForm.category || undefined} allowClear placeholder="请选择或清空分类" options={categoryOptions.map((category) => ({ value: category, label: category }))} onChange={(category) => onFileFormChange({ ...fileForm, category: category ?? '' })} /></label>
      <label>说明<Input.TextArea value={fileForm.description} rows={3} onChange={(event) => onFileFormChange({ ...fileForm, description: event.target.value })} placeholder="请输入文件说明" /></label>
      <div className="privacy-switch-row">
        <div>
          <strong>仅自己可见</strong>
          <small>开启后仅归属人和管理员可查看与操作。</small>
        </div>
        <Switch checked={Boolean(fileForm.isPrivate)} onChange={(checked) => onFileFormChange({ ...fileForm, isPrivate: checked })} checkedChildren="私密" unCheckedChildren="公开" />
      </div>
    </>;
    return asForm ? <form className="antd-file-form" id={`file-${mode}-form`} onSubmit={(event) => void submitFileForm(event, mode)}>{fields}</form> : <div className="antd-file-form">{fields}</div>;
  };

  return (
    <section className="page-stack files-workspace antd-files-workspace" aria-labelledby="files-page-title">
      <Card data-tilt-disabled="true" className="file-browser-panel" title={<h1 id="files-page-title" className="file-page-heading">文件管理</h1>} extra={<div className="antd-file-tools"><Input value={fileKeyword} allowClear onChange={(event) => onFileKeywordChange(event.target.value)} placeholder="名称、分类或说明" prefix={<FileTextOutlined />} /><Button onClick={() => onFileKeywordChange('')}>重置</Button><Button icon={<ReloadOutlined />} loading={isRefreshingFiles} onClick={() => void refreshFiles()}>刷新</Button>{actions.create && <Button type="primary" icon={<InboxOutlined />} onClick={() => { onResetFileForm(); setIsUploadOpen(true); }}>上传文件</Button>}{(actions.restore || actions.permanentDelete) && <Button icon={<DeleteOutlined />} onClick={() => void openRecycleBin()}>回收站{recycleFiles.length ? ` (${recycleFiles.length})` : ''}</Button>}</div>}>
        <div className="file-type-tabs" role="tablist" aria-label="按文件类型筛选">{FILE_KIND_OPTIONS.map((item) => <button className={activeKind === item.key ? 'active' : ''} type="button" role="tab" aria-selected={activeKind === item.key} key={item.key} onClick={() => setActiveKind(item.key)}><span aria-hidden="true">{item.icon}</span>{item.label}<strong>{kindCounts[item.key]}</strong></button>)}</div>
        <div className="file-login-background-toolbar"><span>图片可保存为当前浏览器的登录背景，不依赖外部 URL。</span><Button icon={<PictureOutlined />} onClick={resetLoginBackground}>恢复默认背景</Button></div>
        {visibleFiles.length === 0 ? <Empty description="暂无匹配文件" /> : <div className="file-card-grid">{visibleFiles.map((file) => <FileCard key={getFileIdentity(file)} file={file} actions={actions} onOpenImage={openImage} onEditFile={openEditDialog} onDownloadFile={onDownloadFile} onDeleteFile={onDeleteFile} onSetLoginBackground={setAsLoginBackground} settingLoginBackgroundKey={settingLoginBackgroundKey} />)}</div>}
      </Card>

      <Modal open={isUploadOpen} title="上传文件" okText="上传" cancelText="取消" confirmLoading={isSavingFile} onOk={() => document.getElementById('file-upload-form')?.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }))} onCancel={closeUploadDialog} destroyOnHidden>
        {fileFormContent('upload')}
      </Modal>

      <Modal open={isEditOpen} title="编辑文件信息" okText="保存" cancelText="取消" confirmLoading={isSavingFile} onOk={() => document.getElementById('file-edit-form')?.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }))} onCancel={closeEditDialog} destroyOnHidden>
        {fileFormContent('edit')}
      </Modal>

      <Drawer title="文件回收站" open={isRecycleOpen} size={620} onClose={() => { setIsRecycleOpen(false); setRecycleError(''); }} extra={<Button loading={isRecycleLoading} onClick={() => void openRecycleBin()}>刷新</Button>}>
        <div className="recycle-bin-notice">文件移入回收站后不会自动过期删除；需要你在这里二次确认后点击“永久删除”。永久删除会同时删除数据库记录和磁盘文件，无法恢复。</div>
        {recycleError && <p className="error-message">{recycleError}</p>}
        {isRecycleLoading ? <div className="recycle-loading"><LoadingOutlined spin /> 正在加载回收站…</div> : recycleFiles.length === 0 ? <Empty description="回收站为空" /> : <div className="recycle-file-list">{recycleFiles.map((file) => <article className="recycle-file-card" key={file.id}><div className="recycle-file-main"><strong>{file.displayName}</strong><span>{file.originalName}</span><small>移入时间：{file.deletedAt ? new Date(file.deletedAt).toLocaleString() : '未知'}</small></div><Space>{actions.restore && <Button type="primary" icon={<HistoryOutlined />} onClick={() => onRestoreFile(file.id)}>恢复</Button>}{actions.permanentDelete && <Popconfirm title="确认永久删除该文件？" description="这会删除数据库记录和磁盘文件，无法恢复。" okText="永久删除" okButtonProps={{ danger: true, loading: deletingPermanentId === file.id }} cancelText="取消" onConfirm={() => permanentlyDeleteFromRecycle(file.id)}><Button danger icon={<DeleteOutlined />} loading={deletingPermanentId === file.id}>永久删除</Button></Popconfirm>}</Space></article>)}</div>}
      </Drawer>

      <Modal open={Boolean(textEditorFile)} title={`编辑文本文件：${textEditorFile?.displayName ?? ''}`} okText="保存全部" cancelText="关闭" width="min(1040px, 96vw)" confirmLoading={isTextSaving || isSavingFile} onOk={() => void saveTextContent()} onCancel={() => { setTextEditorFile(null); onResetFileForm(); }} destroyOnHidden>
        {textEditorError && <p className="error-message">{textEditorError}</p>}
        <div className="text-file-edit-panel">
          <section className="text-file-meta-card">
            <h3>文件信息</h3>
            {fileFormContent('edit', false)}
          </section>
          <section className="text-file-content-card">
            <h3>文本内容</h3>
            {isTextLoading ? <div className="recycle-loading"><LoadingOutlined spin /> 正在读取文本内容…</div> : <RichTextEditor value={textEditorContent} onChange={setTextEditorContent} minHeight={360} placeholder="编辑文本、Markdown 或 HTML 内容…" />}
          </section>
        </div>
      </Modal>

      <Modal className="file-image-zoom-modal" open={Boolean(previewFile)} title={previewFile?.displayName} footer={null} width="min(1500px, 98vw)" centered styles={{ body: { padding: 0 } }} onCancel={() => setPreviewFile(null)} destroyOnHidden>
        {previewFile && <figure className="file-image-zoom-wrap" itemScope={!previewFile.isPrivate} itemType={!previewFile.isPrivate ? 'https://schema.org/ImageObject' : undefined}><div className="file-image-zoom-toolbar"><span>{getImageAccessibleText(previewFile)}；滚轮缩放，放大后可拖拽移动</span><Space><Button icon={<ZoomOutOutlined />} onClick={() => setImageScale((current) => clampScale(current - 0.25))}>缩小</Button><strong>{Math.round(imageScale * 100)}%</strong><Button icon={<ZoomInOutlined />} onClick={() => setImageScale((current) => clampScale(current + 0.25))}>放大</Button><Button icon={<CompressOutlined />} onClick={resetImageTransform}>适配</Button></Space></div><div className={`file-image-zoom-stage ${isDraggingImage ? 'dragging' : ''}`} onWheel={onPreviewWheel} onMouseDown={startImageDrag} onMouseMove={moveImageDrag} onMouseUp={stopImageDrag} onMouseLeave={stopImageDrag}>{originalLoading && <div className="original-image-loading"><LoadingOutlined spin /> 正在加载原图…</div>}<img draggable={false} src={resolveFilePreviewUrl(previewFile)} alt={getImageAccessibleText(previewFile)} title={previewFile.description || previewFile.displayName} itemProp={!previewFile.isPrivate ? 'contentUrl' : undefined} style={{ transform: `translate(${imageOffset.x}px, ${imageOffset.y}px) scale(${imageScale})` }} onLoad={() => setOriginalLoading(false)} onError={() => setOriginalLoading(false)} /></div><figcaption className="file-seo-caption" itemProp={!previewFile.isPrivate ? 'caption' : undefined}>{getImageAccessibleText(previewFile)}</figcaption></figure>}
      </Modal>
    </section>
  );
}

type FileCardProps = {
  /** file 表示文件。 */
  file: ManagedFile;
  /** actions 表示操作权限。 */
  actions: ResourceActionAccess;
  /** onOpenImage 表示图片。 */
  onOpenImage: (file: ManagedFile) => void;
  /** onEditFile 表示文件。 */
  onEditFile: (file: ManagedFile) => void;
  /** onDownloadFile 表示文件。 */
  onDownloadFile: (file: ManagedFile) => void;
  /** onDeleteFile 表示文件。 */
  onDeleteFile: (fileId: number) => void;
  /** onSetLoginBackground 表示登录。 */
  onSetLoginBackground: (file: ManagedFile) => void;
  /** settingLoginBackgroundKey 表示登录存储键。 */
  settingLoginBackgroundKey: string | null;
};

/** FileCard 保存模块使用的固定配置或共享状态。 */
function FileCard({ file, actions, onOpenImage, onEditFile, onDownloadFile, onDeleteFile, onSetLoginBackground, settingLoginBackgroundKey }: FileCardProps) {
  /** meta 保存元数据。 */
  const meta = getFileKind(file);
  /** previewUrl 保存预览地址。 */
  const previewUrl = resolveFilePreviewUrl(file);
  /** thumbnailUrl 保存地址。 */
  const thumbnailUrl = file.previewUrl ? previewUrl : `${API_BASE_URL}/api/files/${file.id}/thumbnail`;
  /** isImage 保存图片。 */
  const isImage = meta.key === 'image';
  /** isPDF 保存变量 isPDF。 */
  const isPDF = meta.key === 'pdf';
  /** isIndexableImage 保存图片。 */
  const isIndexableImage = isImage && !file.isPrivate && !file.readOnly;
  /** isSettingLoginBackground 保存登录。 */
  const isSettingLoginBackground = settingLoginBackgroundKey === getFileIdentity(file);
  /** titleId 保存标题标识。 */
  const titleId = `file-title-${file.source || 'managed'}-${file.id}`;
  /** imageText 保存图片。 */
  const imageText = getImageAccessibleText(file);
  return <article className={`file-card tone-${meta.tone}`} aria-labelledby={titleId} itemScope={isIndexableImage} itemType={isIndexableImage ? 'https://schema.org/ImageObject' : undefined}>
    {isIndexableImage && <meta itemProp="contentUrl" content={previewUrl} />}
    <figure className="file-preview-frame">
      {isImage ? <button className="thumbnail-button" type="button" onClick={() => onOpenImage(file)} aria-label={`预览原图：${imageText}`}><img src={thumbnailUrl} alt={imageText} title={file.description || file.displayName} itemProp={isIndexableImage ? 'thumbnailUrl' : undefined} loading="lazy" decoding="async" /><span><EyeOutlined /> 查看原图</span></button> : isPDF ? <a className="file-preview-icon pdf-preview" href={previewUrl} target="_blank" rel="noopener"><FilePdfOutlined /><strong>PDF</strong><small>点击浏览</small></a> : <div className="file-preview-icon">{isImage ? <PictureOutlined /> : <FileImageOutlined />}<span aria-hidden="true">{meta.icon}</span><strong>{meta.label}</strong><small>{getFileExtension(file.originalName).toUpperCase() || meta.description}</small></div>}
      <figcaption className="file-seo-caption" itemProp={isIndexableImage ? 'caption' : undefined}>{isImage ? imageText : `${file.displayName}，${meta.label} 文件`}</figcaption>
    </figure>
    <div className="file-card-body"><div className="file-card-title"><strong id={titleId} title={file.displayName} itemProp={isIndexableImage ? 'name' : undefined}>{file.displayName}</strong><Space size={4} wrap><Tag>{file.category || '未分类'}</Tag>{file.readOnly ? <><Tag color="gold">只读</Tag><Tag color="purple">{file.source === 'internal-chat' ? '内部聊天' : '客服聊天'}</Tag></> : <Tag color={file.isPrivate ? 'warning' : 'blue'}>{file.isPrivate ? '私密' : '公开'}</Tag>}</Space></div><p title={file.originalName}>{file.originalName}</p><small itemProp={isIndexableImage ? 'description' : undefined}>{file.description || '暂无说明'}</small><div className="file-meta-row"><span>归属：{file.ownerName || '未知'}</span><span>{formatFileSize(file.size)}</span><time itemProp={isIndexableImage ? 'dateModified' : undefined} dateTime={file.updatedAt}>{new Date(file.updatedAt).toLocaleString()}</time></div></div>
    <div className="file-card-actions">
      {isImage && <Tooltip title="点击后才加载原始图片"><Button type="link" icon={<EyeOutlined />} onClick={() => onOpenImage(file)}>预览</Button></Tooltip>}
      {isImage && <Button type="link" icon={<PictureOutlined />} loading={isSettingLoginBackground} onClick={() => onSetLoginBackground(file)}>设为登录背景</Button>}
      {isPDF && <a href={previewUrl} target="_blank" rel="noopener"><Button type="link" icon={<EyeOutlined />}>浏览 PDF</Button></a>}
      {actions.update && !file.readOnly && <Button type="link" icon={<EditOutlined />} onClick={() => onEditFile(file)}>编辑</Button>}<Button type="link" icon={<DownloadOutlined />} onClick={() => onDownloadFile(file)}>下载</Button>{actions.delete && !file.readOnly && <Popconfirm title="确认将该文件移入回收站？可通过恢复接口找回。" okText="移入回收站" cancelText="取消" onConfirm={() => onDeleteFile(file.id)}><Button danger type="link" icon={<DeleteOutlined />}>移入回收站</Button></Popconfirm>}
    </div>
  </article>;
}

/** getFileKind 获取对应业务记录。 */
function getFileKind(file: ManagedFile) { return getFileKindFromName(file.originalName || file.displayName, file.contentType); }

/** getFileIdentity 获取对应业务记录。 */
function getFileIdentity(file: ManagedFile) { return `${file.source || 'managed'}:${file.id}`; }

/** resolveFilePreviewUrl 转换并生成对应业务结果。 */
function resolveFilePreviewUrl(file: ManagedFile) { return `${API_BASE_URL}${file.previewUrl || `/api/files/${file.id}/preview`}`; }

/** getFileKindFromName 获取对应业务记录。 */
function getFileKindFromName(filename: string, contentType = ''): FileKindMeta {
  /** ext 保存文件扩展名。 */
  const ext = getFileExtension(filename); const mime = contentType.toLowerCase();
  if (mime.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'image')!;
  if (mime.includes('pdf') || ext === 'pdf') return FILE_KIND_OPTIONS.find((item) => item.key === 'pdf')!;
  if (mime.includes('word') || ['doc', 'docx', 'wps', 'rtf'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'word')!;
  if (mime.includes('sheet') || mime.includes('excel') || ['xls', 'xlsx', 'csv', 'ods'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'spreadsheet')!;
  if (mime.includes('presentation') || mime.includes('powerpoint') || ['ppt', 'pptx', 'odp'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'presentation')!;
  if (mime.includes('zip') || ['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'archive')!;
  if (['exe', 'msi', 'bat', 'cmd', 'apk', 'dmg'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'executable')!;
  if (mime.startsWith('text/') || ['txt', 'md', 'json', 'xml', 'log'].includes(ext)) return FILE_KIND_OPTIONS.find((item) => item.key === 'text')!;
  return FILE_KIND_OPTIONS.find((item) => item.key === 'other')!;
}

/** getFileExtension 保存模块使用的固定配置或共享状态。 */
function getFileExtension(filename: string) { const ext = filename.split('.').pop()?.trim().toLowerCase(); return ext && ext !== filename.toLowerCase() ? ext : ''; }

/** formatFileSize 转换并生成对应业务结果。 */
function formatFileSize(size: number) { if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`; if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`; return `${size} B`; }

/** getImageAccessibleText 获取对应业务记录。 */
function getImageAccessibleText(file: ManagedFile) {
  /** description 保存说明。 */
  const description = file.description.trim();
  /** category 保存分类。 */
  const category = file.category.trim();
  return description || `${file.displayName}${category ? `，${category}` : ''}`;
}

/** LOGIN_BACKGROUND_MAX_EDGE 保存模块使用的固定配置或共享状态。 */
const LOGIN_BACKGROUND_MAX_EDGE = 1920;

/** LOGIN_BACKGROUND_WEBP_QUALITY 保存模块使用的固定配置或共享状态。 */
const LOGIN_BACKGROUND_WEBP_QUALITY = 0.88;

/** LOGIN_BACKGROUND_MAX_DATA_URL_LENGTH 保存模块使用的固定配置或共享状态。 */
const LOGIN_BACKGROUND_MAX_DATA_URL_LENGTH = 4_700_000;

/** createLoginBackgroundDataUrl 创建或追加对应业务记录。 */
async function createLoginBackgroundDataUrl(blob: Blob, fallbackContentType = '') {
  /** mimeType 保存媒体类型类型。 */
  const mimeType = (blob.type || fallbackContentType).toLowerCase();
  if (mimeType && !mimeType.startsWith('image/')) {
    throw new Error('只有图片文件可以设置为登录背景');
  }

  let dataUrl: string;
  try {
    dataUrl = await rasterizeImageBlob(blob);
  /** error 保存当前操作结果以及可能返回的错误状态。 */
  } catch (error) {
    if (mimeType.includes('svg')) {
      throw new Error('SVG 图片无法安全转换为登录背景，请换 JPG、PNG 或 WebP 图片');
    }
    dataUrl = await readBlobAsDataUrl(blob);
  }

  if (dataUrl.length > LOGIN_BACKGROUND_MAX_DATA_URL_LENGTH) {
    throw new Error('图片压缩后仍然过大，请换一张更小的图片再设置登录背景');
  }

  return dataUrl;
}

/** rasterizeImageBlob 实现对应业务逻辑。 */
async function rasterizeImageBlob(blob: Blob) {
  /** objectUrl 保存地址。 */
  const objectUrl = URL.createObjectURL(blob);
  /** image 保存图片。 */
  const image = new Image();

  try {
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error('图片加载失败'));
      image.src = objectUrl;
    });

    /** width 保存宽度。 */
    const width = image.naturalWidth || image.width;
    /** height 保存高度。 */
    const height = image.naturalHeight || image.height;
    if (!width || !height) {
      throw new Error('图片尺寸无效');
    }

    /** scale 保存缩放比例。 */
    const scale = Math.min(1, LOGIN_BACKGROUND_MAX_EDGE / Math.max(width, height));
    /** targetWidth 保存目标宽度。 */
    const targetWidth = Math.max(1, Math.round(width * scale));
    /** targetHeight 保存目标。 */
    const targetHeight = Math.max(1, Math.round(height * scale));
    /** canvas 保存绘图画布。 */
    const canvas = document.createElement('canvas');
    canvas.width = targetWidth;
    canvas.height = targetHeight;
    /** context 保存上下文。 */
    const context = canvas.getContext('2d');
    if (!context) {
      throw new Error('浏览器不支持图片压缩');
    }

    context.imageSmoothingEnabled = true;
    context.imageSmoothingQuality = 'high';
    context.drawImage(image, 0, 0, targetWidth, targetHeight);
    /** dataUrl 保存业务数据地址。 */
    const dataUrl = canvas.toDataURL('image/webp', LOGIN_BACKGROUND_WEBP_QUALITY);
    if (!dataUrl || dataUrl === 'data:,') {
      throw new Error('图片压缩失败');
    }
    return dataUrl;
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}

/** readBlobAsDataUrl 加载对应业务数据。 */
function readBlobAsDataUrl(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    /** reader 保存内容读取器。 */
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result);
        return;
      }
      reject(new Error('图片读取失败'));
    };
    reader.onerror = () => reject(new Error('图片读取失败'));
    reader.readAsDataURL(blob);
  });
}
