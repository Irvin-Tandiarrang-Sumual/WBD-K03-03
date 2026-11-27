import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 1: Authentication & Access Control (REG)', () => {
  test('TC-E2E-REG-001: Verify auth flow, role navigation guards, and unauthorized route redirection', async ({
    page,
  }) => {
    // 1. Access protected route unauthenticated
    await page.goto('/databases');
    await expect(page).toHaveURL(/\/login/);

    // 2. Login as admin
    await page.locator('#login-username').fill('admin');
    await page.locator('#login-password').fill('admin123');
    await page.locator('button[type="submit"]').click();

    // 3. Verify redirected to /databases and Administration section is visible
    await expect(page).toHaveURL(/\/databases/);
    const openMenu = page.getByRole('button', { name: 'Open navigation menu' });
    const mobileNavigation = await openMenu.isVisible();
    if (mobileNavigation) {
      await openMenu.click();
    }
    const sidebar = mobileNavigation
      ? page.getByLabel('Mobile navigation drawer')
      : page.getByRole('navigation', { name: 'Sidebar Navigation' }).first();
    await expect(sidebar.getByText('Administration', { exact: true })).toBeVisible();
    await expect(sidebar.locator('a[href="/admin/users"]')).toBeVisible();
    await expect(sidebar.locator('a[href="/admin/groups"]')).toBeVisible();
    await expect(sidebar.locator('a[href="/admin/databases"]')).toBeVisible();

    // 4. Logout
    const logout = mobileNavigation
      ? sidebar.getByRole('button', { name: 'Sign out' })
      : page.getByRole('button', { name: 'Sign out' }).first();
    await logout.click();
    await expect(page).toHaveURL(/\/login/);

    // 5. Login as regular user
    await page.locator('#login-username').fill('user');
    await page.locator('#login-password').fill('user123');
    await page.locator('button[type="submit"]').click();
    await expect(page).toHaveURL(/\/databases/);

    // 6. Verify Administration section is NOT visible
    await expect(page.locator('text=Administration')).not.toBeVisible();

    // 7. Try accessing /admin/users directly
    await page.goto('/admin/users');
    // Navigation guard must redirect back to /databases or show 403
    await expect(page).toHaveURL(/\/databases/);
  });

  test('TC-E2E-REG-002: Verify user with view_only access has mutation buttons disabled and API rejects with 403', async ({
    auditorLeadPage,
  }) => {
    // 1. auditor_lead navigates to db1_pg (database id 1)
    await auditorLeadPage.goto('/databases/1');

    // 2. Verify mutation buttons are disabled
    const proposeBtn = auditorLeadPage.getByRole('link', { name: 'Propose Migration Draft' });
    const rollbackBtn = auditorLeadPage.locator('button:has-text("Rollback (1-Step)")');
    const forwardBtn = auditorLeadPage.locator('button:has-text("Forward (Redo)")');

    await expect(proposeBtn).toHaveAttribute('href', '/databases/1');
    if (await rollbackBtn.count() > 0) {
      await expect(rollbackBtn).toBeDisabled();
    }
    if (await forwardBtn.count() > 0) {
      await expect(forwardBtn).toBeDisabled();
    }

    // 3. Directly invoke API mutation with auditor_lead session
    const apiURL = process.env.VITE_API_URL || 'http://localhost:8080/api/v1';
    await auditorLeadPage.waitForFunction(() => sessionStorage.getItem('csrf_token'));
    const status = await auditorLeadPage.evaluate(async (url) => {
      const response = await fetch(`${url}/me/databases/1/drafts`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': sessionStorage.getItem('csrf_token') || '',
        },
        body: JSON.stringify({ sqlUp: 'CREATE TABLE auditor_test (id INT PRIMARY KEY);' }),
      });
      return response.status;
    }, apiURL);

    // 4. Assert backend rejects with 403 Forbidden
    expect(status).toBe(403);
  });

  test('TC-E2E-REG-003: Verify clearing session cookie triggers clean redirect to /login', async ({
    userPage,
  }) => {
    // 1. User is on databases page
    await userPage.goto('/databases');
    await expect(userPage).toHaveURL(/\/databases/);

    // 2. Clear browser session cookies
    await userPage.context().clearCookies();

    // 3. Trigger SPA navigation or data refresh
    await userPage.goto('/console');

    // 4. Client loader / auth guard redirects to /login
    await expect(userPage).toHaveURL(/\/login/);
  });
});
