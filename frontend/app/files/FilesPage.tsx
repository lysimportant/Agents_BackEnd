'use client';

import { useEffect, useMemo, useRef, useState, type ChangeEvent, type FormEvent, type MouseEvent, type WheelEvent } from 'react';
import {
  Button,
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
  PictureOutlined,
  HistoryOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
} from '@ant-design/icons';
import { API_BASE_URL, MAX_UPLOAD_SIZE } from '../lib/constants';
import { permanentlyDeleteFile, readTextFileContent, updateFileMetadata, updateTextFileContent } from '../lib/fileApi';
import type { ResourceActionAccess } from '../lib/actionPermissions';
import { RichTextEditor } from '../components/RichTextEditor';
import type { FileForm, ManagedFile } from '../types/admin';

type FilesPageProps = {
  actions: ResourceActionAccess;
  filteredFiles: ManagedFile[];
  recycleFiles: ManagedFile[];
  fileForm: FileForm;
  selectedUploadFile: File | null;
  editingFileId: number | null;
  fileKeyword: string;
  isSavingFile: boolean;
  onFileFormChange: (form: FileForm) => void;
  onSelectUploadFile: (event: ChangeEvent<HTMLInputElement>) => void;
  onSubmitFile: (event: FormEvent<HTMLFormElement>) => Promise<boolean>;
  onResetFileForm: () => void;
  onFileKeywordChange: (keyword: string) => void;
  onEditFile: (file: ManagedFile) => void;
  onDownloadFile: (fileId: number) => void;
  onDeleteFile: (fileId: number) => void;
  onRestoreFile: (fileId: number) => void;
  onLoadRecycleFiles: () => Promise<ManagedFile[]>;
  onRefreshFiles: () => Promise<void>;
};

type FileKind = 'all' | 'image' | 'pdf' | 'word' | 'spreadsheet' | 'presentation' | 'archive' | 'executable' | 'text' | 'other';
type FileKindMeta = { key: FileKind; label: string; icon: string; tone: string; description: string };

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
const CATEGORY_PRESETS = ['制度文档', '图片素材', '合同资料', '报表台账', '安装包', '培训资料', '其它'];

export function FilesPage(props: FilesPageProps) {
  const {
    actions,
    filteredFiles, recycleFiles, fileForm, selectedUploadFile, editingFileId, fileKeyword, isSavingFile,
    onFileFormChange, onSelectUploadFile, onSubmitFile, onResetFileForm, onFileKeywordChange,
    onEditFile, onDownloadFile, onDeleteFile, onRestoreFile, onLoadRecycleFiles, onRefreshFiles,
  } = props;
  const [activeKind, setActiveKind] = useState<FileKind>('all');
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isRecycleOpen, setIsRecycleOpen] = useState(false);
  const [isRecycleLoading, setIsRecycleLoading] = useState(false);
  const [textEditorFile, setTextEditorFile] = useState<ManagedFile | null>(null);
  const [textEditorContent, setTextEditorContent] = useState('');
  const [isTextLoading, setIsTextLoading] = useState(false);
  const [isTextSaving, setIsTextSaving] = useState(false);
  const [deletingPermanentId, setDeletingPermanentId] = useState<number | null>(null);
  const [recycleError, setRecycleError] = useState('');
  const [textEditorError, setTextEditorError] = useState('');
  const [previewFile, setPreviewFile] = useState<ManagedFile | null>(null);
  const [originalLoading, setOriginalLoading] = useState(true);
  const [imageScale, setImageScale] = useState(1);
  const [imageOffset, setImageOffset] = useState({ x: 0, y: 0 });
  const [isDraggingImage, setIsDraggingImage] = useState(false);
  const dragStartRef = useRef({ pointerX: 0, pointerY: 0, offsetX: 0, offsetY: 0 });
  const files = Array.isArray(filteredFiles) ? filteredFiles : [];
  const kindCounts = useMemo(() => {
    const counts = FILE_KIND_OPTIONS.reduce<Record<FileKind, number>>((all, item) => ({ ...all, [item.key]: 0 }), {} as Record<FileKind, number>);
    counts.all = files.length;
    files.forEach((file) => { counts[getFileKind(file).key] += 1; });
    return counts;
  }, [files]);
  const categoryOptions = useMemo(() => Array.from(new Set([...CATEGORY_PRESETS, ...files.map((f) => f.category), fileForm.category].filter(Boolean))), [files, fileForm.category]);
  const visibleFiles = activeKind === 'all' ? files : files.filter((file) => getFileKind(file).key === activeKind);
  const clampScale = (next: number) => Math.max(0.35, Number(next.toFixed(2)));
  const resetImageTransform = () => { setImageScale(1); setImageOffset({ x: 0, y: 0 }); };
  const openImage = (file: ManagedFile) => { setOriginalLoading(true); resetImageTransform(); setPreviewFile(file); };
  const closeUploadDialog = () => { setIsUploadOpen(false); onResetFileForm(); };
  const closeEditDialog = () => { setIsEditOpen(false); onResetFileForm(); };
  const openTextEditor = async (file: ManagedFile) => {
    if (!actions.update) return;
    onEditFile(file);
    setTextEditorFile(file);
    setTextEditorContent('');
    setTextEditorError('');
    setIsTextLoading(true);
    try {
      const content = await readTextFileContent(file.id);
      setTextEditorContent(content);
    } catch (error) {
      setTextEditorError(error instanceof Error ? error.message : '读取文本内容失败');
    } finally {
      setIsTextLoading(false);
    }
  };
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
    } catch (error) {
      setTextEditorError(error instanceof Error ? error.message : '保存文件失败');
    } finally {
      setIsTextSaving(false);
    }
  };
  const permanentlyDeleteFromRecycle = async (fileId: number) => {
    if (!actions.permanentDelete) return;
    setRecycleError('');
    setDeletingPermanentId(fileId);
    try {
      await permanentlyDeleteFile(fileId);
      await onLoadRecycleFiles();
      await onRefreshFiles();
    } catch (error) {
      setRecycleError(error instanceof Error ? error.message : '永久删除文件失败');
    } finally {
      setDeletingPermanentId(null);
    }
  };
  const openEditDialog = (file: ManagedFile) => {
    if (!actions.update) return;
    if (getFileKind(file).key === 'text') {
      void openTextEditor(file);
      return;
    }
    onEditFile(file);
    setIsEditOpen(true);
  };
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
  const onPreviewWheel = (event: WheelEvent<HTMLDivElement>) => {
    event.preventDefault();
    setImageScale((current) => clampScale(current + (event.deltaY < 0 ? 0.12 : -0.12)));
  };
  const startImageDrag = (event: MouseEvent<HTMLDivElement>) => {
    if (imageScale <= 1) return;
    setIsDraggingImage(true);
    dragStartRef.current = { pointerX: event.clientX, pointerY: event.clientY, offsetX: imageOffset.x, offsetY: imageOffset.y };
  };
  const moveImageDrag = (event: MouseEvent<HTMLDivElement>) => {
    if (!isDraggingImage) return;
    setImageOffset({
      x: dragStartRef.current.offsetX + event.clientX - dragStartRef.current.pointerX,
      y: dragStartRef.current.offsetY + event.clientY - dragStartRef.current.pointerY,
    });
  };
  const stopImageDrag = () => setIsDraggingImage(false);

  useEffect(() => {
    if (!editingFileId) setIsEditOpen(false);
  }, [editingFileId]);

  const submitFileForm = async (event: FormEvent<HTMLFormElement>, mode: 'upload' | 'edit') => {
    if (mode === 'upload' ? !actions.create : !actions.update) return;
    if (!(await onSubmitFile(event))) return;
    if (mode === 'upload') setIsUploadOpen(false);
    else setIsEditOpen(false);
  };

  const fileFormContent = (mode: 'upload' | 'edit', asForm = true) => {
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
      <Card data-tilt-disabled="true" className="file-browser-panel" title={<h1 id="files-page-title" className="file-page-heading">文件管理</h1>} extra={<div className="antd-file-tools"><Input value={fileKeyword} allowClear onChange={(event) => onFileKeywordChange(event.target.value)} placeholder="名称、分类或说明" prefix={<FileTextOutlined />} /><Button onClick={() => onFileKeywordChange('')}>重置</Button>{actions.create && <Button type="primary" icon={<InboxOutlined />} onClick={() => { onResetFileForm(); setIsUploadOpen(true); }}>上传文件</Button>}{(actions.restore || actions.permanentDelete) && <Button icon={<DeleteOutlined />} onClick={() => void openRecycleBin()}>回收站{recycleFiles.length ? ` (${recycleFiles.length})` : ''}</Button>}</div>}>
        <div className="file-type-tabs" role="tablist" aria-label="按文件类型筛选">{FILE_KIND_OPTIONS.map((item) => <button className={activeKind === item.key ? 'active' : ''} type="button" role="tab" aria-selected={activeKind === item.key} key={item.key} onClick={() => setActiveKind(item.key)}><span aria-hidden="true">{item.icon}</span>{item.label}<strong>{kindCounts[item.key]}</strong></button>)}</div>
        {visibleFiles.length === 0 ? <Empty description="暂无匹配文件" /> : <div className="file-card-grid">{visibleFiles.map((file) => <FileCard key={file.id} file={file} actions={actions} onOpenImage={openImage} onEditFile={openEditDialog} onDownloadFile={onDownloadFile} onDeleteFile={onDeleteFile} />)}</div>}
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
        {previewFile && <figure className="file-image-zoom-wrap" itemScope={!previewFile.isPrivate} itemType={!previewFile.isPrivate ? 'https://schema.org/ImageObject' : undefined}><div className="file-image-zoom-toolbar"><span>{getImageAccessibleText(previewFile)}；滚轮缩放，放大后可拖拽移动</span><Space><Button icon={<ZoomOutOutlined />} onClick={() => setImageScale((current) => clampScale(current - 0.25))}>缩小</Button><strong>{Math.round(imageScale * 100)}%</strong><Button icon={<ZoomInOutlined />} onClick={() => setImageScale((current) => clampScale(current + 0.25))}>放大</Button><Button icon={<CompressOutlined />} onClick={resetImageTransform}>适配</Button></Space></div><div className={`file-image-zoom-stage ${isDraggingImage ? 'dragging' : ''}`} onWheel={onPreviewWheel} onMouseDown={startImageDrag} onMouseMove={moveImageDrag} onMouseUp={stopImageDrag} onMouseLeave={stopImageDrag}>{originalLoading && <div className="original-image-loading"><LoadingOutlined spin /> 正在加载原图…</div>}<img draggable={false} src={`${API_BASE_URL}/api/files/${previewFile.id}/preview`} alt={getImageAccessibleText(previewFile)} title={previewFile.description || previewFile.displayName} itemProp={!previewFile.isPrivate ? 'contentUrl' : undefined} style={{ transform: `translate(${imageOffset.x}px, ${imageOffset.y}px) scale(${imageScale})` }} onLoad={() => setOriginalLoading(false)} onError={() => setOriginalLoading(false)} /></div><figcaption className="file-seo-caption" itemProp={!previewFile.isPrivate ? 'caption' : undefined}>{getImageAccessibleText(previewFile)}</figcaption></figure>}
      </Modal>
    </section>
  );
}

type FileCardProps = { file: ManagedFile; actions: ResourceActionAccess; onOpenImage: (file: ManagedFile) => void; onEditFile: (file: ManagedFile) => void; onDownloadFile: (fileId: number) => void; onDeleteFile: (fileId: number) => void };
function FileCard({ file, actions, onOpenImage, onEditFile, onDownloadFile, onDeleteFile }: FileCardProps) {
  const meta = getFileKind(file);
  const previewUrl = `${API_BASE_URL}/api/files/${file.id}/preview`;
  const thumbnailUrl = `${API_BASE_URL}/api/files/${file.id}/thumbnail`;
  const isImage = meta.key === 'image';
  const isPDF = meta.key === 'pdf';
  const isIndexableImage = isImage && !file.isPrivate;
  const titleId = `file-title-${file.id}`;
  const imageText = getImageAccessibleText(file);
  return <article className={`file-card tone-${meta.tone}`} aria-labelledby={titleId} itemScope={isIndexableImage} itemType={isIndexableImage ? 'https://schema.org/ImageObject' : undefined}>
    {isIndexableImage && <meta itemProp="contentUrl" content={previewUrl} />}
    <figure className="file-preview-frame">
      {isImage ? <button className="thumbnail-button" type="button" onClick={() => onOpenImage(file)} aria-label={`预览原图：${imageText}`}><img src={thumbnailUrl} alt={imageText} title={file.description || file.displayName} itemProp={isIndexableImage ? 'thumbnailUrl' : undefined} loading="lazy" decoding="async" /><span><EyeOutlined /> 查看原图</span></button> : isPDF ? <a className="file-preview-icon pdf-preview" href={previewUrl} target="_blank" rel="noopener"><FilePdfOutlined /><strong>PDF</strong><small>点击浏览</small></a> : <div className="file-preview-icon">{isImage ? <PictureOutlined /> : <FileImageOutlined />}<span aria-hidden="true">{meta.icon}</span><strong>{meta.label}</strong><small>{getFileExtension(file.originalName).toUpperCase() || meta.description}</small></div>}
      <figcaption className="file-seo-caption" itemProp={isIndexableImage ? 'caption' : undefined}>{isImage ? imageText : `${file.displayName}，${meta.label} 文件`}</figcaption>
    </figure>
    <div className="file-card-body"><div className="file-card-title"><strong id={titleId} title={file.displayName} itemProp={isIndexableImage ? 'name' : undefined}>{file.displayName}</strong><Space size={4} wrap><Tag>{file.category || '未分类'}</Tag><Tag color={file.isPrivate ? 'warning' : 'blue'}>{file.isPrivate ? '私密' : '公开'}</Tag></Space></div><p title={file.originalName}>{file.originalName}</p><small itemProp={isIndexableImage ? 'description' : undefined}>{file.description || '暂无说明'}</small><div className="file-meta-row"><span>归属：{file.ownerName || '未知'}</span><span>{formatFileSize(file.size)}</span><time itemProp={isIndexableImage ? 'dateModified' : undefined} dateTime={file.updatedAt}>{new Date(file.updatedAt).toLocaleString()}</time></div></div>
    <div className="file-card-actions">
      {isImage && <Tooltip title="点击后才加载原始图片"><Button type="link" icon={<EyeOutlined />} onClick={() => onOpenImage(file)}>预览</Button></Tooltip>}
      {isPDF && <a href={previewUrl} target="_blank" rel="noopener"><Button type="link" icon={<EyeOutlined />}>浏览 PDF</Button></a>}
      {actions.update && <Button type="link" icon={<EditOutlined />} onClick={() => onEditFile(file)}>编辑</Button>}<Button type="link" icon={<DownloadOutlined />} onClick={() => onDownloadFile(file.id)}>下载</Button>{actions.delete && <Popconfirm title="确认将该文件移入回收站？可通过恢复接口找回。" okText="移入回收站" cancelText="取消" onConfirm={() => onDeleteFile(file.id)}><Button danger type="link" icon={<DeleteOutlined />}>移入回收站</Button></Popconfirm>}
    </div>
  </article>;
}

function getFileKind(file: ManagedFile) { return getFileKindFromName(file.originalName || file.displayName, file.contentType); }
function getFileKindFromName(filename: string, contentType = ''): FileKindMeta {
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
function getFileExtension(filename: string) { const ext = filename.split('.').pop()?.trim().toLowerCase(); return ext && ext !== filename.toLowerCase() ? ext : ''; }
function formatFileSize(size: number) { if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`; if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`; return `${size} B`; }
function getImageAccessibleText(file: ManagedFile) {
  const description = file.description.trim();
  const category = file.category.trim();
  return description || `${file.displayName}${category ? `，${category}` : ''}`;
}
