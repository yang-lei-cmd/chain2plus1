import { useState } from 'react';
import { useAuth } from '../lib/auth';
import { useToast } from '../components/Toast';
import { generateQR } from '../lib/qrcode';

export default function SharePage() {
  const { user } = useAuth();
  const { show } = useToast();
  const [copied, setCopied] = useState(false);
  const [showQR, setShowQR] = useState(false);

  if (!user) return null;

  const inviteCode = user.invite_code || '';
  const shareUrl = `${window.location.origin}/#/register?invite=${inviteCode}`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      show('链接已复制', 'success');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      const ta = document.createElement('textarea');
      ta.value = shareUrl;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
      setCopied(true);
      show('链接已复制', 'success');
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div>
      {/* Share Card */}
      <div className="section-card">
        <h3>🔗 分享邀请</h3>
        <p className="share-intro">
          邀请好友加入，您将获得 <strong>10% 直推分润</strong> 和 <strong>8% 间接分润</strong>！
        </p>

        <div className="share-code-box">
          <div className="share-label">邀请码</div>
          <div className="share-code">{inviteCode}</div>
        </div>

        <div className="share-link-box">
          <div className="share-label">分享链接</div>
          <div className="share-link-row">
            <input type="text" className="share-link-input" value={shareUrl} readOnly />
            <button className="btn btn-sm btn-primary" onClick={handleCopy}>
              {copied ? '已复制 ✓' : '复制'}
            </button>
          </div>
        </div>

        {/* QR Code Section */}
        <div className="share-qr-section">
          {!showQR ? (
            <button className="btn btn-primary" onClick={() => setShowQR(true)}>
              📷 查看分享二维码
            </button>
          ) : (
            <div className="share-qr-container" dangerouslySetInnerHTML={{ __html: generateQR(shareUrl, 200) }} />
          )}
        </div>

        <div className="share-actions">
          <button
            className="btn btn-primary"
            onClick={() => {
              const text = `🎉 加入链动2+1分销系统！使用我的邀请码: ${inviteCode}\n${shareUrl}`;
              window.open(`https://servicewechat.com/weixin/index.html#${encodeURIComponent(text)}`, '_blank');
              show('请在微信中分享给好友', 'info');
            }}
          >
            📱 分享到微信
          </button>
        </div>

        <div className="share-stats">
          <div className="share-stat-item">
            <span className="share-stat-value">10%</span>
            <span className="share-stat-label">直推分润</span>
          </div>
          <div className="share-stat-item">
            <span className="share-stat-value">8%</span>
            <span className="share-stat-label">间接分润</span>
          </div>
          <div className="share-stat-item">
            <span className="share-stat-value">2+1</span>
            <span className="share-stat-label">链动模式</span>
          </div>
        </div>
      </div>
    </div>
  );
}
