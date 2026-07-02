// Chain2Plus1 Performance Load Test
// Usage: k6 run benchmark/load-test.js
// Requires: Go backend running on localhost:8080

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const failRate = new Rate('failed_requests');
const loginTrend = new Trend('login_duration');
const registerTrend = new Trend('register_duration');
const orderTrend = new Trend('order_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '10s', target: 10 },   // Ramp up to 10 VUs
    { duration: '20s', target: 20 },   // Ramp to 20 VUs
    { duration: '10s', target: 0 },    // Ramp down
  ],
  thresholds: {
    failed_requests: ['rate<0.05'],    // < 5% failure rate
    http_req_duration: ['p(95)<2000'], // 95% of requests < 2s
    login_duration: ['p(95)<1000'],
    register_duration: ['p(95)<1500'],
  },
};

const BASE_URL = 'http://localhost:8080/api/v1';

// Shared state across VUs
const sharedUsers = [];
let userCounter = 0;

export default function () {
  group('Core Business Flow', () => {
    // 1. Register
    const username = `loadtest_${__VU}_${__ITER}`;
    const password = 'Test123456';
    
    const registerRes = http.post(`${BASE_URL}/auth/register`, JSON.stringify({
      username,
      password,
      phone: `1380000${String(__VU).padStart(4, '0')}`,
      email: `${username}@test.com`,
    }), { headers: { 'Content-Type': 'application/json' } });

    registerTrend.add(registerRes.timings.duration);
    const regOk = check(registerRes, {
      'register status 201': (r) => r.status === 201,
      'register has message': (r) => r.json('message') === '注册成功',
    });
    failRate.add(!regOk);
    sleep(0.5);

    // 2. Login
    const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
      username,
      password,
    }), { headers: { 'Content-Type': 'application/json' } });

    loginTrend.add(loginRes.timings.duration);
    const loginOk = check(loginRes, {
      'login status 200': (r) => r.status === 200,
      'login has token': (r) => r.json('token') !== undefined,
    });
    failRate.add(!loginOk);

    if (!loginOk) return;

    const token = loginRes.json('token');
    const authHeaders = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    };

    sleep(0.5);

    // 3. Get profile
    const profileRes = http.get(`${BASE_URL}/user/profile`, { headers: authHeaders });
    check(profileRes, {
      'profile status 200': (r) => r.status === 200,
      'profile has user': (r) => r.json('user') !== undefined,
    });
    sleep(0.3);

    // 4. List orders
    const ordersRes = http.get(`${BASE_URL}/order/list`, { headers: authHeaders });
    check(ordersRes, {
      'orders status 200': (r) => r.status === 200,
    });
    sleep(0.3);

    // 5. List profits
    const profitsRes = http.get(`${BASE_URL}/profit/list`, { headers: authHeaders });
    check(profitsRes, {
      'profits status 200': (r) => r.status === 200,
    });
    sleep(0.3);

    // 6. Leaderboard
    const lbRes = http.get(`${BASE_URL}/leaderboard/total_earned`, { headers: authHeaders });
    check(lbRes, {
      'leaderboard status 200': (r) => r.status === 200,
    });
    sleep(0.5);
  });

  // Admin flow (only VU 1)
  if (__VU === 1) {
    group('Admin Flow', () => {
      const adminLogin = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
        username: 'admin',
        password: 'Admin@2024',
      }), { headers: { 'Content-Type': 'application/json' } });

      if (!adminLogin.json('token')) return;
      const adminToken = adminLogin.json('token');
      const adminHeaders = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${adminToken}`,
      };

      // Admin stats
      const statsRes = http.get(`${BASE_URL}/admin/stats`, { headers: adminHeaders });
      check(statsRes, { 'admin stats 200': (r) => r.status === 200 });
      sleep(0.3);

      // Admin users
      const usersRes = http.get(`${BASE_URL}/admin/users`, { headers: adminHeaders });
      check(usersRes, { 'admin users 200': (r) => r.status === 200 });
      sleep(0.3);

      // Admin withdraws
      const wdRes = http.get(`${BASE_URL}/admin/withdraw`, { headers: adminHeaders });
      check(wdRes, { 'admin withdraw 200': (r) => r.status === 200 });
      sleep(0.3);

      // Dashboard stats
      const dashRes = http.get(`${BASE_URL}/admin/dashboard/stats`, { headers: adminHeaders });
      check(dashRes, { 'dashboard stats 200': (r) => r.status === 200 });
      sleep(0.3);

      // Agent report (user ID 1)
      const agentRes = http.get(`${BASE_URL}/admin/agent-report/1`, { headers: adminHeaders });
      check(agentRes, { 'agent report 200': (r) => r.status === 200 });
      sleep(0.5);
    });
  }
}
