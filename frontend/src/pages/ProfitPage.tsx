import { useState } from 'react';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';
import type { ProfitShare } from '../lib/api';

export default function ProfitPage() {
  const { show } = useToast();
  const [profits, setProfits] = useState<ProfitShare[]>([]);
  const [loading, setLoading] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.listProfits();
      setProfits(data.profits || []);
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  const fm = (v: number) => '¥' + (v / 100).toFixed(2);

  return (
    <div>
      <div className="section-card">
        <h3>📈 收益明细</h3>
        <button className="btn btn-primary" onClick={load}>刷新</button>
        {loading ? <p className="loading">加载中...</p> : (
          profits.length === 0 ? <p className="empty-state">暂无收益记录</p> : (
            <div className="list">
              {profits.map(p => (
                <div key={p.id} className="list-item">
                  <div className="list-item-info">
                    <span>第{p.level}级分润</span>
                    <span className="amount">{fm(p.amount)}</span>
                  </div>
                  <div className="list-item-meta">
                    <span className="status-badge">{p.type}</span>
                    <span className="date">{p.created_at?.substring(0, 10)}</span>
                  </div>
                </div>
              ))}
            </div>
          )
        )}
      </div>
    </div>
  );
}
