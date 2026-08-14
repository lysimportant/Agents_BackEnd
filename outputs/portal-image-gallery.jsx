import React from 'react';

export default function ImageGalleryPreview() {
  const [theme, setTheme] = React.useState('light');
  const [selectedImage, setSelectedImage] = React.useState(null);
  const [hoveredImage, setHoveredImage] = React.useState(null);

  const themes = {
    light: { bg: '#f7fafc', card: '#ffffff', text: '#1a202c', muted: '#718096', border: '#e2e8f0', overlay: 'rgba(0,0,0,0.8)' },
    dark: { bg: '#1a202c', card: '#2d3748', text: '#f7fafc', muted: '#a0aec0', border: '#4a5568', overlay: 'rgba(0,0,0,0.95)' },
    warm: { bg: '#fff5f5', card: '#ffffff', text: '#742a2a', muted: '#9b2c2c', border: '#feb2b2', overlay: 'rgba(116,42,42,0.9)' }
  };

  const currentTheme = themes[theme];

  // 模拟瀑布流图片数据
  const images = [
    { id: 1, width: 400, height: 600, color: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', title: '现代建筑设计', category: '建筑' },
    { id: 2, width: 400, height: 300, color: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)', title: '抽象艺术作品', category: '艺术' },
    { id: 3, width: 400, height: 500, color: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)', title: '自然风光摄影', category: '摄影' },
    { id: 4, width: 400, height: 400, color: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)', title: 'UI设计示例', category: '设计' },
    { id: 5, width: 400, height: 550, color: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)', title: '产品展示图', category: '商业' },
    { id: 6, width: 400, height: 350, color: 'linear-gradient(135deg, #30cfd0 0%, #330867 100%)', title: '科技概念图', category: '科技' },
    { id: 7, width: 400, height: 450, color: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)', title: '插画作品', category: '插画' },
    { id: 8, width: 400, height: 520, color: 'linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%)', title: '室内设计', category: '设计' },
    { id: 9, width: 400, height: 380, color: 'linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%)', title: '时尚摄影', category: '摄影' }
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
              fontWeight: theme === t ? 'bold' : 'normal'
            }}
          >
            {t === 'light' ? '明亮' : t === 'dark' ? '深色' : '暖色'}
          </button>
        ))}
      </div>

      {/* 页面标题 */}
      <div style={{
        padding: '80px 20px 40px',
        maxWidth: '1400px',
        margin: '0 auto'
      }}>
        <h1 style={{
          fontSize: '36px',
          fontWeight: 'bold',
          color: currentTheme.text,
          marginBottom: '12px'
        }}>
          图片库
        </h1>
        <p style={{
          fontSize: '16px',
          color: currentTheme.muted
        }}>
          探索精选高质量图片资源
        </p>
      </div>

      {/* 瀑布流网格 */}
      <div style={{
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '0 20px 80px',
        columns: '3 400px',
        columnGap: '20px'
      }}>
        {images.map((image) => (
          <div
            key={image.id}
            onClick={() => setSelectedImage(image)}
            onMouseEnter={() => setHoveredImage(image.id)}
            onMouseLeave={() => setHoveredImage(null)}
            style={{
              breakInside: 'avoid',
              marginBottom: '20px',
              cursor: 'pointer',
              position: 'relative'
            }}
          >
            <div style={{
              background: currentTheme.card,
              borderRadius: '12px',
              overflow: 'hidden',
              boxShadow: hoveredImage === image.id
                ? '0 20px 40px rgba(0,0,0,0.15)'
                : '0 4px 12px rgba(0,0,0,0.08)',
              transition: 'all 0.3s ease',
              transform: hoveredImage === image.id ? 'scale(1.02)' : 'scale(1)',
              border: `1px solid ${currentTheme.border}`
            }}>
              {/* 图片区域 */}
              <div style={{
                width: '100%',
                height: `${(image.height / image.width) * 100}%`,
                paddingBottom: `${(image.height / image.width) * 100}%`,
                background: image.color,
                position: 'relative'
              }}>
                {/* 悬停遮罩 */}
                <div style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: 0,
                  background: hoveredImage === image.id
                    ? 'rgba(0,0,0,0.3)'
                    : 'transparent',
                  transition: 'background 0.3s ease',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center'
                }}>
                  {hoveredImage === image.id && (
                    <div style={{
                      background: 'rgba(255,255,255,0.95)',
                      color: currentTheme.text,
                      padding: '8px 16px',
                      borderRadius: '20px',
                      fontSize: '14px',
                      fontWeight: '500',
                      animation: 'fadeIn 0.2s ease'
                    }}>
                      点击查看
                    </div>
                  )}
                </div>

                {/* 分类标签 */}
                <div style={{
                  position: 'absolute',
                  top: '12px',
                  left: '12px',
                  background: 'rgba(0,0,0,0.6)',
                  backdropFilter: 'blur(8px)',
                  color: 'white',
                  padding: '4px 12px',
                  borderRadius: '16px',
                  fontSize: '12px',
                  fontWeight: '500'
                }}>
                  {image.category}
                </div>
              </div>

              {/* 标题 */}
              <div style={{
                padding: '16px',
                color: currentTheme.text,
                fontSize: '15px',
                fontWeight: '500'
              }}>
                {image.title}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* 图片预览浮层 */}
      {selectedImage && (
        <div
          onClick={() => setSelectedImage(null)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: currentTheme.overlay,
            backdropFilter: 'blur(20px)',
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '40px',
            animation: 'fadeIn 0.2s ease'
          }}
        >
          {/* 关闭按钮 */}
          <button
            onClick={() => setSelectedImage(null)}
            style={{
              position: 'absolute',
              top: '20px',
              right: '20px',
              width: '48px',
              height: '48px',
              borderRadius: '50%',
              background: 'rgba(255,255,255,0.2)',
              border: '2px solid rgba(255,255,255,0.3)',
              color: 'white',
              fontSize: '24px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.2s ease'
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'rgba(255,255,255,0.3)';
              e.currentTarget.style.transform = 'scale(1.1)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'rgba(255,255,255,0.2)';
              e.currentTarget.style.transform = 'scale(1)';
            }}
          >
            ✕
          </button>

          {/* 图片容器 */}
          <div
            onClick={(e) => e.stopPropagation()}
            style={{
              maxWidth: '90%',
              maxHeight: '90%',
              background: currentTheme.card,
              borderRadius: '16px',
              overflow: 'hidden',
              boxShadow: '0 25px 50px rgba(0,0,0,0.5)'
            }}
          >
            <div style={{
              width: '600px',
              maxWidth: '100%',
              height: `${(selectedImage.height / selectedImage.width) * 600}px`,
              background: selectedImage.color
            }} />
            <div style={{
              padding: '24px',
              background: currentTheme.card
            }}>
              <h3 style={{
                fontSize: '20px',
                fontWeight: 'bold',
                color: currentTheme.text,
                marginBottom: '8px'
              }}>
                {selectedImage.title}
              </h3>
              <div style={{
                fontSize: '14px',
                color: currentTheme.muted
              }}>
                分类: {selectedImage.category} · 尺寸: {selectedImage.width} × {selectedImage.height}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 说明 */}
      <div style={{
        padding: '40px 20px',
        textAlign: 'center',
        borderTop: `1px solid ${currentTheme.border}`,
        color: currentTheme.muted
      }}>
        <p style={{ fontSize: '14px', maxWidth: '800px', margin: '0 auto' }}>
          这是图片瀑布流页面效果。支持响应式多列布局、点击预览、悬停效果和三主题切换。
        </p>
      </div>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
      `}</style>
    </div>
  );
}
