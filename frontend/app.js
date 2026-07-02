// ========== 配置 ==========
const API_BASE = 'http://localhost:8080/api/v1';
let TOKEN = localStorage.getItem('token') || '';
let CURRENT_USER = JSON.parse(localStorage.getItem('user') || '{}');

// ========== Axios 配置 ==========
axios.defaults.baseURL = API_BASE;
axios.defaults.timeout = 10000;

axios.interceptors.request.use(config => {
    if (TOKEN) {
        config.headers['Authorization'] = `Bearer ${TOKEN}`;
    }
    return config;
});

axios.interceptors.response.use(
    res => res.data,
    err => {
        const msg = err.response?.data?.error || err.message || '网络错误';
        if (err.response?.status === 401) {
            TOKEN = '';
            CURRENT_USER = {};
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            showToast('登录已过期，请重新登录');
            showAuthPage();
        }
        showToast(msg);
        return Promise.reject(err);
    }
);

// ========== 工具函数 ==========
function fmtMoney(cents) {
    return '¥' + ((cents || 0) / 100).toFixed(2);
}

function fmtMoneyShort(cents) {
    const yuan = (cents || 0) / 100;
    if (yuan >= 10000) return '¥' + (yuan / 10000).toFixed(2) + '万';
    return '¥' + yuan.toFixed(2);
}

function formatDate(t) {
    if (!t) return '-';
    return t.toString().length === 10 ? t.substring(0, 10) : t.substring(0, 16).replace('T', ' ');
}

function statusBadge(status) {
    const map = {
        pending: ['待审核', 'status-pending'],
        approved: ['已通过', 'status-approved'],
        rejected: ['已拒绝', 'status-rejected'],
        paid: ['已打款', 'status-paid'],
        completed: ['已完成', 'status-completed'],
        active: ['正常', 'status-approved'],
        disabled: ['禁用', 'status-rejected'],
    };
    const [label, cls] = map[status] || [status, ''];
    return `<span class="status-badge ${cls}">${label}</span>`;
}

function showToast(msg) {
    const el = document.createElement('div');
    el.className = 'toast';
    el.textContent = msg;
    document.body.appendChild(el);
    setTimeout(() => el.remove(), 2500);
}

// ========== 认证相关 ==========
function showAuthPage() {
    document.getElementById('auth-page').style.display = '';
    document.getElementById('app-page').style.display = 'none';
}

function showMainPage() {
    document.getElementById('auth-page').style.display = 'none';
    document.getElementById('app-page').style.display = '';
}

function showRegister() {
    document.getElementById('login-form').style.display = 'none';
    document.getElementById('register-form').style.display = '';
}

function showLogin() {
    document.getElementById('register-form').style.display = 'none';
    document.getElementById('login-form').style.display = '';
}

async function login() {
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value;
    if (!username || !password) return showToast('请填写用户名和密码');
    try {
        const data = await axios.post('/auth/login', { username, password });
        TOKEN = data.token;
        CURRENT_USER = {
            id: data.user.id,
            username: data.user.username,
            phone: data.user.phone,
            email: data.user.email,
            role: data.user.role,
            level: data.user.level,
            balance: data.user.balance,
            total_earned: data.user.total_earned,
            invite_code: data.user.invite_code,
            invitee_code: data.user.invitee_code,
        };
        localStorage.setItem('token', TOKEN);
        localStorage.setItem('user', JSON.stringify(CURRENT_USER));
        showToast('登录成功');
        showMainPage();
        loadDashboard();
        loadProfile();
        if (CURRENT_USER.role === 'admin') {
            document.getElementById('admin-menu-item').style.display = 'flex';
            document.getElementById('admin-dashboard-menu').style.display = 'flex';
        }
    } catch (e) {
        // error handler intercepts
    }
}

async function register() {
    const username = document.getElementById('reg-username').value.trim();
    const password = document.getElementById('reg-password').value;
    const phone = document.getElementById('reg-phone').value.trim();
    const email = document.getElementById('reg-email').value.trim();
    const invite_code = document.getElementById('reg-invite').value.trim();
    if (!username || !password) return showToast('请填写用户名和密码');
    if (password.length < 6) return showToast('密码至少6位');
    try {
        const body = { username, password };
        if (phone) body.phone = phone;
        if (email) body.email = email;
        if (invite_code) body.invite_code = invite_code;
        const data = await axios.post('/auth/register', body);
        TOKEN = data.token;
        CURRENT_USER = {
            id: data.user.id,
            username: data.user.username,
            phone: data.user.phone || '',
            email: data.user.email || '',
            role: data.user.role,
            level: data.user.level,
            balance: data.user.balance,
            total_earned: data.user.total_earned,
            invite_code: data.user.invite_code,
            invitee_code: data.user.invitee_code,
        };
        localStorage.setItem('token', TOKEN);
        localStorage.setItem('user', JSON.stringify(CURRENT_USER));
        showToast('注册成功');
        showMainPage();
        loadDashboard();
        loadProfile();
    } catch (e) {}
}

function logout() {
    TOKEN = '';
    CURRENT_USER = {};
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    showAuthPage();
}

// ========== 导航 ==========
const pageTitles = {
    home: '首页', order: '充值订单', withdraw: '提现',
    leaderboard: '排行榜', team: '关系链', profit: '收益明细',
    profile: '个人中心', admin: '管理后台'
};

let currentSection = 'home';

function navigateTo(section) {
    // Hide all sections
    document.querySelectorAll('.content-section').forEach(s => s.style.display = 'none');
    // Show target
    const target = document.getElementById(section + '-section');
    if (target) target.style.display = '';
    
    // Update header title
    document.getElementById('header-title').textContent = pageTitles[section] || '首页';
    
    // Update bottom nav
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    const navBtn = document.querySelector(`.nav-item[onclick*="'${section}'"]`);
    if (navBtn) navBtn.classList.add('active');

    // Load data on navigation
    if (section === 'home') loadDashboard();
    else if (section === 'order') loadProducts();
    else if (section === 'withdraw') loadWithdrawPage();
    else if (section === 'leaderboard') switchLeaderboard('total_earned', document.querySelector('.leaderboard-tabs .tab-btn.active') || document.querySelector('.leaderboard-tabs .tab-btn'));
    else if (section === 'team') loadUserTree();
    else if (section === 'profit') loadProfits('all');
    else if (section === 'profile') loadProfile();
    else if (section === 'admin') loadAdminStats();

    currentSection = section;
    window.scrollTo(0, 0);
}

// ========== 首页仪表盘 ==========
async function loadDashboard() {
    if (!TOKEN) return;
    try {
        const data = await axios.get('/user/profile');
        document.getElementById('dashboard-balance').textContent = fmtMoney(data.balance);
        document.getElementById('dashboard-total-earned').textContent = fmtMoney(data.total_earned || 0);
        document.getElementById('dashboard-team-size').textContent = data.team_size || 0;
        document.getElementById('user-level').textContent = data.level || 0;
        document.getElementById('user-balance-header').textContent = fmtMoney(data.balance);
        loadRecentProfits();
    } catch (e) {}
}

async function loadRecentProfits() {
    try {
        const profits = await axios.get('/profit/list', { params: { page: 1, page_size: 5 } });
        const container = document.getElementById('recent-profits');
        if (!profits.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><div class="empty-state-text">暂无收益记录</div></div>';
            return;
        }
        container.innerHTML = profits.map(p => `
            <div class="list-item">
                <div class="list-item-header">
                    <span class="list-item-title">${p.order_no}</span>
                    ${statusBadge(p.type)}
                </div>
                <div class="list-item-footer">
                    <span class="list-item-meta">${formatDate(p.created_at)}</span>
                    <span class="amount-positive">+${typeof p.amount === 'string' ? p.amount : fmtMoney(p.amount)}</span>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

// ========== 个人资料 ==========
async function loadProfile() {
    if (!CURRENT_USER.id) return;
    try {
        const data = await axios.get('/user/profile');
        document.getElementById('profile-username').textContent = data.username || '';
        document.getElementById('profile-phone').textContent = '手机：' + (data.phone || '-');
        document.getElementById('profile-email').textContent = '邮箱：' + (data.email || '-');
        document.getElementById('profile-invite').textContent = '邀请码：' + (data.invite_code || '-');
        document.getElementById('profile-level').textContent = '等级：Lv.' + (data.level || 0);
        document.getElementById('user-level').textContent = data.level || 0;
    } catch (e) {}
}

// ========== 商品和订单 ==========
async function loadProducts() {
    const container = document.getElementById('product-list');
    try {
        const products = await axios.get('/admin/products');
        if (!products.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📦</div><div class="empty-state-text">暂无商品信息</div></div>';
            return;
        }
        container.innerHTML = products.map(p => `
            <div class="product-card" onclick="createOrder(${p.id}, '${p.name}', ${p.price})">
                <div class="product-image">🛍️</div>
                <div class="product-info">
                    <div class="product-name">${p.name}</div>
                    <div class="product-price">${fmtMoney(p.price * 100)}</div>
                    <div class="product-supplier">${p.supplier_name || ''}</div>
                </div>
            </div>
        `).join('');
        loadOrderList();
    } catch (e) {}
}

async function createOrder(product_id, product_name, price) {
    if (!TOKEN) return showToast('请先登录');
    const confirmMsg = `确认购买 ${product_name}？\n金额：${fmtMoney(price * 100)}`;
    if (!confirm(confirmMsg)) return;
    try {
        await axios.post('/order/create', { product_id });
        showToast('下单成功');
        loadProducts();
        loadDashboard();
    } catch (e) {}
}

async function loadOrderList() {
    try {
        const orders = await axios.get('/order/list', { params: { page: 1, page_size: 20 } });
        const container = document.getElementById('order-list');
        if (!orders.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📋</div><div class="empty-state-text">暂无订单</div></div>';
            return;
        }
        container.innerHTML = orders.map(o => `
            <div class="list-item">
                <div class="list-item-header">
                    <span class="list-item-title">${o.order_no}</span>
                    ${statusBadge(o.status)}
                </div>
                <div class="list-item-footer">
                    <span class="list-item-meta">${o.product_name || ''}</span>
                    <span class="amount-negative">-${o.amount}</span>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

// ========== 提现 ==========
async function loadWithdrawPage() {
    loadWithdrawList();
}

async function applyWithdraw() {
    const amount = parseInt(document.getElementById('withdraw-amount').value);
    const bank_name = document.getElementById('withdraw-bank').value.trim();
    const account_name = document.getElementById('withdraw-account-name').value.trim();
    const account_no = document.getElementById('withdraw-account-no').value.trim();

    if (!amount || amount < 10000) return showToast('最低提现金额：¥100.00');
    if (!bank_name || !account_name || !account_no) return showToast('请填写完整收款信息');

    try {
        await axios.post('/withdraw/apply', {
            amount: amount * 100, // convert to cents
            bank_name,
            account_name,
            account_no,
        });
        showToast('提现申请已提交');
        document.getElementById('withdraw-amount').value = '';
        document.getElementById('withdraw-bank').value = '';
        document.getElementById('withdraw-account-name').value = '';
        document.getElementById('withdraw-account-no').value = '';
        loadWithdrawList();
        loadDashboard();
    } catch (e) {}
}

async function loadWithdrawList() {
    try {
        const withdraws = await axios.get('/withdraw/list', { params: { page: 1, page_size: 20 } });
        const container = document.getElementById('withdraw-list');
        if (!withdraws.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">💸</div><div class="empty-state-text">暂无提现记录</div></div>';
            return;
        }
        container.innerHTML = withdraws.map(w => `
            <div class="list-item">
                <div class="list-item-header">
                    <span class="list-item-title">${fmtMoney(w.amount)}</span>
                    ${statusBadge(w.status)}
                </div>
                <div class="list-item-footer">
                    <span class="list-item-meta">${formatDate(w.created_at)}</span>
                    <span class="list-item-meta">${w.bank_name || ''}</span>
                </div>
                ${w.remark ? `<div class="list-item-meta" style="margin-top:4px">${w.remark}</div>` : ''}
            </div>
        `).join('');
    } catch (e) {}
}

// ========== 收益明细 ==========
let currentProfitFilter = 'all';

async function loadProfits(filter) {
    currentProfitFilter = filter || 'all';
    try {
        const profits = await axios.get('/profit/list', { params: { page: 1, page_size: 50 } });
        let filtered = profits;
        if (currentProfitFilter !== 'all') {
            const levelMap = { level1: 1, level2: 2 };
            filtered = profits.filter(p => p.level === levelMap[currentProfitFilter]);
        }
        const container = document.getElementById('profit-list');
        if (!filtered.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📊</div><div class="empty-state-text">暂无收益记录</div></div>';
            return;
        }
        container.innerHTML = filtered.map(p => {
            const levelLabel = p.level === 1 ? '一级推广' : p.level === 2 ? '二级推广' : p.type || '推广奖励';
            return `
                <div class="list-item">
                    <div class="list-item-header">
                        <span class="list-item-title">${levelLabel}</span>
                        ${statusBadge(p.status)}
                    </div>
                    <div class="list-item-meta">${p.order_no || ''} - ${p.from_user || ''}</div>
                    <div class="list-item-footer" style="margin-top:8px">
                        <span class="list-item-meta">${formatDate(p.created_at)}</span>
                        <span class="amount-positive">+${fmtMoney(typeof p.amount === 'string' ? parseFloat(p.amount) * 100 : p.amount)}</span>
                    </div>
                    ${p.description ? `<div class="list-item-meta" style="margin-top:4px;font-style:italic">${p.description}</div>` : ''}
                </div>
            `;
        }).join('');
    } catch (e) {}
}

function filterProfits(filter, btn) {
    document.querySelectorAll('#profit-section .filter-btn').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    loadProfits(filter);
}

// ========== 排行榜 ==========
async function switchLeaderboard(type, btn) {
    if (btn) {
        document.querySelectorAll('.leaderboard-tabs .tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
    }
    try {
        const data = await axios.get(`/leaderboard/${type}`, { params: { page: 1, page_size: 50 } });
        const container = document.getElementById('leaderboard-list');
        if (!data.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">🏆</div><div class="empty-state-text">暂无数据</div></div>';
            return;
        }
        const titles = { total_earned: '累计收益', team_size: '团队人数', recharge: '充值金额' };
        const unit = { total_earned: '', team_size: '人', recharge: '' };
        container.innerHTML = data.map(item => {
            const rankClass = item.rank <= 3 ? `rank-${item.rank}` : '';
            const valueStr = typeof item.rank_value === 'number' ? (unit[type] ? item.rank_value + unit[type] : fmtMoney(item.rank_value)) : item.rank_value;
            return `
                <div class="leaderboard-item">
                    <div class="rank-number ${rankClass}">#${item.rank}</div>
                    <div class="leaderboard-info">
                        <div class="leaderboard-name">${item.username}</div>
                        <div class="leaderboard-value">${valueStr}</div>
                    </div>
                </div>
            `;
        }).join('');
    } catch (e) {}
}

// ========== 关系链 ==========
async function loadUserTree() {
    const container = document.getElementById('user-tree');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const data = await axios.get('/user/tree');
        renderTree(data, container);
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">🌳</div><div class="empty-state-text">加载失败</div></div>';
    }
}

function renderTree(user, container) {
    const children = user.children || [];
    let html = `<div class="tree-node tree-node-root">
        <div class="tree-user">
            <div class="tree-user-name">${user.username}</div>
            <div class="tree-user-info">邀请码：${user.invite_code || '-'}</div>
            <div class="tree-user-info">余额：${fmtMoney(user.balance)}</div>
        </div>
    </div>`;
    if (children.length) {
        html += '<div class="tree-node">';
        html += '<div style="font-weight:600;margin-bottom:8px;color:var(--text-secondary)">└─ 我的下级</div>';
        children.forEach(child => {
            html += renderTreeNode(child);
        });
        html += '</div>';
    } else {
        html += '<div style="text-align:center;padding:20px;color:var(--text-light)">暂无下级用户</div>';
    }
    container.innerHTML = html;
}

function renderTreeNode(user) {
    const children = user.children || [];
    let html = `
        <div class="tree-node">
            <div class="tree-user">
                <div class="tree-user-name">${user.username}</div>
                <div class="tree-user-info">邀请码：${user.invite_code || '-'}</div>
                <div class="tree-user-info">余额：${fmtMoney(user.balance)}</div>
            </div>`;
    if (children.length) {
        html += '<div style="margin-top:8px">';
        children.forEach(child => {
            html += renderTreeNode(child);
        });
        html += '</div>';
    }
    html += '</div>';
    return html;
}

// ========== 管理后台 ==========
async function loadAdminStats() {
    if (!TOKEN || CURRENT_USER.role !== 'admin') return showToast('无权限访问');
    try {
        const data = await axios.get('/admin/stats');
        document.getElementById('admin-total-users').textContent = data.total_users || 0;
        document.getElementById('admin-active-users').textContent = data.active_users || 0;
        document.getElementById('admin-total-orders').textContent = data.total_orders || 0;
        document.getElementById('admin-total-revenue').textContent = fmtMoney(data.total_revenue || 0);
        document.getElementById('admin-today-orders').textContent = data.today_stats?.total_orders || 0;
        document.getElementById('admin-today-revenue').textContent = fmtMoney(data.today_stats?.total_revenue || 0);
        document.getElementById('admin-pending-withdraw').textContent = data.pending_withdrawals || 0;
        loadAdminUsers();
        loadAdminWithdraws();
        loadAdminSuppliers();
        loadAdminProducts();
    } catch (e) {}
}

function switchAdminTab(tab, btn) {
    document.querySelectorAll('.admin-tabs .tab-btn').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    document.querySelectorAll('.admin-panel').forEach(p => p.style.display = 'none');
    document.getElementById('admin-' + tab + '-panel').style.display = '';
}

async function loadAdminUsers() {
    try {
        const users = await axios.get('/admin/users', { params: { page: 1, page_size: 50 } });
        const container = document.getElementById('admin-user-list');
        container.innerHTML = users.map(u => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${u.username}</div>
                    <div class="list-item-meta">手机号：${u.phone || '-'} | 余额：${fmtMoney(u.balance)}</div>
                </div>
                <div class="operator">
                    ${statusBadge(u.status === 1 ? 'active' : 'disabled')}
                    <button class="btn btn-sm ${u.status === 1 ? 'btn-warning' : 'btn-success'}"
                        onclick="toggleUserStatus(${u.id}, ${u.status === 1 ? 0 : 1})">
                        ${u.status === 1 ? '禁用' : '启用'}
                    </button>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

async function toggleUserStatus(userId, status) {
    const action = status === 1 ? '启用' : '禁用';
    if (!confirm(`确定${action}该用户？`)) return;
    try {
        await axios.patch(`/admin/users/${userId}/status`, { status });
        showToast(`${action}成功`);
        loadAdminUsers();
        loadAdminStats();
    } catch (e) {}
}

async function loadAdminWithdraws() {
    try {
        const withdraws = await axios.get('/admin/withdraw/list', { params: { page: 1, page_size: 50 } });
        const container = document.getElementById('admin-withdraw-list');
        if (!withdraws.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">✅</div><div class="empty-state-text">暂无提现记录</div></div>';
            return;
        }
        container.innerHTML = withdraws.map(w => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${fmtMoney(w.amount)} (${w.actual_amount ? fmtMoney(w.actual_amount) : '-'})</div>
                    <div class="list-item-meta">${w.account_name} - ${w.account_no || ''}</div>
                    <div class="list-item-meta">状态：${statusBadge(w.status)}</div>
                </div>
                ${w.status === 'pending' ? `
                    <div class="operator">
                        <button class="btn btn-sm btn-success" onclick="approveWithdraw(${w.id}, 'approved')">通过</button>
                        <button class="btn btn-sm btn-danger" onclick="approveWithdraw(${w.id}, 'rejected')">拒绝</button>
                    </div>
                ` : '<span class="list-item-meta">已完成</span>'}
            </div>
        `).join('');
    } catch (e) {}
}

async function approveWithdraw(withdrawId, action) {
    const remark = action === 'approved' ? '' : prompt('请输入拒绝原因：') || '拒绝';
    if (action === 'approved' && !confirm('确认通过该提现申请？')) return;
    try {
        await axios.patch(`/admin/withdraw/${withdrawId}/approve`, { action, remark });
        showToast(action === 'approved' ? '已通过' : '已拒绝');
        loadAdminWithdraws();
        loadAdminStats();
    } catch (e) {}
}

async function loadAdminSuppliers() {
    try {
        const suppliers = await axios.get('/admin/suppliers');
        const container = document.getElementById('admin-supplier-list');
        container.innerHTML = suppliers.map(s => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${s.name}</div>
                    <div class="list-item-meta">${s.contact || '-'} | ${s.phone || '-'}</div>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

async function loadAdminProducts() {
    try {
        const products = await axios.get('/admin/products');
        const container = document.getElementById('admin-product-list');
        container.innerHTML = products.map(p => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${p.name}</div>
                    <div class="list-item-meta">${p.supplier_name || '-'} | ${fmtMoney(p.price * 100)}</div>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

// ========== 跳转到数据分析看板 ==========
function navigateToAdminDashboard() {
    window.location.href = 'dashboard.html';
}

// ========== 初始化 ==========
(function init() {
    if (TOKEN && CURRENT_USER.id) {
        showMainPage();
        loadDashboard();
        loadProfile();
        if (CURRENT_USER.role === 'admin') {
            document.getElementById('admin-menu-item').style.display = 'flex';
            document.getElementById('admin-dashboard-menu').style.display = 'flex';
        }
    } else {
        showAuthPage();
    }
})();

// ========== Phase 5: 第三方支付 ==========
let selectedPaymentMethod = 'wechat';
let currentPaymentId = null;
let paymentPollTimer = null;

function selectPaymentMethod(method, el) {
    selectedPaymentMethod = method;
    document.querySelectorAll('.payment-method-option').forEach(e => e.classList.remove('active'));
    if (el) el.classList.add('active');
}

async function createPayment() {
    const amountInput = document.getElementById('payment-amount');
    const amountYuan = parseFloat(amountInput.value);
    if (!amountYuan || amountYuan < 1) return showToast('请输入有效金额');

    const amount = Math.round(amountYuan * 100); // 转为分

    try {
        const data = await axios.post('/payment/create', {
            channel: selectedPaymentMethod,
            amount: amount,
            description: '账户充值'
        });

        currentPaymentId = data.id;

        // 显示二维码区域
        const qrcodeArea = document.getElementById('payment-qrcode-area');
        const qrcodeContainer = document.getElementById('payment-qrcode');
        qrcodeArea.style.display = '';
        qrcodeContainer.innerHTML = '<div class="qrcode-spinner"><div class="spinner"></div><p>正在生成支付二维码...</p></div>';

        // 轮询支付状态
        startPaymentPoll(data.payment_no);

        showToast('支付已创建，等待付款');
    } catch (e) {
        const msg = e.response?.data?.error || e.message || '创建支付失败';
        showToast(msg);
    }
}

function startPaymentPoll(paymentNo) {
    if (paymentPollTimer) clearInterval(paymentPollTimer);
    paymentPollTimer = setInterval(async () => {
        try {
            const data = await axios.get(`/payment/status/${paymentNo}`);
            if (data.status === 'paid') {
                clearInterval(paymentPollTimer);
                paymentPollTimer = null;
                showPaymentResult(true, data);
                loadDashboard();
            } else if (data.status === 'failed' || data.status === 'cancelled') {
                clearInterval(paymentPollTimer);
                paymentPollTimer = null;
                showPaymentResult(false, data);
            }
        } catch (e) {}
    }, 3000);
}

function showPaymentResult(success, data) {
    const resultArea = document.getElementById('payment-result-area');
    const resultDiv = document.getElementById('payment-result');
    resultArea.style.display = '';

    if (success) {
        resultDiv.innerHTML = `
            <div class="payment-result-success">✅</div>
            <h4>支付成功</h4>
            <p>金额：${fmtMoney(data.real_amount)}</p>
            <p>时间：${formatDate(data.created_at)}</p>
        `;
    } else {
        resultDiv.innerHTML = `
            <div class="payment-result-failed">❌</div>
            <h4>支付失败</h4>
            <p>${data.error || '支付已取消'}</p>
        `;
    }
}

async function loadPaymentList() {
    try {
        const payments = await axios.get('/payment/list', { params: { page: 1, page_size: 20 } });
        const container = document.getElementById('payment-list');
        if (!payments.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">💳</div><div class="empty-state-text">暂无支付记录</div></div>';
            return;
        }
        container.innerHTML = payments.map(p => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${fmtMoney(p.amount)}</div>
                    <div class="list-item-meta">${p.channel === 'wechat' ? '微信' : '支付宝'} | ${p.payment_no || ''}</div>
                </div>
                ${statusBadge(p.status)}
            </div>
        `).join('');
    } catch (e) {}
}

// 导航到支付页面时加载列表
const origNavigateTo = navigateTo;
navigateTo = function(section) {
    if (section === 'payment') loadPaymentList();
    origNavigateTo(section);
};

// ========== Phase 5: 灵活就业 ==========

// --- 灵活就业注册 ---
function showFreelancerRegister() {
    document.getElementById('freelancer-modal').style.display = 'flex';
}

async function submitFreelancerRegister() {
    const real_name = document.getElementById('fl-realname').value.trim();
    const id_card = document.getElementById('fl-idcard').value.trim();
    const phone = document.getElementById('fl-phone').value.trim();
    const email = document.getElementById('fl-email').value.trim();
    const skill_tags_raw = document.getElementById('fl-tags').value.trim();
    const bio = document.getElementById('fl-bio').value.trim();

    if (!real_name || !id_card) return showToast('请填写姓名和身份证');
    if (!skill_tags_raw) return showToast('请填写技能标签');

    const skill_tags = skill_tags_raw.split(/[,，]/).map(t => t.trim()).filter(Boolean);

    try {
        await axios.post('/freelancer/register', {
            real_name, id_card, phone, email, skill_tags, bio
        });
        showToast('注册成功，等待审核');
        closeModal('freelancer-modal');
        document.getElementById('fl-realname').value = '';
        document.getElementById('fl-idcard').value = '';
        document.getElementById('fl-phone').value = '';
        document.getElementById('fl-email').value = '';
        document.getElementById('fl-tags').value = '';
        document.getElementById('fl-bio').value = '';
    } catch (e) {
        showToast(e.response?.data?.error || '注册失败');
    }
}

// --- 灵活用工 Tab 切换 ---
function switchFreelanceTab(tab, btn) {
    document.querySelectorAll('.freelance-tabs-main .tab-btn').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    document.querySelectorAll('.freelance-panel').forEach(p => p.style.display = 'none');
    const panelMap = {
        'tasks': 'freelance-tasks-panel',
        'my-tasks': 'freelance-my-tasks-panel',
        'time-logs': 'freelance-time-logs-panel',
        'settlements': 'freelance-settlements-panel'
    };
    const panel = document.getElementById(panelMap[tab]);
    if (panel) panel.style.display = '';

    // 加载对应数据
    if (tab === 'tasks') loadTaskBoard();
    else if (tab === 'my-tasks') loadMyTasks();
    else if (tab === 'time-logs') loadTimeLogs();
    else if (tab === 'settlements') loadSettlements();
}

// --- 导航到灵活用工 ---
navigateTo_orig = navigateTo;
navigateTo = function(section) {
    if (section === 'freelance') loadTaskBoard();
    else if (section === 'freelance-task') loadPublishedTasks();
    navigateTo_orig(section);
};

// --- 任务大厅 ---
async function loadTaskBoard() {
    const container = document.getElementById('task-board-list');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const tasks = await axios.get('/task/list', { params: { status: 'published', page: 1, page_size: 50 } });
        if (!tasks.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📋</div><div class="empty-state-text">暂无可用任务</div></div>';
            return;
        }
        container.innerHTML = tasks.map(t => `
            <div class="task-card" onclick="showTaskDetail(${t.id})">
                <div class="task-card-header">
                    <span class="task-card-title">${t.title}</span>
                    ${statusBadge(t.status)}
                </div>
                <div class="task-card-body">${t.description || ''}</div>
                <div class="task-card-footer">
                    <span class="task-card-budget">${fmtMoney(t.budget)}</span>
                    <div class="task-card-tags">
                        ${(t.tags || '').split(',').filter(Boolean).map(tag => `<span class="task-tag">${tag.trim()}</span>`).join('')}
                    </div>
                </div>
            </div>
        `).join('');
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><div class="empty-state-text">加载失败</div></div>';
    }
}

// --- 任务详情 ---
async function showTaskDetail(taskId) {
    try {
        const task = await axios.get(`/task/${taskId}`);
        const content = document.getElementById('task-detail-content');
        content.innerHTML = `
            <h4 style="margin-bottom:12px">${task.title}</h4>
            <div class="task-detail-row"><span class="task-detail-label">状态</span><span class="task-detail-value">${statusBadge(task.status)}</span></div>
            <div class="task-detail-row"><span class="task-detail-label">预算</span><span class="task-detail-value" style="color:var(--danger-color);font-weight:700;font-size:16px">${fmtMoney(task.budget)}</span></div>
            <div class="task-detail-row"><span class="task-detail-label">描述</span><span class="task-detail-value">${task.description || '-'}</span></div>
            <div class="task-detail-row"><span class="task-detail-label">标签</span><span class="task-detail-value">${(task.tags || '-')}</span></div>
            <div class="task-detail-row"><span class="task-detail-label">截止</span><span class="task-detail-value">${task.deadline || '-'}</span></div>
            <div class="task-detail-row"><span class="task-detail-label">发布者</span><span class="task-detail-value">${task.publisher_name || '-'}</span></div>
        `;
        const actions = document.getElementById('task-detail-actions');
        if (task.status === 'published') {
            actions.innerHTML = `<button class="btn btn-primary" onclick="acceptTask(${task.id})">接受任务</button>`;
        } else {
            actions.innerHTML = '<span style="color:var(--text-secondary)">该任务暂不可接受</span>';
        }
        document.getElementById('task-detail-modal').style.display = 'flex';
    } catch (e) {
        showToast('加载任务详情失败');
    }
}

async function acceptTask(taskId) {
    if (!confirm('确认接受该任务？')) return;
    try {
        await axios.post(`/task/${taskId}/accept`);
        showToast('任务已接受');
        closeModal('task-detail-modal');
        loadTaskBoard();
    } catch (e) {
        showToast(e.response?.data?.error || '接受任务失败');
    }
}

// --- 我的任务 ---
async function loadMyTasks() {
    const container = document.getElementById('my-task-list');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const tasks = await axios.get('/task/my-tasks', { params: { page: 1, page_size: 50 } });
        if (!tasks.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><div class="empty-state-text">暂无接手的任务</div></div>';
            return;
        }
        container.innerHTML = tasks.map(t => `
            <div class="task-card" onclick="showTaskDetail(${t.id})">
                <div class="task-card-header">
                    <span class="task-card-title">${t.title}</span>
                    ${statusBadge(t.status)}
                </div>
                <div class="task-card-footer">
                    <span class="task-card-budget">${fmtMoney(t.budget)}</span>
                    <span class="list-item-meta">${t.freelancer_name || ''}</span>
                </div>
            </div>
        `).join('');
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><div class="empty-state-text">加载失败</div></div>';
    }
}

// --- 提交任务 ---
async function submitTask(taskId) {
    const description = prompt('请输入完成说明：');
    if (description === null) return;
    try {
        await axios.post(`/task/${taskId}/submit`, { description });
        showToast('任务已提交');
        loadMyTasks();
    } catch (e) {
        showToast(e.response?.data?.error || '提交失败');
    }
}

// --- 发布任务 ---
async function publishTask() {
    const title = document.getElementById('task-title').value.trim();
    const description = document.getElementById('task-desc').value.trim();
    const tags = document.getElementById('task-tags').value.trim();
    const budget = parseInt(document.getElementById('task-budget').value);
    const deadline = document.getElementById('task-deadline').value;

    if (!title || !budget) return showToast('请填写任务和预算');

    try {
        await axios.post('/task/publish', { title, description, tags, budget, deadline });
        showToast('任务发布成功');
        document.getElementById('task-title').value = '';
        document.getElementById('task-desc').value = '';
        document.getElementById('task-tags').value = '';
        document.getElementById('task-budget').value = '';
        document.getElementById('task-deadline').value = '';
        loadPublishedTasks();
    } catch (e) {
        showToast(e.response?.data?.error || '发布失败');
    }
}

async function loadPublishedTasks() {
    const container = document.getElementById('published-task-list');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const tasks = await axios.get('/task/published', { params: { page: 1, page_size: 50 } });
        if (!tasks.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📝</div><div class="empty-state-text">暂无发布的任务</div></div>';
            return;
        }
        container.innerHTML = tasks.map(t => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${t.title}</div>
                    <div class="list-item-meta">${fmtMoney(t.budget)} | ${statusBadge(t.status)}</div>
                </div>
            </div>
        `).join('');
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><div class="empty-state-text">加载失败</div></div>';
    }
}

// --- 工时记录 ---
function toggleTimeLogForm() {
    const form = document.getElementById('time-log-form');
    form.style.display = form.style.display === 'none' ? '' : 'none';
}

async function createTimeLog() {
    const task_id = parseInt(document.getElementById('timelog-task-id').value);
    const start_time = document.getElementById('timelog-start').value;
    const end_time = document.getElementById('timelog-end').value;
    const description = document.getElementById('timelog-description').value.trim();

    if (!task_id || !start_time || !end_time) return showToast('请填写必要字段');

    try {
        await axios.post('/timelog/create', { task_id, start_time, end_time, description });
        showToast('工时记录已提交');
        document.getElementById('timelog-task-id').value = '';
        document.getElementById('timelog-start').value = '';
        document.getElementById('timelog-end').value = '';
        document.getElementById('timelog-description').value = '';
        loadTimeLogs();
    } catch (e) {
        showToast(e.response?.data?.error || '提交失败');
    }
}

async function loadTimeLogs() {
    const container = document.getElementById('time-log-list');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const logs = await axios.get('/timelog/list', { params: { page: 1, page_size: 50 } });
        if (!logs.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⏱️</div><div class="empty-state-text">暂无工时记录</div></div>';
            return;
        }
        container.innerHTML = logs.map(l => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">任务 #${l.task_id}</div>
                    <div class="list-item-meta">${formatDate(l.start_time)} ~ ${formatDate(l.end_time)}</div>
                    ${l.description ? `<div class="list-item-meta">${l.description}</div>` : ''}
                </div>
                <div class="amount-positive">${l.hours ? l.hours.toFixed(1) + 'h' : ''}</div>
            </div>
        `).join('');
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><div class="empty-state-text">加载失败</div></div>';
    }
}

// --- 结算明细 ---
async function loadSettlements() {
    const container = document.getElementById('settlement-list');
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try {
        const settlements = await axios.get('/settlement/list', { params: { page: 1, page_size: 50 } });
        if (!settlements.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">💰</div><div class="empty-state-text">暂无结算记录</div></div>';
            return;
        }
        container.innerHTML = settlements.map(s => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${fmtMoney(s.amount)}</div>
                    <div class="list-item-meta">${statusBadge(s.status)}</div>
                    ${s.description ? `<div class="list-item-meta">${s.description}</div>` : ''}
                </div>
                <span class="list-item-meta">${formatDate(s.created_at)}</span>
            </div>
        `).join('');
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><div class="empty-state-text">加载失败</div></div>';
    }
}

// --- 结算确认 ---
async function confirmSettlement(settlementId) {
    if (!confirm('确认结算？')) return;
    try {
        await axios.post(`/settlement/${settlementId}/confirm`);
        showToast('结算成功');
        loadSettlements();
    } catch (e) {
        showToast(e.response?.data?.error || '操作失败');
    }
}

// ========== 评分 ==========
let currentRating = 3;
let currentRatingTaskId = null;
let currentRatingFreelancerId = null;

function setRating(score) {
    currentRating = score;
    document.querySelectorAll('.star-rating-input .star').forEach((s, i) => {
        s.classList.toggle('filled', i < score);
        s.textContent = i < score ? '★' : '☆';
    });
}

function showRatingModal(taskId, freelancerId) {
    currentRatingTaskId = taskId;
    currentRatingFreelancerId = freelancerId;
    currentRating = 5;
    setRating(5);
    document.getElementById('rating-comment').value = '';
    document.getElementById('rating-modal').style.display = 'flex';
}

async function submitRating() {
    if (!currentRatingTaskId || !currentRatingFreelancerId) return showToast('缺少必要信息');
    try {
        await axios.post('/rating/create', {
            task_id: currentRatingTaskId,
            freelancer_id: currentRatingFreelancerId,
            score: currentRating,
            comment: document.getElementById('rating-comment').value.trim()
        });
        showToast('评分成功');
        closeModal('rating-modal');
    } catch (e) {
        showToast(e.response?.data?.error || '评分失败');
    }
}

// ========== 管理后台扩展: 支付 + 灵活用工 ==========
async function loadAdminPayments() {
    try {
        const payments = await axios.get('/admin/payments', { params: { page: 1, page_size: 50 } });
        const container = document.getElementById('admin-payment-list');
        if (!payments.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">✅</div><div class="empty-state-text">暂无支付记录</div></div>';
            return;
        }
        container.innerHTML = payments.map(p => `
            <div class="list-item">
                <div>
                    <div class="list-item-title">${fmtMoney(p.amount)} (${p.channel === 'wechat' ? '微信' : '支付宝'})</div>
                    <div class="list-item-meta">${p.payment_no || ''} | 用户#${p.user_id}</div>
                    <div class="list-item-meta">${statusBadge(p.status)}</div>
                </div>
            </div>
        `).join('');
    } catch (e) {}
}

async function loadAdminFreelancers(filter) {
    try {
        const params = filter === 'pending' ? { status: 'pending', page: 1, page_size: 50 } : { page: 1, page_size: 50 };
        const freelancers = await axios.get('/admin/freelancers', { params });
        const container = document.getElementById('admin-freelancer-list');
        if (!freelancers.length) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">✅</div><div class="empty-state-text">暂无灵活用工人员</div></div>';
            return;
        }
        container.innerHTML = freelancers.map(f => `
            <div class="list-item">
                <div class="freelancer-info">
                    <div class="list-item-title">${f.real_name}</div>
                    <div class="list-item-meta">${f.phone || '-'} | ${f.skill_tags || '-'}</div>
                    <div class="list-item-meta">${statusBadge(f.status)}</div>
                </div>
                ${f.status === 'pending' ? `
                    <div class="freelancer-actions">
                        <button class="btn btn-sm btn-success" onclick="approveFreelancer(${f.id}, 'approved')">通过</button>
                        <button class="btn btn-sm btn-danger" onclick="approveFreelancer(${f.id}, 'rejected')">拒绝</button>
                    </div>
                ` : '<span class="list-item-meta">已审核</span>'}
            </div>
        `).join('');
    } catch (e) {}
}

async function approveFreelancer(id, action) {
    const remark = action === 'rejected' ? (prompt('拒绝原因：') || '拒绝') : '';
    try {
        await axios.patch(`/admin/freelancer/${id}/approve`, { action, remark });
        showToast(action === 'approved' ? '已通过' : '已拒绝');
        loadAdminFreelancers();
    } catch (e) {
        showToast(e.response?.data?.error || '操作失败');
    }
}

function switchAdminFreelanceTab(tab, btn) {
    document.querySelectorAll('.admin-freelancer-tabs .sub-tab-btn').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    loadAdminFreelancers(tab);
}

// 在管理后台加载时也加载新 Tab 数据
const origLoadAdminStats = loadAdminStats;
loadAdminStats = function() {
    if (!TOKEN || CURRENT_USER.role !== 'admin') return showToast('无权限访问');
    origLoadAdminStats.apply(this, arguments);
    loadAdminPayments();
    loadAdminFreelancers('pending');
};

// Tab 切换时也加载新面板
const origSwitchAdminTab = switchAdminTab;
switchAdminTab = function(tab, btn) {
    if (tab === 'payments') loadAdminPayments();
    else if (tab === 'freelancers') loadAdminFreelancers('pending');
    origSwitchAdminTab(tab, btn);
};

// ========== 弹窗关闭 ==========
const origCloseModal = closeModal || function() {};
window.closeModal = function(id) {
    document.getElementById(id).style.display = 'none';
    // 重置评分
    if (id === 'rating-modal') {
        currentRating = 3;
        currentRatingTaskId = null;
        currentRatingFreelancerId = null;
        setRating(3);
    }
};
