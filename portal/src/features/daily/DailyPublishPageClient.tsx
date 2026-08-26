'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import Image from 'next/image';
import {
  ArrowLeft,
  Bold,
  Code2,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Plus,
  Quote,
  RemoveFormatting,
  Strikethrough,
  Underline,
  Image as ImageIcon,
  X,
  Check,
} from 'lucide-react';
import { Link, useRouter } from '@/i18n/navigation';
import { useAuth } from '@/features/auth/AuthProvider';
import { createDaily, listImages, uploadDailyMedia } from '@/services/publicApi';
import { resolveMediaUrl } from '@/config/constants';
import type { PublicFileListItem } from '@/types/publicContent';

/** 日常正文允许的用户可见字符上限，服务端会使用同一上限再次校验。 */
const DAILY_MAX_LENGTH = 2000;

/** 统计编辑器 HTML 中用户可见的 Unicode 字符数量。 */
function countVisibleCharacters(editorHTML: string): number {
  if (typeof document === 'undefined') {
    return 0;
  }
  const temporaryContainer = document.createElement('div');
  temporaryContainer.innerHTML = editorHTML;
  return Array.from(temporaryContainer.textContent ?? '').length;
}

/** 日常发布页提供富文本编辑、可见范围选择和发布确认。 */
export function DailyPublishPageClient() {
  const t = useTranslations('daily');
  const common = useTranslations('common');
  const router = useRouter();
  const { isLoggedIn } = useAuth();
  const editorRef = useRef<HTMLDivElement>(null);
  const publishButtonRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const dialogInitialFocusRef = useRef<HTMLInputElement>(null);
  const linkDialogRef = useRef<HTMLDivElement>(null);
  const linkInputRef = useRef<HTMLInputElement>(null);
  const selectionRef = useRef<Range | null>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const videoInputRef = useRef<HTMLInputElement>(null);
  const [editorHTML, setEditorHTML] = useState('');
  const [visibleCharacterCount, setVisibleCharacterCount] = useState(0);
  const [isPrivate, setIsPrivate] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [linkURL, setLinkURL] = useState('');
  const [coverPickerOpen, setCoverPickerOpen] = useState(false);
  const [coverFiles, setCoverFiles] = useState<PublicFileListItem[]>([]);
  const [selectedCoverID, setSelectedCoverID] = useState<number | null>(null);
  const [isCoverLoading, setIsCoverLoading] = useState(false);
  const [isMediaUploading, setIsMediaUploading] = useState(false);

  /** 同步编辑器内容和可见字符数量，避免提交前才发现超出限制。 */
  const updateEditorState = () => {
    const nextHTML = editorRef.current?.innerHTML ?? '';
    setEditorHTML(nextHTML);
    setVisibleCharacterCount(countVisibleCharacters(nextHTML));
  };

  /** 在当前选区执行浏览器原生富文本命令，并同步编辑器状态。 */
  const executeEditorCommand = (command: string, value?: string) => {
    editorRef.current?.focus();
    document.execCommand(command, false, value);
    updateEditorState();
  };

  /** 打开自定义链接对话框并保存当前选区，最终地址仍由后端正文白名单再次校验。 */
  const insertLink = () => {
    const selection = window.getSelection();
    selectionRef.current = selection && selection.rangeCount > 0 ? selection.getRangeAt(0).cloneRange() : null;
    setLinkURL('');
    setLinkDialogOpen(true);
  };

  /** 恢复编辑器选区并插入链接，拒绝危险协议与空地址。 */
  const confirmLink = () => {
    const normalizedURL = linkURL.trim();
    if (!/^(https?:|mailto:)/i.test(normalizedURL)) {
      return;
    }
    editorRef.current?.focus();
    const selection = window.getSelection();
    if (selection && selectionRef.current) {
      selection.removeAllRanges();
      selection.addRange(selectionRef.current);
    }
    executeEditorCommand('createLink', normalizedURL);
    setLinkDialogOpen(false);
  };

  /** 将上传后的公开媒体插入编辑器，正文提交时会归一化为相对公开 API 地址。 */
  const insertUploadedMedia = async (file: File) => {
    if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) {
      setError(t('mediaTypeInvalid'));
      return;
    }
    setIsMediaUploading(true);
    setError('');
    try {
      const uploadedFile = await uploadDailyMedia(file);
      const mediaURL = resolveMediaUrl(uploadedFile.previewUrl);
      const mediaHTML = uploadedFile.contentType.startsWith('video/')
        ? `<video src="${mediaURL}" controls preload="metadata"></video>`
        : `<img src="${mediaURL}" alt="${file.name.replace(/[&<>\"]+/g, '')}" />`;
      editorRef.current?.focus();
      document.execCommand('insertHTML', false, mediaHTML);
      updateEditorState();
    } catch {
      setError(t('mediaUploadFailed'));
    } finally {
      setIsMediaUploading(false);
    }
  };

  /** 处理工具栏中的本地媒体选择，并允许再次选择同一个文件。 */
  const handleMediaInputChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0];
    event.target.value = '';
    if (selectedFile) {
      await insertUploadedMedia(selectedFile);
    }
  };

  /** 处理拖入编辑器的本地图片或视频。 */
  const handleMediaDrop = (event: React.DragEvent<HTMLDivElement>) => {
    const selectedFile = event.dataTransfer.files[0];
    if (!selectedFile) {
      return;
    }
    event.preventDefault();
    void insertUploadedMedia(selectedFile);
  };

  /** 处理从剪贴板粘贴的本地媒体文件；普通文本仍使用浏览器默认粘贴行为。 */
  const handleMediaPaste = (event: React.ClipboardEvent<HTMLDivElement>) => {
    const selectedFile = event.clipboardData.files[0];
    if (!selectedFile) {
      return;
    }
    if (!selectedFile.type.startsWith('image/') && !selectedFile.type.startsWith('video/')) {
      return;
    }
    event.preventDefault();
    void insertUploadedMedia(selectedFile);
  };

  /** 将浏览器显示用的后端绝对媒体 URL 归一化为公开 API 相对地址。 */
  const normalizeDailyMediaHTML = (rawHTML: string): string => {
    if (typeof document === 'undefined') {
      return rawHTML;
    }
    const temporaryContainer = document.createElement('div');
    temporaryContainer.innerHTML = rawHTML;
    const backendBaseURL = resolveMediaUrl('/');
    temporaryContainer.querySelectorAll<HTMLImageElement | HTMLVideoElement | HTMLSourceElement>('img[src], video[src], source[src]').forEach((mediaElement) => {
      const sourceURL = mediaElement.getAttribute('src') ?? '';
      if (!sourceURL.startsWith(backendBaseURL)) {
        return;
      }
      try {
        const parsedURL = new URL(sourceURL);
        if (parsedURL.pathname.startsWith('/api/public/files/')) {
          mediaElement.setAttribute('src', parsedURL.pathname);
        }
      } catch {
        // 无法解析的地址交给后端白名单清洗移除。
      }
    });
    return temporaryContainer.innerHTML;
  };

  /** 加载公开图片供封面选择，不提供本地文件输入。 */
  const openCoverPicker = async () => {
    setCoverPickerOpen(true);
    if (coverFiles.length > 0) {
      return;
    }
    setIsCoverLoading(true);
    try {
      const response = await listImages({ pageSize: 50 }, { credentials: 'include' });
      setCoverFiles(response.items);
    } catch {
      setError(t('coverLoadFailed'));
    } finally {
      setIsCoverLoading(false);
    }
  };

  // 确认框打开时把焦点移入弹窗，支持 Escape 和 Tab 循环，关闭后归还到发布按钮。
  useEffect(() => {
    if (!dialogOpen) {
      return;
    }
    const publishButton = publishButtonRef.current;
    dialogInitialFocusRef.current?.focus();
    const handleDialogKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isSubmitting) {
        setDialogOpen(false);
        return;
      }
      if (event.key !== 'Tab') {
        return;
      }
      const focusableElements = dialogRef.current?.querySelectorAll<HTMLElement>(
        'button, input, a[href], [tabindex]:not([tabindex="-1"])',
      );
      if (!focusableElements?.length) {
        return;
      }
      const firstFocusableElement = focusableElements[0];
      const lastFocusableElement = focusableElements[focusableElements.length - 1];
      if (event.shiftKey && document.activeElement === firstFocusableElement) {
        event.preventDefault();
        lastFocusableElement.focus();
      } else if (!event.shiftKey && document.activeElement === lastFocusableElement) {
        event.preventDefault();
        firstFocusableElement.focus();
      }
    };
    document.addEventListener('keydown', handleDialogKeyDown);
    return () => {
      document.removeEventListener('keydown', handleDialogKeyDown);
      publishButton?.focus();
    };
  }, [dialogOpen, isSubmitting]);

  /** 校验登录态和可见字符数量后打开发布确认框。 */
  const openPublishDialog = () => {
    setError('');
    if (!isLoggedIn) {
      setError(t('loginRequired'));
      return;
    }
    if (visibleCharacterCount === 0 && !/<(img|video)\b/i.test(editorHTML)) {
      setError(t('contentRequired'));
      return;
    }
    if (visibleCharacterCount > DAILY_MAX_LENGTH) {
      setError(t('contentTooLong'));
      return;
    }
    setDialogOpen(true);
  };

  /** 提交富文本日常，成功后返回列表页，避免编辑内容直接嵌在列表中。 */
  const submitDaily = async () => {
    setIsSubmitting(true);
    setError('');
    try {
      await createDaily(normalizeDailyMediaHTML(editorHTML), isPrivate, selectedCoverID ?? undefined);
      router.push('/daily');
    } catch {
      setError(t('publishFailed'));
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mt-8 space-y-6">
      <div className="flex items-center justify-between gap-3">
        <Link
          href="/daily"
          className="inline-flex min-h-11 items-center gap-2 rounded-lg px-3 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          {t('backToDaily')}
        </Link>
        <span className="text-xs text-muted-foreground" aria-live="polite">
          {t('characterCount', { count: visibleCharacterCount, max: DAILY_MAX_LENGTH })}
        </span>
      </div>

      <section className="rounded-lg border border-border bg-surface p-4 sm:p-6">
        <div className="flex flex-wrap items-center gap-1 border-b border-border pb-3" role="toolbar" aria-label={t('formatToolbar')}>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('bold')} aria-label={t('bold')} title={t('bold')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Bold className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('italic')} aria-label={t('italic')} title={t('italic')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Italic className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('underline')} aria-label={t('underline')} title={t('underline')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Underline className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('strikeThrough')} aria-label={t('strikethrough')} title={t('strikethrough')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Strikethrough className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('insertUnorderedList')} aria-label={t('bulletedList')} title={t('bulletedList')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <List className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('insertOrderedList')} aria-label={t('numberedList')} title={t('numberedList')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <ListOrdered className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('formatBlock', '<blockquote>')} aria-label={t('blockquote')} title={t('blockquote')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Quote className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('formatBlock', '<pre>')} aria-label={t('codeBlock')} title={t('codeBlock')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <Code2 className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={insertLink} aria-label={t('insertLink')} title={t('insertLink')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <LinkIcon className="h-4 w-4" aria-hidden="true" />
          </button>
          <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => executeEditorCommand('removeFormat')} aria-label={t('clearFormatting')} title={t('clearFormatting')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            <RemoveFormatting className="h-4 w-4" aria-hidden="true" />
          </button>
          <label className="ml-auto flex min-h-10 items-center gap-2 text-xs text-muted-foreground">
            <span className="sr-only">{t('headingLevel')}</span>
            <select
              defaultValue=""
              onChange={(event) => {
                if (event.target.value) {
                  executeEditorCommand('formatBlock', event.target.value);
                }
                event.target.value = '';
              }}
              className="h-10 rounded-lg border border-border bg-background px-2 text-sm text-foreground outline-none focus:border-primary"
            >
              <option value="">{t('paragraph')}</option>
              <option value="<h1>">{t('headingOne')}</option>
              <option value="<h2>">{t('headingTwo')}</option>
              <option value="<h3>">{t('headingThree')}</option>
            </select>
          </label>
        </div>
        <div
          ref={editorRef}
          contentEditable
          role="textbox"
          aria-multiline="true"
          aria-label={t('editorLabel')}
          data-placeholder={t('editorPlaceholder')}
          onInput={updateEditorState}
          onDrop={handleMediaDrop}
          onPaste={handleMediaPaste}
          onDragOver={(event) => { if (event.dataTransfer.types.includes('Files')) event.preventDefault(); }}
          className="rich-text-editor article-content mt-4 min-h-64 max-w-none rounded-lg border border-border bg-background p-4 outline-none focus:border-primary"
          suppressContentEditableWarning
        />
        <input ref={imageInputRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp,image/avif" className="hidden" onChange={handleMediaInputChange} />
        <input ref={videoInputRef} type="file" accept="video/mp4,video/webm,video/ogg,video/quicktime" className="hidden" onChange={handleMediaInputChange} />
        <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <button type="button" onClick={() => imageInputRef.current?.click()} disabled={isMediaUploading} className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-border px-3 text-sm transition-colors hover:bg-muted disabled:opacity-50"><ImageIcon className="h-4 w-4" aria-hidden="true" />{t('insertImage')}</button>
          <button type="button" onClick={() => videoInputRef.current?.click()} disabled={isMediaUploading} className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-border px-3 text-sm transition-colors hover:bg-muted disabled:opacity-50"><span aria-hidden="true">▶</span>{t('insertVideo')}</button>
          {isMediaUploading ? <span role="status">{t('mediaUploading')}</span> : null}
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button type="button" onClick={() => void openCoverPicker()} className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-border px-4 text-sm transition-colors hover:bg-muted">
            <ImageIcon className="h-4 w-4" aria-hidden="true" />
            {t('chooseCover')}
          </button>
          {selectedCoverID ? <span className="text-xs text-muted-foreground">{t('coverSelected')}</span> : <span className="text-xs text-muted-foreground">{t('coverRandomHint')}</span>}
        </div>
        {error ? <p className="mt-3 text-sm text-red-500" role="alert">{error}</p> : null}
        <div className="mt-4 flex flex-wrap items-center justify-end gap-2">
          <Link href="/daily" className="inline-flex min-h-11 items-center rounded-lg border border-border px-4 text-sm transition-colors hover:bg-muted">
            {common('cancel')}
          </Link>
          <button ref={publishButtonRef} type="button" onClick={openPublishDialog} className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90">
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('publish')}
          </button>
        </div>
      </section>

      {dialogOpen ? (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-overlay p-4">
          <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="daily-confirm-title" aria-describedby="daily-confirm-description" className="w-full max-w-sm rounded-lg border border-border bg-surface p-5 shadow-xl">
            <h2 id="daily-confirm-title" className="text-lg font-semibold">{t('confirmTitle')}</h2>
            <p id="daily-confirm-description" className="mt-2 text-sm text-muted-foreground">{t('confirmDescription')}</p>
            <div className="mt-4 space-y-2">
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 text-sm">
                <input ref={dialogInitialFocusRef} name="daily-visibility" type="radio" checked={!isPrivate} onChange={() => setIsPrivate(false)} className="mt-0.5" />
                <span><strong className="block">{t('publicOption')}</strong><span className="text-muted-foreground">{t('publicHint')}</span></span>
              </label>
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 text-sm">
                <input name="daily-visibility" type="radio" checked={isPrivate} onChange={() => setIsPrivate(true)} className="mt-0.5" />
                <span><strong className="block">{t('privateOption')}</strong><span className="text-muted-foreground">{t('privateHint')}</span></span>
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button type="button" onClick={() => setDialogOpen(false)} className="min-h-11 rounded-lg border border-border px-4 text-sm transition-colors hover:bg-muted">{common('cancel')}</button>
              <button type="button" disabled={isSubmitting} onClick={() => void submitDaily()} className="min-h-11 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground disabled:opacity-50">{isSubmitting ? common('loading') : t('confirmPublish')}</button>
            </div>
          </div>
        </div>
      ) : null}

      {linkDialogOpen ? (
        <div className="fixed inset-0 z-[75] flex items-center justify-center bg-overlay p-4">
          <div ref={linkDialogRef} role="dialog" aria-modal="true" aria-labelledby="daily-link-title" className="w-full max-w-sm rounded-lg border border-border bg-surface p-5 shadow-xl">
            <div className="flex items-center justify-between gap-3">
              <h2 id="daily-link-title" className="text-lg font-semibold">{t('linkTitle')}</h2>
              <button type="button" onClick={() => setLinkDialogOpen(false)} aria-label={common('close')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted"><X className="h-4 w-4" aria-hidden="true" /></button>
            </div>
            <label className="mt-4 block text-sm font-medium" htmlFor="daily-link-url">{t('linkURLLabel')}</label>
            <input ref={linkInputRef} id="daily-link-url" value={linkURL} onChange={(event) => setLinkURL(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); confirmLink(); } }} placeholder="https://" autoFocus className="mt-2 h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" />
            <p className="mt-2 text-xs text-muted-foreground">{t('linkHint')}</p>
            <div className="mt-5 flex justify-end gap-2">
              <button type="button" onClick={() => setLinkDialogOpen(false)} className="min-h-11 rounded-lg border border-border px-4 text-sm hover:bg-muted">{common('cancel')}</button>
              <button type="button" onClick={confirmLink} disabled={!/^(https?:|mailto:)/i.test(linkURL.trim())} className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground disabled:opacity-50"><Check className="h-4 w-4" aria-hidden="true" />{t('insertLink')}</button>
            </div>
          </div>
        </div>
      ) : null}

      {coverPickerOpen ? (
        <div className="fixed inset-0 z-[72] flex items-center justify-center bg-overlay p-4">
          <div role="dialog" aria-modal="true" aria-labelledby="daily-cover-title" className="flex max-h-[85vh] w-full max-w-2xl flex-col rounded-lg border border-border bg-surface p-5 shadow-xl">
            <div className="flex items-center justify-between gap-3">
              <h2 id="daily-cover-title" className="text-lg font-semibold">{t('chooseCover')}</h2>
              <button type="button" onClick={() => setCoverPickerOpen(false)} aria-label={common('close')} className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted"><X className="h-4 w-4" aria-hidden="true" /></button>
            </div>
            {isCoverLoading ? <p className="py-8 text-center text-sm text-muted-foreground">{common('loading')}</p> : coverFiles.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">{t('coverEmpty')}</p> : <div className="mt-4 grid max-h-[60vh] grid-cols-2 gap-3 overflow-y-auto sm:grid-cols-3 md:grid-cols-4">
              {coverFiles.map((cover) => <button key={cover.id} type="button" onClick={() => { setSelectedCoverID(cover.id); setCoverPickerOpen(false); }} className={`group overflow-hidden rounded-lg border text-left transition-colors ${selectedCoverID === cover.id ? 'border-primary ring-2 ring-primary/30' : 'border-border hover:border-primary/60'}`}>
                <Image src={resolveMediaUrl(cover.thumbnailUrl || cover.mediumUrl)} alt={cover.altText || cover.displayName} width={cover.imageWidth || 480} height={cover.imageHeight || 480} unoptimized className="aspect-square w-full object-cover" />
                <span className="block truncate px-2 py-2 text-xs text-muted-foreground">{cover.displayName}</span>
              </button>)}
            </div>}
            {selectedCoverID ? <button type="button" onClick={() => { setSelectedCoverID(null); setCoverPickerOpen(false); }} className="mt-4 self-end text-sm text-muted-foreground hover:text-foreground">{t('removeCover')}</button> : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}
