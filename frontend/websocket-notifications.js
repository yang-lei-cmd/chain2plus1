// ========== WebSocket 实时通知模块 (Phase 6) ==========
let ws = null;
let wsReconnectTimer = null;
const NOTIFICATION_QUEUE = []; // 通知队列

/**
 * 连接到 WebSocket 通知服务
 */
function connectWebSocket() {
    if (!TOKEN) return; // 未登录不连接
    
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.hostname || '127.0.0.1';
    const port = window.location.port || '8080';
    const wsUrl = `${proto}//${host}:${port}/ws?token=${TOKEN}`;
    
    try {
        ws = new WebSocket(wsUrl);
    } catch (e) {
        console.error('[WS] 连接创建失败:', e);
        scheduleReconnect();
        return;
    }
    
    ws.onopen = function() {
        console.log('[WS] 连接成功');
        showNotificationIndicator(true);
        
        // 发送一条心跳保持连接
        ws.send(JSON.stringify({ type: 'ping' }));
    };
    
    ws.onmessage = function(event) {
        try {
            const notification = JSON.parse(event.data);
            handleNotification(notification);
        } catch (e) {
            console.error('[WS] 消息解析失败:', e);
        }
    };
    
    ws.onerror = function(error) {
        console.error('[WS] 连接错误:', error);
    };
    
    ws.onclose = function(event) {
        console.log('[WS] 连接关闭:', event.code, event.reason);
        showNotificationIndicator(false);
        
        // 如果不是正常关闭，则安排重连
        if (event.code !== 1000) {
            scheduleReconnect();
        }
    };
}

/**
 * 调度重连（延迟指数退避）
 */
function scheduleReconnect() {
    if (wsReconnectTimer) return;
    
    const delay = Math.min(1000 * Math.pow(2, ws?.bufferedAmount || 1), 30000);
    console.log(`[WS] ${delay}ms 后尝试重连...`);
    
    wsReconnectTimer = setTimeout(() => {
        wsReconnectTimer = null;
        connectWebSocket();
    }, delay);
}

/**
 * 处理收到的通知
 */
function handleNotification(notification) {
    console.log('[WS] 收到通知:', notification);
    
    // 添加到通知队列
    NOTIFICATION_QUEUE.push(notification);
    
    // 显示通知
    showNotification(notification);
    
    // 更新页面元素（如果相关）
    refreshCurrentViewIfNeeded(notification);
}

/**
 * 显示通知弹窗
 */
function showNotification(notification) {
    const { type, message, data } = notification;
    
    // 创建通知元素
    const notifDiv = document.createElement('div');
    notifDiv.className = 'ws-notification';
    notifDiv.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        max-width: 400px;
        padding: 16px;
        background: #fff;
        border-left: 4px solid #4CAF50;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 10000;
        animation: slideInRight 0.3s ease;
        font-size: 14px;
    `;
    
    // 根据事件类型设置图标
    const icons = {
        'task_assigned': '\u{1F4CB}',
        'work_submitted': '\u{1F4DD}',
        'task_approved': '\u2705',
        'task_rejected': '\u274C',
        'task_published': '\u{1F4E3}',
        'payment_sent': '\u{1F4B0}',
        'rating_created': '\u2B50',
        'freelancer_approved': '\u2705',
        'freelancer_rejected': '\u274C'
    };
    
    const icon = icons[type] || '\u2139\uFE0F';
    
    notifDiv.innerHTML = `
        <div style="display:flex;align-items:start;gap:12px;">
            <span style="font-size:24px;">${icon}</span>
            <div style="flex:1;">
                <div style="font-weight:bold;margin-bottom:4px;">${getMessage(type)}</div>
                <div style="color:#666;font-size:13px;">${message}</div>
                ${data?.task_title ? `<div style="margin-top:8px;padding:8px;background:#f5f5f5;border-radius:4px;font-size:12px;">
                    <strong>任务:</strong> ${data.task_title}<br>
                    <strong>ID:</strong> ${data.task_id || '-'}
                </div>` : ''}
            </div>
            <button onclick="this.parentElement.parentElement.remove()" style="background:none;border:none;cursor:pointer;font-size:18px;color:#999;">&times;</button>
        </div>
    `;
    
    document.body.appendChild(notifDiv);
    
    // 5秒后自动消失
    setTimeout(() => {
        notifDiv.style.animation = 'slideOutRight 0.3s ease';
        setTimeout(() => notifDiv.remove(), 300);
    }, 5000);
}

/**
 * 获取中文事件名称
 */
function getMessage(type) {
    const messages = {
        'task_assigned': '新任务分配',
        'work_submitted': '工作成果已提交',
        'task_approved': '审核通过',
        'task_rejected': '审核未通过',
        'task_published': '新任务发布',
        'payment_sent': '款项已发送',
        'rating_created': '收到新评分',
        'freelancer_approved': '身份审核通过',
        'freelancer_rejected': '身份审核未通过'
    };
    return messages[type] || '系统通知';
}

/**
 * 显示通知指示器
 */
function showNotificationIndicator(connected) {
    const indicator = document.getElementById('ws-indicator');
    if (!indicator) return;
    
    if (connected) {
        indicator.style.display = 'block';
        indicator.style.background = '#4CAF50';
    } else {
        indicator.style.background = '#f44336';
    }
}

/**
 * 根据通知刷新当前视图
 */
function refreshCurrentViewIfNeeded(notification) {
    const { type } = notification;
    
    // 如果是任务相关的通知，刷新任务列表
    if (['task_assigned', 'work_submitted', 'task_approved', 'task_rejected'].includes(type)) {
        if (document.getElementById('task-list')) {
            loadMyTasks();
        }
    }
    
    // 如果是评分通知，刷新评分
    if (type === 'rating_created') {
        if (document.getElementById('rating-list')) {
            loadRatings();
        }
    }
}

// ========== CSS 动画 ==========
const style = document.createElement('style');
style.textContent = `
    @keyframes slideInRight {
        from { transform: translateX(400px); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOutRight {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(400px); opacity: 0; }
    }
`;
document.head.appendChild(style);

// ========== 页面加载时连接 WS ==========
document.addEventListener('DOMContentLoaded', () => {
    // 监听登录/登出事件
    const observer = new MutationObserver(() => {
        const newToken = localStorage.getItem('token');
        if (newToken !== TOKEN) {
            TOKEN = newToken;
            if (TOKEN) {
                connectWebSocket();
            } else {
                disconnectWebSocket();
            }
        }
    });
    observer.observe(document, { childList: true, subtree: true });
    
    // 如果已登录，立即连接
    if (TOKEN) {
        connectWebSocket();
    }
});

/**
 * 断开 WebSocket 连接
 */
function disconnectWebSocket() {
    if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer);
        wsReconnectTimer = null;
    }
    if (ws) {
        ws.close(1000, 'User logout');
        ws = null;
    }
    console.log('[WS] 已断开连接');
}
