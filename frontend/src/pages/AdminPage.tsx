import { useState } from 'react';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';
import type { Withdraw } from '../lib/api';

interface AuditLog {
  id: number;
  user_id: number;
  username: string;
  action: string;
  target: string;
  detail: string;
  ip: string;
  created_at: string;
}

export default function AdminPage() {
  const { show } = useToast();
  const [tab, setTab] = useState<'withdraws' | 'orders' | 'stats' | 'audit'>('stats');
  const [withdraws, setWithdraws] = useState<Withdraw[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);

  const loadStats = async () => {
    try {
      const data = await api.getAdminStats();
      setStats(data.stats);
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  const loadWithdraws = async () => {
    try {
      const data = await api.listAdminWithdraws();
      setWithdraws(data.withdraws || []);
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  const handleApprove = async (id: number, action: 'approve' | 'reject') => {
    try {
      await api.approveWithdraw(id, action);
      show(`提现${action === 'approve' ? '已批准' : '已拒绝'}`, 'success');
      loadWithdraws();
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  const loadAuditLogs = async () => {
    try {
      const data = await api.get<{ audit_logs: AuditLog[] }>('/admin/audit-logs');
      setAuditLogs(data.audit_logs || []);
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  const fm = (v: number) => '¥' + (v / 100).toFixed(2);

  return (
    <div>
      <div className="tab-bar">
        <button className={`tab ${tab === 'stats' ? 'active' : ''}`} onClick={() => { setTab('stats'); loadStats(); }}>概览</button>
        <button className={`tab ${tab === 'withdraws' ? 'active' : ''}`} onClick={() => { setTab('withdraws'); loadWithdraws(); }}>提现审核</button>
        <button className={`tab ${tab === 'orders' ? 'active' : ''}`} onClick={() => setTab('orders')}>订单</button>
        <button className={`tab ${tab === 'audit' ? 'active' : ''}`} onClick={() => { setTab('audit'); loadAuditLogs(); }}>审计日志</button>
      </div>

      {tab === 'stats' && (
        <div className="section-card">
          <h3>📊 管理概览</h3>
          <button className="btn btn-primary" onClick={loadStats}>刷新</button>
          {stats ? (
            <div className="stats-grid">
              <div className="stat-item"><span className="stat-label">总用户</span><span className="stat-value">{stats.total_users || 0}</span></div>
              <div className="stat-item"><span className="stat-label">总订单</span><span className="stat-value">{stats.total_orders || 0}</span></div>
              <div className="stat-item"><span className="stat-label">总金额</span><span className="stat-value">{fm(stats.total_revenue || 0)}</span></div>
              <div className="stat-item"><span className="stat-label">待审核提现</span><span className="stat-value">{stats.pending_withdraws || 0}</span></div>
            </div>
          ) : <p className="empty-state">点击刷新加载数据</p>}
        </div>
      )}

      {tab === 'withdraws' && (
        <div className="section-card">
          <h3>💳 提现审核</h3>
          <button className="btn btn-primary" onClick={loadWithdraws}>刷新</button>
          {withdraws.length === 0 ? <p className="empty-state">暂无提现记录</p> : (
            <div className="list">
              {withdraws.map(w => (
                <div key={w.id} className="list-item">
                  <div className="list-item-info">
                    <span>{w.account_name} - {w.bank_name}</span>
                    <span className="amount">{fm(w.amount)}</span>
                  </div>
                  <div className="list-item-meta">
                    <span className="status-badge">{w.status}</span>
                    <span className="date">{w.created_at?.substring(0, 10)}</span>
                  </div>
                  {w.status === 'pending' && (
                    <div className="list-item-actions">
                      <button className="btn btn-sm btn-success" onClick={() => handleApprove(w.id, 'approve')}>批准</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleApprove(w.id, 'reject')}>拒绝</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {tab === 'orders' && (
        <div className="section-card">
          <h3>📦 订单管理</h3>
          <p className="empty-state">订单管理功能开发中</p>
        </div>
      )}

      {tab === 'audit' && (
        <div className="section-card">
          <h3>📋 审计日志</h3>
          <button className="btn btn-primary" onClick={loadAuditLogs}>刷新</button>
          {auditLogs.length === 0 ? <p className="empty-state">暂无审计日志</p> : (
            <div className="list">
              {auditLogs.map(log => (
                <div key={log.id} className="list-item">
                  <div className="list-item-info">
                    <span>{log.username} - {log.action}</span>
                    <span className="status-badge">{log.target}</span>
                  </div>
                  <div className="list-item-meta">
                    <span>{log.ip}</span>
                    <span className="date">{log.created_at?.substring(0, 16).replace('T', ' ')}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
