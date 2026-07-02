// ========== 类型定义 ==========

export interface User {
  id: number;
  username: string;
  phone: string;
  email: string;
  role: string;
  invite_code: string;
  level: number;
  status: number;
  balance: number;   // 元 (float)
  total_earned: number;
}

export interface LoginRes {
  token: string;
  user: User;
}

export interface Order {
  id: number;
  user_id: number;
  product_id: number;
  product_name: string;
  order_no: string;
  amount: number;
  status: string;
  payment_method: string;
  created_at: string;
}

export interface ProfitShare {
  id: number;
  from_user_id: number;
  to_user_id: number;
  order_id: number;
  level: number;
  amount: number;
  type: string;
  status: string;
  description: string;
  created_at: string;
}

export interface Withdraw {
  id: number;
  user_id: number;
  amount: number;
  fee: number;
  actual_amount: number;
  bank_name: string;
  account_name: string;
  account_no: string;
  status: string;
  remark: string;
  created_at: string;
}

export interface Payment {
  id: number;
  payment_no: string;
  channel: string;
  amount: number;
  status: string;
  created_at: string;
}

export interface Freelancer {
  id: number;
  user_id: number;
  real_name: string;
  skill_tags: string[];
  bio: string;
  avg_rating: number;
  total_jobs: number;
  total_earnings: number;
  status: string;
}

export interface FreelanceTask {
  id: number;
  title: string;
  description: string;
  category: string;
  budget: number;
  status: string;
  publisher_id: number;
  assigned_to: number | null;
  created_at: string;
}

export interface Pagination {
  total: number;
  page: number;
  pages: number;
}

// ========== Typed API Client ==========

const BASE_URL = '/api/v1';

class ApiError extends Error {
  constructor(
    public status: number,
    public body: any,
  ) {
    super(body?.error || body?.message || `API Error ${status}`);
    this.name = 'ApiError';
  }
}

function getToken(): string {
  try {
    const stored = localStorage.getItem('chain2plus1_token');
    return stored || '';
  } catch {
    return '';
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    if (res.status === 401) {
      localStorage.removeItem('chain2plus1_token');
      localStorage.removeItem('chain2plus1_user');
      window.location.hash = '/login';
    }
    throw new ApiError(res.status, body);
  }

  return res.json();
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'POST', body: data ? JSON.stringify(data) : undefined }),
  patch: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'PATCH', body: data ? JSON.stringify(data) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),

  // ---- Auth ----
  login: (username: string, password: string) =>
    api.post<LoginRes>('/auth/login', { username, password }),

  register: (data: {
    username: string;
    password: string;
    phone?: string;
    email?: string;
    invite_code?: string;
  }) => api.post<{ message: string; user: User }>('/auth/register', data),

  // ---- User ----
  getProfile: () => api.get<{ user: User; children_count: number }>('/user/profile'),
  getUserTree: () => api.get<{ tree: any }>('/user/tree'),

  // ---- Orders ----
  createOrder: (product_id: number, payment_method: string) =>
    api.post<{ message: string; order: Order; commissions: any[] }>('/order/create', {
      product_id,
      payment_method,
    }),
  listOrders: (page = 1) => api.get<{ orders: Order[]; pagination: Pagination }>(`/order/list?page=${page}`),

  // ---- Profits ----
  listProfits: (page = 1) =>
    api.get<{ profits: ProfitShare[]; pagination: Pagination }>(`/profit/list?page=${page}`),

  // ---- Withdraw ----
  applyWithdraw: (data: {
    amount: number;
    bank_name: string;
    account_name: string;
    account_no: string;
  }) => api.post<{ message: string; withdraw: Withdraw }>('/withdraw/apply', data),
  listWithdraws: (page = 1) =>
    api.get<{ withdraws: Withdraw[]; pagination: Pagination }>(`/withdraw/list?page=${page}`),
  recharge: (amount: number, payment_mode: string) =>
    api.post<{ message: string; balance: number }>('/recharge', { amount, payment_mode }),

  // ---- Leaderboard ----
  getLeaderboard: (type: 'total_earned' | 'team_size' | 'recharge') =>
    api.get<{ leaderboard: any[] }>(`/leaderboard/${type}`),

  // ---- Payment ----
  createPayment: (data: {
    order_id: number;
    channel: string;
    amount: number;
    subject: string;
    notify_url: string;
  }) => api.post<{ message: string; payment: any }>('/payment/create', data),
  queryPayment: (paymentNo: string) =>
    api.get<{ payment: any }>(`/payment/status/${paymentNo}`),
  listPayments: (page = 1) =>
    api.get<{ payments: any[]; pagination: Pagination }>(`/payment/my-payments?page=${page}`),

  // ---- Freelance ----
  registerFreelancer: (data: {
    real_name: string;
    id_card: string;
    phone: string;
    email: string;
    skill_tags: string[];
    bio: string;
  }) => api.post<{ message: string; freelancer: Freelancer }>('/freelancer/register', data),
  createTask: (data: {
    title: string;
    description: string;
    category: string;
    budget: number;
    duration_hours: number;
    skill_tags?: string[];
  }) => api.post<{ message: string; task: FreelanceTask }>('/task/create', data),
  listTasks: (params?: { status?: string; page?: number }) => {
    const q = new URLSearchParams();
    if (params?.status) q.set('status', params.status);
    if (params?.page) q.set('page', String(params.page));
    return api.get<{ tasks: FreelanceTask[]; pagination: Pagination }>(`/task/list?${q}`);
  },
  assignTask: (task_id: number, freelancer_id: number) =>
    api.post<{ message: string }>('/task/assign', { task_id, freelancer_id }),

  // ---- Admin ----
  getAdminStats: () => api.get<{ stats: any }>('/admin/stats'),
  listAdminUsers: (page = 1) =>
    api.get<{ users: any[]; pagination: Pagination }>(`/admin/users?page=${page}`),
  listAdminWithdraws: (page = 1) =>
    api.get<{ withdraws: Withdraw[]; pagination: Pagination }>(`/admin/withdraw/list?page=${page}`),
  approveWithdraw: (id: number, action: string, remark = '') =>
    api.patch<{ message: string }>(`/admin/withdraw/${id}/approve`, { action, remark }),
  listAdminOrders: (page = 1) =>
    api.get<{ orders: any[]; pagination: Pagination }>(`/admin/orders?page=${page}`),

  // ---- Dashboard ----
  getDashboardStats: () => api.get<{ stats: any }>('/admin/dashboard/stats'),
  getRevenueTrend: (days = 7) =>
    api.get<{ trend: any[] }>(`/admin/dashboard/revenue-trend?days=${days}`),
  getUserGrowth: (days = 7) =>
    api.get<{ growth: any[] }>(`/admin/dashboard/user-growth?days=${days}`),

  // ---- CSV Export ----
  getExportURL: (type: 'profits' | 'orders' | 'withdraws', start?: string, end?: string) => {
    const params = new URLSearchParams();
    if (start) params.set('start', start);
    if (end) params.set('end', end);
    return `${BASE_URL}/admin/export/${type}?${params}`;
  },

  // ---- Agent Report ----
  getAgentReport: (userId: number) =>
    api.get<{ user: any; team_stats: any; earnings: any; top_children: any[] }>(`/admin/agent-report/${userId}`),
  getTeamTree: (userId: number, depth = 3) =>
    api.get<{ tree: any; depth: number }>(`/admin/team-tree/${userId}?depth=${depth}`),
};
