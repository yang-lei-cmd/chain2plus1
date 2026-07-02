import { useState, useEffect } from 'react';
import { useAuth } from '../lib/auth';
import { useWebSocket } from '../lib/websocket';

export default function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const ws = useWebSocket();
  const [menuOpen, setMenuOpen] = useState(false);

  // Connect WebSocket on mount if logged in
  useEffect(() => {
    if (user) {
      ws.connect();
    } else {
      ws.disconnect();
    }
    return () => ws.disconnect();
  }, [user]);

  const navItems = [
    { path: '/', label: '首页', icon: '🏠' },
    { path: '/orders', label: '下单', icon: '📦' },
    { path: '/withdraw', label: '提现', icon: '💳' },
    { path: '/profits', label: '收益', icon: '📈' },
    { path: '/freelance', label: '任务', icon: '🔧' },
  ];

  if (user?.role === 'admin') {
    navItems.push({ path: '/admin', label: '管理', icon: '⚙️' });
  }

  return (
    <div className="app-layout">
      <header className="header">
        <div className="header-left">
          <span
            className={`ws-indicator ${ws.isConnected ? 'ws-connected' : 'ws-disconnected'}`}
            title={ws.isConnected ? '已连接' : '未连接'}
          />
          <h2 className="header-title">链动2+1</h2>
        </div>
        <div className="header-right">
          <span className="balance-display">
            ¥{((user?.balance || 0)).toFixed(2)}
          </span>
          <button className="btn-icon" onClick={() => setMenuOpen(!menuOpen)}>
            {user?.username || '?'}
          </button>
          {menuOpen && (
            <div className="dropdown-menu">
              <div className="dropdown-item" onClick={() => { window.location.hash = '/profile'; setMenuOpen(false); }}>
                👤 个人中心
              </div>
              <div className="dropdown-item" onClick={() => { logout(); window.location.hash = '/login'; }}>
                🚪 退出登录
              </div>
            </div>
          )}
        </div>
      </header>

      <main className="main-content">{children}</main>

      <nav className="bottom-nav">
        {navItems.map(item => (
          <a
            key={item.path}
            href={`#${item.path}`}
            className={`nav-item ${window.location.hash === `#${item.path}` || window.location.hash === '' && item.path === '/' ? 'active' : ''}`}
            onClick={() => setMenuOpen(false)}
          >
            <span className="nav-icon">{item.icon}</span>
            <span className="nav-label">{item.label}</span>
          </a>
        ))}
      </nav>
    </div>
  );
}
