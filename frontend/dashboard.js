/**
 * Phase 7: 数据分析看板 - 前端逻辑
 * 调用后端 Dashboard API，使用 Chart.js 渲染图表
 */

// ========== 全局状态 ==========
let TOKEN = '';
let currentTimeRange = '7d';

// Chart.js 实例（用于销毁重建）
const charts = {
    revenue: null,
    userGrowth: null,
    orderStatus: null,
    topUsersBar: null,
    taskStatus: null,
    ratingTrend: null,
};

// ========== 初始化 ==========
document.addEventListener('DOMContentLoaded', () => {
    TOKEN = localStorage.getItem('token') || '';
    if (!TOKEN) {
        window.location.href = 'index.html';
        return;
    }
    loadData();
});

// ========== 数据加载 ==========
async function loadData() {
    showLoading(true);
    try {
        await Promise.all([
            loadOverview(),
            loadTopUsers(),
            loadFreelanceStats(),
        ]);
    } catch (err) {
        console.error('[Dashboard] 加载失败:', err);
        showToast('数据加载失败: ' + err.message);
    } finally {
        showLoading(false);
    }
}

// 概览数据（统计卡片 + 三个图表）
async function loadOverview() {
    // 并发请求所有概览 API
    const [statsRes, trendRes, growthRes, orderRes] = await Promise.all([
        apiGet('/admin/dashboard/stats'),
        apiGet(`/admin/dashboard/revenue-trend?days=${getTimeRangeDays()}`),
        apiGet(`/admin/dashboard/user-growth?days=${getTimeRangeDays()}`),
        apiGet('/admin/dashboard/order-stats'),
    ]);

    // 渲染统计卡片
    renderStatCards(statsRes.data || {});

    // 渲染收益趋势图
    renderRevenueTrendChart(trendRes.data || []);

    // 渲染用户增长图
    renderUserGrowthChart(growthRes.data || []);

    // 渲染订单状态饼图
    renderOrderStatusChart(orderRes.data || {});
}

// Top 用户排行
async function loadTopUsers() {
    try {
        const res = await apiGet('/admin/dashboard/top-users?limit=10');
        const users = res.data || [];
        renderTopUsersTable(users);
        renderTopUsersBarChart(users);
    } catch (err) {
        console.warn('[Dashboard] Top用户加载失败:', err.message);
    }
}

// 自由职业者统计
async function loadFreelanceStats() {
    try {
        const res = await apiGet('/admin/dashboard/freelance-stats');
        const stats = res.data || {};
        renderFreelanceStats(stats);
        renderTaskStatusChart(stats.taskStatus || {});
        renderRatingTrendChart(stats.ratingTrend || []);
    } catch (err) {
        console.warn('[Dashboard] 自由职业者统计加载失败:', err.message);
    }
}

// ========== 渲染函数 ==========

// 渲染统计卡片
function renderStatCards(data) {
    const container = document.getElementById('stats-cards');
    const cards = [
        { icon: '👥', label: '总用户数', value: data.total_users || 0, change: data.user_growth_rate || '0%' },
        { icon: '🛒', label: '总订单数', value: data.total_orders || 0, change: data.order_growth_rate || '0%' },
        { icon: '💰', label: '总收益（元）', value: formatMoney(data.total_revenue || 0), change: data.revenue_growth_rate || '0%' },
        { icon: '📈', label: '今日收益（元）', value: formatMoney(data.today_revenue || 0) },
        { icon: '💸', label: '今日订单', value: data.today_orders || 0 },
        { icon: '🏦', label: '待审核提现', value: data.pending_withdrawals || 0 },
    ];

    container.innerHTML = cards.map(card => `
        <div class="stat-card">
            <div class="stat-icon">${card.icon}</div>
            <div class="stat-label">${card.label}</div>
            <div class="stat-value">${card.value}</div>
            ${card.change ? `<div class="stat-change ${card.change.startsWith('+') ? 'positive' : ''}">${card.change}</div>` : ''}
        </div>
    `).join('');
}

// 收益趋势图
function renderRevenueTrendChart(data) {
    const ctx = document.getElementById('revenue-chart').getContext('2d');
    if (charts.revenue) charts.revenue.destroy();

    const labels = data.map(d => d.day);
    const amounts = data.map(d => d.amount_yuan);

    charts.revenue = new Chart(ctx, {
        type: 'line',
        data: {
            labels,
            datasets: [{
                label: '日收益（元）',
                data: amounts,
                borderColor: '#4f46e5',
                backgroundColor: 'rgba(79, 70, 229, 0.1)',
                fill: true,
                tension: 0.3,
                pointRadius: 3,
                pointHoverRadius: 5,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { beginAtZero: true, ticks: { callback: v => '¥' + v } },
            }
        }
    });
}

// 用户增长图
function renderUserGrowthChart(data) {
    const ctx = document.getElementById('user-growth-chart').getContext('2d');
    if (charts.userGrowth) charts.userGrowth.destroy();

    const labels = data.map(d => d.day);
    const counts = data.map(d => d.count);

    charts.userGrowth = new Chart(ctx, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                label: '新增用户',
                data: counts,
                backgroundColor: 'rgba(79, 70, 229, 0.7)',
                borderRadius: 4,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { beginAtZero: true },
            }
        }
    });
}

// 订单状态饼图
function renderOrderStatusChart(data) {
    const ctx = document.getElementById('order-status-chart').getContext('2d');
    if (charts.orderStatus) charts.orderStatus.destroy();

    const statusCounts = data.status_counts || {};
    const labels = Object.keys(statusCounts);
    const values = labels.map(k => statusCounts[k] || 0);
    const colors = ['#dc2626', '#16a34a', '#2563eb', '#f59e0b', '#6b7280'];

    charts.orderStatus = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: labels.length ? labels : ['暂无数据'],
            datasets: [{
                data: values.length ? values : [1],
                backgroundColor: colors.slice(0, labels.length),
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { position: 'bottom' },
            }
        }
    });
}

// Top 用户表格
function renderTopUsersTable(users) {
    const tbody = document.querySelector('#top-users-table tbody');
    const emptyEl = document.getElementById('top-users-empty');

    if (!users.length) {
        tbody.parentElement.style.display = 'none';
        emptyEl.style.display = 'block';
        return;
    }

    tbody.parentElement.style.display = '';
    emptyEl.style.display = 'none';

    tbody.innerHTML = users.map((u, i) => {
        const rank = i + 1;
        const rankClass = rank <= 3 ? `rank-${rank}` : 'rank-default';
        return `
            <tr>
                <td><span class="rank-badge ${rankClass}">${rank}</span></td>
                <td>${u.username || '-'}</td>
                <td>¥${formatMoneyYuan(u.total_profit || 0)}</td>
                <td>${u.team_size || 0}</td>
                <td>${u.order_count || 0}</td>
            </tr>
        `;
    }).join('');
}

// Top 用户柱状图
function renderTopUsersBarChart(users) {
    if (!users || !users.length) return;

    const ctx = document.getElementById('top-users-bar-chart').getContext('2d');
    if (charts.topUsersBar) charts.topUsersBar.destroy();

    const labels = users.map(u => u.username || `User#${u.user_id}`);
    const amounts = users.map(u => (u.total_profit || 0) / 100); // 分转元

    charts.topUsersBar = new Chart(ctx, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                label: '累计收益（元）',
                data: amounts,
                backgroundColor: 'rgba(79, 70, 229, 0.6)',
                borderRadius: 4,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            indexAxis: 'y',
            plugins: { legend: { display: false } },
            scales: {
                x: { ticks: { callback: v => '¥' + v } },
            }
        }
    });
}

// 自由职业者统计卡片
function renderFreelanceStats(stats) {
    const container = document.getElementById('freelance-stats-cards');
    const cards = [
        { icon: '📝', label: '总任务数', value: stats.total_tasks || 0 },
        { icon: '✅', label: '已完成', value: stats.completed_tasks || 0 },
        { icon: '⏳', label: '进行中', value: stats.in_progress_tasks || 0 },
        { icon: '⭐', label: '平均评分', value: (stats.avg_rating || 0).toFixed(1) },
    ];

    container.innerHTML = cards.map(card => `
        <div class="stat-card">
            <div class="stat-icon">${card.icon}</div>
            <div class="stat-label">${card.label}</div>
            <div class="stat-value">${card.value}</div>
        </div>
    `).join('');
}

// 任务状态分布图
function renderTaskStatusChart(statuses) {
    const ctx = document.getElementById('task-status-chart').getContext('2d');
    if (charts.taskStatus) charts.taskStatus.destroy();

    const labels = Object.keys(statuses);
    const values = labels.map(k => statuses[k] || 0);
    const colors = ['#4f46e5', '#16a34a', '#f59e0b', '#dc2626', '#6b7280'];

    charts.taskStatus = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: labels.length ? labels : ['暂无数据'],
            datasets: [{
                data: values.length ? values : [1],
                backgroundColor: colors.slice(0, labels.length),
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'bottom' } },
        }
    });
}

// 评分趋势图
function renderRatingTrendChart(data) {
    const ctx = document.getElementById('rating-trend-chart').getContext('2d');
    if (charts.ratingTrend) charts.ratingTrend.destroy();

    const labels = data.map(d => d.date);
    const ratings = data.map(d => parseFloat(d.avg_rating) || 0);

    charts.ratingTrend = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels.length ? labels : ['暂无数据'],
            datasets: [{
                label: '平均评分',
                data: ratings.length ? ratings : [0],
                borderColor: '#f59e0b',
                backgroundColor: 'rgba(245, 158, 11, 0.1)',
                fill: true,
                tension: 0.3,
                pointRadius: 4,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { min: 0, max: 5, ticks: { stepSize: 1 } },
            }
        }
    });
}

// ========== 工具函数 ==========

// API 请求
function apiGet(url) {
    return axios.get(`http://127.0.0.1:8080${url}`, {
        headers: { 'Authorization': `Bearer ${TOKEN}` },
    });
}

// 金额格式化（分转元）
function formatMoneyYuan(cents) {
    return (cents / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

// 旧接口兼容
function formatMoney(cents) {
    if (typeof cents === 'string' && cents.includes('growth')) return cents;
    return formatMoneyYuan(cents);
}

// 获取时间范围天数
function getTimeRangeDays() {
    switch (currentTimeRange) {
        case '7d': return 7;
        case '30d': return 30;
        case '90d': return 90;
        default: return 7;
    }
}

// 切换时间范围
function changeTimeRange(range, btn) {
    currentTimeRange = range;
    document.querySelectorAll('.time-filter-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    loadOverview(); // 重新加载概览数据
}

// 显示/隐藏加载状态
function showLoading(show) {
    const loadingEl = document.getElementById('dashboard-loading');
    const tabs = document.querySelectorAll('.dash-tab-content');
    if (show) {
        loadingEl.style.display = 'flex';
        tabs.forEach(t => t.style.display = 'none');
    } else {
        loadingEl.style.display = 'none';
        document.getElementById('dash-overview-tab').classList.add('active');
    }
}

// 返回上一页
function goBack() {
    window.location.href = 'index.html';
}

// Toast 提示
function showToast(msg) {
    const div = document.createElement('div');
    div.textContent = msg;
    div.style.cssText = `
        position: fixed; top: 20px; right: 20px; z-index: 9999;
        background: #dc2626; color: #fff; padding: 12px 20px;
        border-radius: 8px; font-size: 14px; box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        animation: slideInRight 0.3s ease;
    `;
    document.body.appendChild(div);
    setTimeout(() => div.remove(), 3000);
}

// 初始化 Chart.js 动画 CSS
(function injectAnimationCSS() {
    const style = document.createElement('style');
    style.textContent = `
        @keyframes slideInRight {
            from { transform: translateX(100%); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
    `;
    document.head.appendChild(style);
})();
