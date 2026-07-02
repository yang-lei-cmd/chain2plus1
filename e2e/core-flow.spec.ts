// @ts-check
import { test, expect } from '@playwright/test';

const TEST_USER = `e2e_${Date.now()}`;
const TEST_PASS = 'E2ETest123';

test.describe('P0: 核心业务流 E2E', () => {

  test('1. 用户注册', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.auth-container');

    // Switch to register
    await page.click('text=立即注册');
    await page.fill('#reg-username', TEST_USER);
    await page.fill('#reg-password', TEST_PASS);
    await page.fill('#reg-phone', '13800000100');
    await page.fill('#reg-email', `${TEST_USER}@test.com`);
    await page.click('button:has-text("注册")');

    // Wait for success toast and redirect to login
    await expect(page.locator('.toast-success')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('h2:has-text("用户登录")')).toBeVisible();
  });

  test('2. 用户登录', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.auth-container');

    await page.fill('#login-username', TEST_USER);
    await page.fill('#login-password', TEST_PASS);
    await page.click('button:has-text("登录")');

    // Wait for main app to show
    await expect(page.locator('.app-layout')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.dashboard-cards')).toBeVisible();
  });

  test('3. 首页展示', async ({ page }) => {
    await page.goto('/');
    // Login first
    await page.fill('#login-username', TEST_USER);
    await page.fill('#login-password', TEST_PASS);
    await page.click('button:has-text("登录")');
    await page.waitForSelector('.app-layout');

    // Verify dashboard cards
    await expect(page.locator('.dashboard-cards')).toBeVisible();
    const cards = await page.locator('.card-value').allTextContents();
    expect(cards.length).toBe(4); // balance, earnings, level, invite code
  });

  test('4. 导航菜单', async ({ page }) => {
    await page.goto('/');
    await page.fill('#login-username', TEST_USER);
    await page.fill('#login-password', TEST_PASS);
    await page.click('button:has-text("登录")');
    await page.waitForSelector('.app-layout');

    // Check bottom nav has required items
    const navLabels = await page.locator('.nav-label').allTextContents();
    expect(navLabels).toContain('首页');
    expect(navLabels).toContain('下单');
    expect(navLabels).toContain('提现');
    expect(navLabels).toContain('收益');
    expect(navLabels).toContain('邀请');
    expect(navLabels).toContain('任务');
  });

  test('5. 分享页面', async ({ page }) => {
    await page.goto('/');
    await page.fill('#login-username', TEST_USER);
    await page.fill('#login-password', TEST_PASS);
    await page.click('button:has-text("登录")');
    await page.waitForSelector('.app-layout');

    // Navigate to share page
    await page.click('text=邀请');
    await expect(page.locator('text=邀请好友加入')).toBeVisible();
    await expect(page.locator('.share-code')).toBeVisible();
  });
});

test.describe('P1: 管理后台 E2E', () => {
  test('管理员登录', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.auth-container');

    await page.fill('#login-username', 'admin');
    await page.fill('#login-password', 'Admin@2024');
    await page.click('button:has-text("登录")');

    await page.waitForSelector('.app-layout');
    // Admin should see "管理" nav item
    await expect(page.locator('text=管理')).toBeVisible();
  });
});
