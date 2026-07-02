import { useState, useEffect } from 'react';
import { useAuth } from '../lib/auth';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';

export default function HomePage() {
  const { user, updateBalance } = useAuth();
  const { show } = useToast();
  const [products, setProducts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [rechargeAmt, setRechargeAmt] = useState(10000);
  const [rechargeMode, setRechargeMode] = useState('wechat');

  useEffect(() => {
    fetchProducts();
  }, []);

  const fetchProducts = async () => {
    try {
      // Try fetching from admin endpoint first
      const data = await api.get<{ products: any[] }>('/admin/products');
      setProducts(data.products || []);
    } catch {
      // Fallback
    } finally {
      setLoading(false);
    }
  };

  const handleOrder = async (productId: number) => {
    try {
      const data = await api.createOrder(productId, 'mock_wechat');
      show(`下单成功: ${data.message}`, 'success');
      if (data.commissions?.length) {
        show(`获得分润: ${data.commissions.length} 笔`, 'info');
      }
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  const handleRecharge = async () => {
    try {
      const data = await api.recharge(rechargeAmt, rechargeMode);
      show(data.message, 'success');
      updateBalance(data.balance);
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  const fm = (cents: number) => '¥' + (cents / 100).toFixed(2);

  return (
    <div>
      {/* Dashboard Cards */}
      <div className="dashboard-cards">
        <div className="card">
          <div className="card-icon">💰</div>
          <div className="card-info">
            <span className="card-label">可用余额</span>
            <span className="card-value">¥{((user?.balance || 0)).toFixed(2)}</span>
          </div>
        </div>
        <div className="card">
          <div className="card-icon">📈</div>
          <div className="card-info">
            <span className="card-label">累计收益</span>
            <span className="card-value">¥{((user?.total_earned || 0) / 100).toFixed(2)}</span>
          </div>
        </div>
        <div className="card">
          <div className="card-icon">🏆</div>
          <div className="card-info">
            <span className="card-label">用户等级</span>
            <span className="card-value">Lv.{user?.level || 1}</span>
          </div>
        </div>
        <div className="card">
          <div className="card-icon">🔗</div>
          <div className="card-info">
            <span className="card-label">邀请码</span>
            <span className="card-value code">{user?.invite_code || '-'}</span>
          </div>
        </div>
      </div>

      {/* Recharge */}
      <div className="section-card">
        <h3>💰 充值</h3>
        <div className="form-row">
          <input
            type="number"
            value={rechargeAmt}
            onChange={e => setRechargeAmt(Number(e.target.value))}
            placeholder="金额(分)"
          />
          <select value={rechargeMode} onChange={e => setRechargeMode(e.target.value)}>
            <option value="wechat">微信支付</option>
            <option value="alipay">支付宝</option>
          </select>
          <button className="btn btn-primary" onClick={handleRecharge}>充值</button>
        </div>
      </div>

      {/* Products */}
      <div className="section-card">
        <h3>📦 商品列表</h3>
        {products.length === 0 ? (
          <p className="empty-state">暂无商品</p>
        ) : (
          <div className="product-list">
            {products.map((p: any) => (
              <div key={p.id} className="product-item">
                <div className="product-info">
                  <span className="product-name">{p.name}</span>
                  <span className="product-price">{fm(p.price)}</span>
                </div>
                <button className="btn btn-sm btn-primary" onClick={() => handleOrder(p.id)}>
                  购买
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
