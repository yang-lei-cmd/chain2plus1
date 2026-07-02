import { useState } from 'react';
import { api } from '../lib/api';
import { useAuth } from '../lib/auth';
import { useToast } from '../components/Toast';

export default function WithdrawPage() {
  const { user, updateBalance } = useAuth();
  const { show } = useToast();
  const [amount, setAmount] = useState(10000);
  const [bankName, setBankName] = useState('');
  const [accountName, setAccountName] = useState('');
  const [accountNo, setAccountNo] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (amount < 10000) return show('最低提现100元', 'error');
    if (!bankName || !accountName || !accountNo) return show('请填写完整银行信息', 'error');
    setSubmitting(true);
    try {
      const data = await api.applyWithdraw({ amount, bank_name: bankName, account_name: accountName, account_no: accountNo });
      show(data.message, 'success');
      updateBalance((user?.balance || 0) - amount / 100);
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setSubmitting(false);
    }
  };

  const fm = (v: number) => '¥' + v.toFixed(2);

  return (
    <div>
      <div className="section-card">
        <h3>💳 提现申请</h3>
        <div className="form-group">
          <label>可用余额: {fm(user?.balance || 0)}</label>
        </div>
        <div className="form-group">
          <input type="number" value={amount} onChange={e => setAmount(Number(e.target.value))} placeholder="金额(分)" />
          <small>最低100元(10000分), 手续费1%</small>
        </div>
        <div className="form-group">
          <input type="text" value={bankName} onChange={e => setBankName(e.target.value)} placeholder="银行名称" />
        </div>
        <div className="form-group">
          <input type="text" value={accountName} onChange={e => setAccountName(e.target.value)} placeholder="开户名" />
        </div>
        <div className="form-group">
          <input type="text" value={accountNo} onChange={e => setAccountNo(e.target.value)} placeholder="银行卡号" />
        </div>
        <button className="btn btn-primary" onClick={handleSubmit} disabled={submitting}>
          {submitting ? '提交中...' : '提交提现'}
        </button>
      </div>
    </div>
  );
}
