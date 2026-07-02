import { useState } from 'react';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';
import type { FreelanceTask } from '../lib/api';

export default function FreelancePage() {
  const { show } = useToast();
  const [tasks, setTasks] = useState<FreelanceTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<'browse' | 'create'>('browse');

  // Create task form
  const [title, setTitle] = useState('');
  const [desc, setDesc] = useState('');
  const [category, setCategory] = useState('dev');
  const [budget, setBudget] = useState(1000);

  const loadTasks = async () => {
    setLoading(true);
    try {
      const data = await api.listTasks({ status: 'open' });
      setTasks(data.tasks || []);
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!title || !desc) return show('请填写标题和描述', 'error');
    try {
      await api.createTask({ title, description: desc, category, budget, duration_hours: 24 });
      show('任务创建成功', 'success');
      setTitle('');
      setDesc('');
      setTab('browse');
      loadTasks();
    } catch (e: any) {
      show(e.message, 'error');
    }
  };

  return (
    <div>
      <div className="tab-bar">
        <button className={`tab ${tab === 'browse' ? 'active' : ''}`} onClick={() => setTab('browse')}>任务大厅</button>
        <button className={`tab ${tab === 'create' ? 'active' : ''}`} onClick={() => setTab('create')}>发布任务</button>
      </div>

      {tab === 'browse' ? (
        <div className="section-card">
          <h3>🔧 任务大厅</h3>
          <button className="btn btn-primary" onClick={loadTasks}>刷新</button>
          {loading ? <p className="loading">加载中...</p> : (
            tasks.length === 0 ? <p className="empty-state">暂无任务</p> : (
              <div className="list">
                {tasks.map(t => (
                  <div key={t.id} className="list-item">
                    <div className="list-item-info">
                      <span>{t.title}</span>
                      <span className="amount">¥{(t.budget / 100).toFixed(2)}</span>
                    </div>
                    <div className="list-item-meta">
                      <span className="status-badge">{t.category}</span>
                      <span className="status-badge">{t.status}</span>
                    </div>
                  </div>
                ))}
              </div>
            )
          )}
        </div>
      ) : (
        <div className="section-card">
          <h3>📝 发布任务</h3>
          <div className="form-group"><input value={title} onChange={e => setTitle(e.target.value)} placeholder="任务标题" /></div>
          <div className="form-group"><textarea value={desc} onChange={e => setDesc(e.target.value)} placeholder="任务描述" rows={4} /></div>
          <div className="form-group">
            <select value={category} onChange={e => setCategory(e.target.value)}>
              <option value="dev">开发</option>
              <option value="design">设计</option>
              <option value="marketing">营销</option>
              <option value="writing">写作</option>
            </select>
          </div>
          <div className="form-group">
            <input type="number" value={budget} onChange={e => setBudget(Number(e.target.value))} placeholder="预算(分)" />
          </div>
          <button className="btn btn-primary" onClick={handleCreate}>发布</button>
        </div>
      )}
    </div>
  );
}
