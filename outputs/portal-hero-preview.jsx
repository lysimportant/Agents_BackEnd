import React from 'react';

export default function PortalHeroPreview() {
  const [theme, setTheme] = React.useState('light');

  const themes = {
    light: {
      bg: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      text: '#ffffff',
      overlay: 'rgba(0, 0, 0, 0.3)',
      card: '#ffffff',
      cardText: '#1a202c'
    },
    dark: {
      bg: 'linear-gradient(135deg, #1a202c 0%, #2d3748 100%)',
      text: '#ffffff',
      overlay: 'rgba(0, 0, 0, 0.5)',
      card: '#2d3748',
      cardText: '#ffffff'
    },
    warm: {
      bg: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
      text: '#ffffff',
      overlay: 'rgba(255, 255, 255, 0.1)',
      card: '#ffffff',
      cardText: '#2d3748'
    }
  };

  const currentTheme = themes[theme];

  return (
    <div style={{ fontFamily: 'system-ui, -apple-system, sans-serif' }}>
      {/* 主题切换器 */}
      <div style={{
        position: 'fixed',
        top: 20,
        right: 20,
        zIndex: 100,
        display: 'flex',
        gap: '10px',
        background: 'rgba(255, 255, 255, 0.9)',
        padding: '10px',
        borderRadius: '8px',
        boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
      }}>
        <button
          onClick={() => setTheme('light')}
          style={{
            padding: '8px 16px',
            border: 'none',
            borderRadius: '6px',
            background: theme === 'light' ? '#667eea' : '#e2e8f0',
            color: theme === 'light' ? 'white' : '#4a5568',
            cursor: 'pointer',
            fontWeight: theme === 'light' ? 'bold' : 'normal'
          }}
        >
          明亮
        </button>
        <button
          onClick={() => setTheme('dark')}
          style={{
            padding: '8px 16px',
            border: 'none',
            borderRadius: '6px',
            background: theme === 'dark' ? '#2d3748' : '#e2e8f0',
            color: theme === 'dark' ? 'white' : '#4a5568',
            cursor: 'pointer',
            fontWeight: theme === 'dark' ? 'bold' : 'normal'
          }}
        >
          深色
        </button>
        <button
          onClick={() => setTheme('warm')}
          style={{
            padding: '8px 16px',
            border: 'none',
            borderRadius: '6px',
            background: theme === 'warm' ? '#f5576c' : '#e2e8f0',
            color: theme === 'warm' ? 'white' : '#4a5568',
            cursor: 'pointer',
            fontWeight: theme === 'warm' ? 'bold' : 'normal'
          }}
        >
          暖色
        </button>
      </div>

      {/* Hero区域 */}
      <div style={{
        minHeight: '600px',
        background: currentTheme.bg,
        position: 'relative',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        overflow: 'hidden',
        transition: 'all 0.5s ease'
      }}>
        {/* 背景装饰 */}
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: currentTheme.overlay,
          backdropFilter: 'blur(8px)'
        }} />

        {/* 几何装饰 */}
        <div style={{
          position: 'absolute',
          width: '400px',
          height: '400px',
          borderRadius: '50%',
          background: 'rgba(255, 255, 255, 0.1)',
          top: '-200px',
          right: '-200px',
          filter: 'blur(40px)'
        }} />
        <div style={{
          position: 'absolute',
          width: '300px',
          height: '300px',
          borderRadius: '50%',
          background: 'rgba(255, 255, 255, 0.1)',
          bottom: '-150px',
          left: '-150px',
          filter: 'blur(40px)'
        }} />

        {/* 内容区 */}
        <div style={{
          position: 'relative',
          zIndex: 10,
          textAlign: 'center',
          maxWidth: '800px',
          padding: '0 20px'
        }}>
          <h1 style={{
            fontSize: '56px',
            fontWeight: 'bold',
            color: currentTheme.text,
            margin: '0 0 20px 0',
            lineHeight: '1.2',
            textShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            HuaJian AI 内容门户
          </h1>
          <p style={{
            fontSize: '20px',
            color: currentTheme.text,
            margin: '0 0 40px 0',
            lineHeight: '1.6',
            opacity: 0.9
          }}>
            探索精选文章、高质量图片与专业资源
          </p>

          {/* 数据概览卡片 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
            gap: '20px',
            marginTop: '60px'
          }}>
            {[
              { label: '公开文章', value: '1,234', icon: '📝' },
              { label: '精选图片', value: '5,678', icon: '🖼️' },
              { label: '资源文件', value: '892', icon: '📁' },
              { label: '活跃分类', value: '24', icon: '🏷️' }
            ].map((item, index) => (
              <div
                key={index}
                style={{
                  background: currentTheme.card,
                  padding: '30px 20px',
                  borderRadius: '12px',
                  boxShadow: '0 10px 30px rgba(0,0,0,0.1)',
                  transition: 'transform 0.3s ease, box-shadow 0.3s ease',
                  cursor: 'pointer'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.transform = 'translateY(-5px)';
                  e.currentTarget.style.boxShadow = '0 15px 40px rgba(0,0,0,0.15)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.transform = 'translateY(0)';
                  e.currentTarget.style.boxShadow = '0 10px 30px rgba(0,0,0,0.1)';
                }}
              >
                <div style={{ fontSize: '36px', marginBottom: '10px' }}>
                  {item.icon}
                </div>
                <div style={{
                  fontSize: '32px',
                  fontWeight: 'bold',
                  color: currentTheme.cardText,
                  marginBottom: '5px'
                }}>
                  {item.value}
                </div>
                <div style={{
                  fontSize: '14px',
                  color: currentTheme.cardText,
                  opacity: 0.7
                }}>
                  {item.label}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* 说明文字 */}
      <div style={{
        padding: '40px 20px',
        textAlign: 'center',
        background: '#f7fafc',
        color: '#2d3748'
      }}>
        <p style={{ fontSize: '16px', maxWidth: '800px', margin: '0 auto', lineHeight: '1.8' }}>
          这是C端首页Hero区域的视觉效果示例。支持三种主题切换，包含高斯模糊背景、平滑过渡动画和数据概览卡片。
          实际实现中背景可以是真实的门户资源图片。
        </p>
      </div>
    </div>
  );
}