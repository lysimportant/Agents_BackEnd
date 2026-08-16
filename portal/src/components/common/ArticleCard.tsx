import { Link } from '@/i18n/navigation';
import { formatDate } from '@/utils/format';
import type { PublicArticleListItem } from '@/types/publicContent';

/** ArticleCard 展示文章摘要卡片，包含标题、摘要、分类与发布时间。 */
export function ArticleCard({
  article,
  locale,
}: {
  article: PublicArticleListItem;
  locale: string;
}) {
  const href = '/articles/' + article.id + '/' + article.slug;

  return (
    <article className="group flex flex-col rounded-xl border border-border bg-surface p-5 transition-shadow hover:shadow-md">
      <Link href={href} className="flex flex-1 flex-col gap-2">
        <h3 className="line-clamp-2 text-base font-semibold leading-snug transition-colors group-hover:text-primary">
          {article.title}
        </h3>
        {article.summary ? (
          <p className="line-clamp-2 text-sm text-muted-foreground">{article.summary}</p>
        ) : null}
      </Link>
      <div className="mt-4 flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span className="truncate">{article.category || '—'}</span>
        <time dateTime={article.publishedAt} className="shrink-0">
          {formatDate(article.publishedAt, locale)}
        </time>
      </div>
    </article>
  );
}
