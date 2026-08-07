/**
 * 文章卡片，展示标题、摘要、分类、作者与发布时间，并链接到文章详情。
 */
import { Link } from '@/navigation';
import { useFormatter } from 'next-intl';
import type { PublicArticleListItem } from '@/types/publicContent';

/** ArticleCard 渲染单篇文章的卡片摘要。 */
export function ArticleCard({ article }: { article: PublicArticleListItem }) {
  const format = useFormatter();
  return (
    <Link
      href={'/articles/' + article.id + '/' + encodeURIComponent(article.slug)}
      className="content-card fade-up block p-4 transition-transform duration-200 hover:-translate-y-0.5"
    >
      <h2 className="text-lg font-semibold leading-snug">{article.title}</h2>
      <p className="mt-2 line-clamp-2 text-sm text-ink-muted">{article.summary}</p>
      <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-muted">
        <span>{article.category}</span>
        <span>{article.author}</span>
        <time dateTime={article.publishedAt}>{format.dateTime(new Date(article.publishedAt), { dateStyle: 'medium' })}</time>
      </div>
    </Link>
  );
}