import { useState } from 'react';
import { useAuth } from '../lib/auth';
import { api } from '../lib/api';
import { useToast } from '../components/Toast';

export default function LoginPage() {
  const { login } = useAuth();
  const { show } = useToast();

  const [isRegister, setIsRegister] = useState(false);
  const [loading, setLoading] = useState(false);

  // Login fields
  const [lUser, setLUser] = useState('');
  const [lPass, setLPass] = useState('');

  // Register fields
  const [rUser, setRUser] = useState('');
  const [rPass, setRPass] = useState('');
  const [rPhone, setRPhone] = useState('');
  const [rEmail, setREmail] = useState('');
  const [rInvite, setRInvite] = useState('');

  const handleLogin = async () => {
    if (!lUser || !lPass) return show('请填写用户名和密码', 'error');
    setLoading(true);
    try {
      const data = await api.login(lUser, lPass);
      login(data.token, data.user);
      show('登录成功', 'success');
      window.location.hash = '/';
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async () => {
    if (!rUser || !rPass) return show('请填写用户名和密码', 'error');
    if (rPass.length < 6) return show('密码至少6位', 'error');
    setLoading(true);
    try {
      await api.register({
        username: rUser,
        password: rPass,
        phone: rPhone || undefined,
        email: rEmail || undefined,
        invite_code: rInvite || undefined,
      });
      show('注册成功，请登录', 'success');
      setIsRegister(false);
      setLUser(rUser);
    } catch (e: any) {
      show(e.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-container">
        <div className="logo">
          <h1>链动2+1</h1>
          <p>分销管理系统</p>
        </div>

        {!isRegister ? (
          <div className="auth-form">
            <h2>用户登录</h2>
            <div className="form-group">
              <input
                type="text"
                placeholder="用户名"
                value={lUser}
                onChange={e => setLUser(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleLogin()}
              />
            </div>
            <div className="form-group">
              <input
                type="password"
                placeholder="密码"
                value={lPass}
                onChange={e => setLPass(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleLogin()}
              />
            </div>
            <button className="btn btn-primary" onClick={handleLogin} disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </button>
            <p className="switch-form">
              还没有账号？
              <a href="#" onClick={() => setIsRegister(true)}>立即注册</a>
            </p>
          </div>
        ) : (
          <div className="auth-form">
            <h2>用户注册</h2>
            <div className="form-group">
              <input type="text" placeholder="用户名(3-32位)" value={rUser} onChange={e => setRUser(e.target.value)} />
            </div>
            <div className="form-group">
              <input type="password" placeholder="密码(至少6位)" value={rPass} onChange={e => setRPass(e.target.value)} />
            </div>
            <div className="form-group">
              <input type="tel" placeholder="手机号(选填)" value={rPhone} onChange={e => setRPhone(e.target.value)} />
            </div>
            <div className="form-group">
              <input type="email" placeholder="邮箱(选填)" value={rEmail} onChange={e => setREmail(e.target.value)} />
            </div>
            <div className="form-group">
              <input type="text" placeholder="邀请码(选填)" value={rInvite} onChange={e => setRInvite(e.target.value)} />
            </div>
            <button className="btn btn-primary" onClick={handleRegister} disabled={loading}>
              {loading ? '注册中...' : '注册'}
            </button>
            <p className="switch-form">
              已有账号？
              <a href="#" onClick={() => setIsRegister(false)}>立即登录</a>
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
