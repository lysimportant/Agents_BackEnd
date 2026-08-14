import React from 'react';

export default function ArticleCardsPreview() {
  const [theme, setTheme] = React.useState('light');
  const [hoveredCard, setHoveredCard] = React.useState(null);

  const themes = {
    light: { bg: '#f7fafc', card: '#ffffff', text: '#1a202c', muted: '#718096', border: '#e2e8f0' },
    dark: { bg: '#1a202c', card: '#2d3748', text: '#f7fafc', muted: '#a0aec0', border: '#4a5568' },
    warm: { bg: '#fff5f5', card: '#ffffff', text: '#742a2a', muted: '#9b2c2c', border: '#feb2b2' }
  };

  const currentTheme = themes[theme];

  const articles = [
    {
      id: 1,
      title: 'Next.js 16 App Router 完整指南',
      summary: '深入探讨 Next.js 16 的核心特性，包括服务端组件、流式渲染和增量静态再生成。',
      category: '前端开发',
      author: '张三',
      views: 2840,
      publishedAt: '2026-08-10',
      coverImage: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
    },
    {
      id: 2,
      title: '构建高性能的企业级应用',
      summary: '从架构设计到性能优化，分享构建可扩展企业应用的最佳实践和经验教训。',
      category: '架构设计',
      author: '李四',
      views: 1923,
      publishedAt: '2026-08-12',
      coverImage: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
    },
    {
      id: 3,
      title: 'TypeScript 5.0 新特性解析',
      summary: '详细介绍 TypeScript 5.0 带来的新功能，包括装饰器、类型推断增强等。',
      category: '编程语言',
      author: '王五',
      views: 3156,
      publishedAt: '2026-08-13',
      coverImage: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)'
    },
    {
      id: 4,
      title: 'React Server Components 实战',
      summary: '通过实际案例展示如何在生产环境中使用 React 服务端组件提升性能。',
      category: '前端开发',
      author: '赵六',
      views: 4521,
      publishedAt: '2026-08-14',
      coverImage: 'linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%)'
    },
    {
      id: 5,
      title: '无障碍设计原则与实践',
      summary: '学习如何设计和开发符合 WCAG 标准的无障碍 Web 应用程序。',
      category: '用户体验',
      author: '孙七',
      views: 1687,
      publishedAt: '2026-08-15',
      coverImage: 'linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%)'
    },
    {
      id: 6,
      title: 'Docker 容器化最佳实践',
      summary: '掌握 Docker 容器化技术，优化镜像大小，提升部署效率。',
      category: '运维部署',
      author: '周八',
      views: 2134,
      publishedAt: '2026-08-15',
      coverImage: 'linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%)'
    }
  ];

  return (
    <div style={{
      minHeight: '100vh',
      background: currentTheme.bg,
      fontFamily: 'system-ui, -apple-system, sans-serif',
      transition: 'background 0.3s ease'
    }}>
      {/* 主题切换器 */}
      <div style={{
        position: 'fixed',
        top: 20,
        right: 20,
        zIndex: 100,
        display: 'flex',
        gap: '10px',
        background: currentTheme.card,
        padding: '10px',
        borderRadius: '8px',
        boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
        border: `1px solid ${currentTheme.border}`
      }}>
        {['light', 'dark', 'warm'].map((t) => (
          <button
            key={t}
            onClick={() => setTheme(t)}
            style={{
              padding: '8px 16px',
              border: 'none',
              borderRadius: '6px',
              background: theme === t ? '#667eea' : currentTheme.border,
              color: theme === t ? 'white' : currentTheme.text,
              cursor: 'pointer',
              fontWeight: theme === t ? 'bold' : 'normal',
              transition: 'all 0.2s ease'
            }}
          >
            {t === 'light' ? '明亮' : t === 'dark' ? '深色' : '暖色'}
          </button>
        ))}
      </div>

      {/* 页面标题区 */}
      <div style={{
        padding: '80px 20px 40px',
        maxWidth: '1200px',
        margin: '0 auto'
      }}>
        <h1 style={{
          fontSize: '36px',
          fontWeight: 'bold',
          color: currentTheme.text,
          marginBottom: '12px'
        }}>
          精选文章
        </h1>
        <p style={{
          fontSize: '16px',
          color: currentTheme.muted
        }}>
          探索最新的技术文章与深度见解
        </p>

        {/* 筛选栏 */}
        <div style={{
          display: 'flex',
          gap: '12px',
          marginTop: '30px',
          flexWrap: 'wrap',
          alignItems: 'center'
        }}>
          <span style={{ color: currentTheme.muted, fontSize: '14px' }}>分类：</span>
          {['全部', '前端开发', '架构设计', '编程语言', '用户体验', '运维部署'].map((cat) => (
            <button
              key={cat}
              style={{
                padding: '6px 16px',
                border: `1px solid ${currentTheme.border}`,
                borderRadius: '20px',
                background: cat === '全部' ? currentTheme.text : currentTheme.card,
                color: cat === '全部' ? currentTheme.card : currentTheme.text,
                fontSize: '14px',
                cursor: 'pointer',
                transition: 'all 0.2s ease'
              }}
              onMouseEnter={(e) => {
                if (cat !== '全部') {
                  e.currentTarget.style.background = currentTheme.border;
                }
              }}
              onMouseLeave={(e) => {
                if (cat !== '全部') {
                  e.currentTarget.style.background = currentTheme.card;
                }
              }}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* 文章卡片网格 */}
      <div style={{
        maxWidth: '1200px',
        margin: '0 auto',
        padding: '0 20px 80px',
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))',
        gap: '24px'
      }}>
        {articles.map((article) => (
          <article
            key={article.id}
            onMouseEnter={() => setHoveredCard(article.id)}
            onMouseLeave={() => setHoveredCard(null)}
            style={{
              background: currentTheme.card,
              borderRadius: '12px',
              overflow: 'hidden',
              boxShadow: hoveredCard === article.id
                ? '0 20px 40px rgba(0,0,0,0.15)'
                : '0 4px 12px rgba(0,0,0,0.08)',
              transition: 'all 0.3s ease',
              transform: hoveredCard === article.id ? 'translateY(-8px)' : 'translateY(0)',
              border: `1px solid ${currentTheme.border}`,
              cursor: 'pointer'
            }}
          >
            {/* 封面图 */}
            <div style={{
              height: '200px',
              background: article.coverImage,
              position: 'relative',
              overflow: 'hidden'
            }}>
              <div style={{
                position: 'absolute',
                top: '12px',
                right: '12px',
                background: 'rgba(0,0,0,0.6)',
                backdropFilter: 'blur(8px)',
                color: 'white',
                padding: '4px 12px',
                borderRadius: '16px',
                fontSize: '12px',
                fontWeight: '500'
              }}>
                {article.category}
              </div>
            </div>

            {/* 内容区 */}
            <div style={{ padding: '20px' }}>
              <h2 style={{
                fontSize: '20px',
                fontWeight: 'bold',
                color: currentTheme.text,
                marginBottom: '12px',
                lineHeight: '1.4',
                display: '-webkit-box',
                WebkitLineClamp: 2,
                WebkitBoxOrient: 'vertical',
                overflow: 'hidden'
              }}>
                {article.title}
              </h2>

              <p style={{
                fontSize: '14px',
                color: currentTheme.muted,
                lineHeight: '1.6',
                marginBottom: '16px',
                display: '-webkit-box',
                WebkitLineClamp: 3,
                WebkitBoxOrient: 'vertical',
                overflow: 'hidden'
              }}>
                {article.summary}
              </p>

              {/* 元信息 */}
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                paddingTop: '16px',
                borderTop: `1px solid ${currentTheme.border}`
              }}>
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}>
                  <div style={{
                    width: '32px',
                    height: '32px',
                    borderRadius: '50%',
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: 'white',
                    fontSize: '14px',
                    fontWeight: 'bold'
                  }}>
                    {article.author[0]}
                  </div>
                  <div>
                    <div style={{
                      fontSize: '13px',
                      color: currentTheme.text,
                      fontWeight: '500'
                    }}>
                      {article.author}
                    </div>
                    <div style={{
                      fontSize: '12px',
                      color: currentTheme.muted
                    }}>
                      {article.publishedAt}
                    </div>
                  </div>
                </div>

                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  color: currentTheme.muted,
                  fontSize: '13px'
                }}>
                  <span>👁️</span>
                  <span>{article.views.toLocaleString()}</span>
                </div>
              </div>
            </div>
          </article>
        ))}
      </div>

      {/* 说明 */}
      <div style={{
        padding: '40px 20px',
        textAlign: 'center',
        borderTop: `1px solid ${currentTheme.border}`,
        color: currentTheme.muted
      }}>
        <p style={{ fontSize: '14px', maxWidth: '800px', margin: '0 auto' }}>
          这是文章列表页的卡片布局效果。支持响应式网格、悬停动画、分类筛选和三主题切换。
        </p>
      </div>
    </div>
  );
}
