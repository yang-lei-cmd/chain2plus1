import { useState, useEffect } from 'react';

export default function PWAInstallPrompt() {
  const [deferredPrompt, setDeferredPrompt] = useState<any>(null);
  const [showPrompt, setShowPrompt] = useState(false);
  const [installed, setInstalled] = useState(false);

  useEffect(() => {
    const handler = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e);
      setShowPrompt(true);
    };

    window.addEventListener('beforeinstallprompt', handler);
    window.addEventListener('appinstalled', () => {
      setInstalled(true);
      setShowPrompt(false);
    });

    return () => window.removeEventListener('beforeinstallprompt', handler);
  }, []);

  // Also show prompt for iOS Safari
  const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent);
  const isStandalone = window.matchMedia('(display-mode: standalone)').matches;

  if (installed || isStandalone) return null;

  const handleInstall = async () => {
    if (deferredPrompt) {
      deferredPrompt.prompt();
      const result = await deferredPrompt.userChoice;
      if (result.outcome === 'accepted') {
        setInstalled(true);
      }
      setDeferredPrompt(null);
      setShowPrompt(false);
    }
  };

  const handleDismiss = () => {
    setShowPrompt(false);
    // Show again in 7 days
    localStorage.setItem('pwa_dismissed', String(Date.now() + 7 * 86400000));
  };

  // Check if previously dismissed
  const dismissedUntil = localStorage.getItem('pwa_dismissed');
  if (dismissedUntil && Date.now() < Number(dismissedUntil)) {
    return null;
  }

  if (!showPrompt && !isIOS) return null;

  return (
    <div className="pwa-prompt-overlay">
      <div className="pwa-prompt-card">
        <div className="pwa-prompt-icon">📱</div>
        <h4 className="pwa-prompt-title">添加到主屏幕</h4>
        {isIOS ? (
          <p className="pwa-prompt-text">
            点击 Safari 分享按钮 <strong>「添加到主屏幕」</strong>，获得 App 般的使用体验
          </p>
        ) : (
          <p className="pwa-prompt-text">
            安装到桌面，无需下载，即开即用，支持离线访问
          </p>
        )}
        <div className="pwa-prompt-buttons">
          {!isIOS && (
            <button className="btn btn-primary btn-sm" onClick={handleInstall}>
              立即安装
            </button>
          )}
          <button className="btn btn-sm btn-pwa-later" onClick={handleDismiss}>
            暂不
          </button>
        </div>
      </div>
    </div>
  );
}
