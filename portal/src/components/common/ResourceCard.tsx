import { Download, ExternalLink, FileText } from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import { resolveMediaUrl } from '@/config/constants';
import { formatDate, formatFileSize } from '@/utils/format';
import type { PublicFileListItem } from '@/types/publicContent';

/** 判断资源是否为可在浏览器内联预览的 PDF。 */
function isInlinePreviewable(contentType: string): boolean {
  return contentType === 'application/pdf';
}

/** ResourceCard 展示非图片文件资源，PDF 提供预览，其余仅提供受控下载。 */
export async function ResourceCard({
  resource,
  locale,
}: {
  resource: PublicFileListItem;
  locale: string;
}) {
  const t = await getTranslations('resources');
  const downloadUrl = resolveMediaUrl(resource.downloadUrl);
  const previewUrl = resolveMediaUrl(resource.previewUrl);
  const canPreview = isInlinePreviewable(resource.contentType) && previewUrl;

  return (
    <article className="flex flex-col gap-3 rounded-xl border border-border bg-surface p-5">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <FileText className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="min-w-0">
          <h3 className="break-words text-base font-semibold">{resource.displayName}</h3>
          {resource.description ? (
            <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
              {resource.description}
            </p>
          ) : null}
        </div>
      </div>

      <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <div className="flex justify-between gap-2">
          <dt>{t('fileType')}</dt>
          <dd className="truncate">{resource.contentType || '—'}</dd>
        </div>
        <div className="flex justify-between gap-2">
          <dt>{t('fileSize')}</dt>
          <dd>{formatFileSize(resource.size, locale)}</dd>
        </div>
        <div className="flex justify-between gap-2">
          <dt>{t('publishedAt')}</dt>
          <dd>{formatDate(resource.publishedAt, locale)}</dd>
        </div>
      </dl>

      <div className="flex gap-2">
        <a
          href={downloadUrl}
          className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          {t('download')}
        </a>
        {canPreview ? (
          <a
            href={previewUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium transition-colors hover:bg-muted"
          >
            <ExternalLink className="h-4 w-4" aria-hidden="true" />
            {t('preview')}
          </a>
        ) : null}
      </div>
    </article>
  );
}
