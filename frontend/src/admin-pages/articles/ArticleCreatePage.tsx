'use client';

import type { FormEvent } from 'react';
import { ArrowLeftOutlined, FileAddOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import type { ArticleForm } from '@/src/types/admin';
import { ArticleEditorForm } from '@/src/admin-pages/articles/ArticlesPage';

type ArticleCreatePageProps = {
  /** articleForm 保存待创建文章的完整表单内容。 */
  articleForm: ArticleForm;
  /** isSavingArticle 表示创建请求是否正在进行。 */
  isSavingArticle: boolean;
  /** onArticleFormChange 在字段变化时同步待创建文章。 */
  onArticleFormChange: (form: ArticleForm) => void;
  /** onSubmitArticle 创建文章，并在成功后返回文章列表。 */
  onSubmitArticle: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  /** onCancel 放弃本次创建并返回文章列表。 */
  onCancel: () => void;
};

/** ArticleCreatePage 在独立页面中承载文章创建表单。 */
export function ArticleCreatePage({
  articleForm,
  isSavingArticle,
  onArticleFormChange,
  onSubmitArticle,
  onCancel,
}: ArticleCreatePageProps) {
  return <section className="page-stack article-create-page" aria-labelledby="article-create-title">
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
  </section>;
}
