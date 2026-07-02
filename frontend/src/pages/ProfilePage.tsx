import { useState } from 'react';
import { api } from '../lib/api';
import { useAuth } from '../lib/auth';
import { useToast } from '../components/Toast';

export default function ProfilePage() {
  const { user } = useAuth();
  const { show } = useToast();
  const [tree, setTree] = useState<any>(null);
  const [showTree, setShowTree] = useState(false);

  const loadTree = async () => {
    try {
      const data = await api.getUserTree();
      setTree(data.tree);
      setShowTree(true);
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  return (
    <div>
      <div className="section-card">
        <h3>👤 个人信息</h3>
        <div className="info-grid">
          <div className="info-row"><span className="label">用户名</span><span>{user?.username}</span></div>
          <div className="info-row"><span className="label">手机</span><span>{user?.phone || '-'}</span></div>
          <div className="info-row"><span className="label">邮箱</span><span>{user?.email || '-'}</span></div>
          <div className="info-row"><span className="label">等级</span><span>Lv.{user?.level || 1}</span></div>
          <div className="info-row"><span className="label">角色</span><span>{user?.role === 'admin' ? '管理员' : '用户'}</span></div>
          <div className="info-row"><span className="label">邀请码</span><span className="code">{user?.invite_code}</span></div>
          <div className="info-row"><span className="label">余额</span><span>¥{((user?.balance || 0)).toFixed(2)}</span></div>
          <div className="info-row"><span className="label">累计收益</span><span>¥{((user?.total_earned || 0) / 100).toFixed(2)}</span></div>
        </div>
      </div>

      <div className="section-card">
        <button className="btn btn-primary" onClick={loadTree}>查看关系链</button>
        {showTree && tree && (
          <div className="tree-view">
            <pre>{JSON.stringify(tree, null, 2)}</pre>
          </div>
        )}
      </div>
    </div>
  );
}
