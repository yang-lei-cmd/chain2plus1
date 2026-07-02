import { useState } from 'react';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';
import type { Order } from '../lib/api';

export default function OrderPage() {
  const { show } = useToast();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);

  const loadOrders = async () => {
    setLoading(true);
    try {
      const data = await api.listOrders(page);
      setOrders(data.orders || []);
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="section-card">
        <h3>📦 我的订单</h3>
        <button className="btn btn-primary" onClick={loadOrders}>刷新</button>
        {loading ? <p className="loading">加载中...</p> : (
          orders.length === 0 ? <p className="empty-state">暂无订单</p> : (
            <div className="list">
              {orders.map(o => (
                <div key={o.id} className="list-item">
                  <div className="list-item-info">
                    <span>{o.product_name || `订单#${o.id}`}</span>
                    <span className="amount">¥{(o.amount / 100).toFixed(2)}</span>
                  </div>
                  <div className="list-item-meta">
                    <span className="status-badge">{o.status}</span>
                    <span className="date">{o.created_at?.substring(0, 10)}</span>
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
