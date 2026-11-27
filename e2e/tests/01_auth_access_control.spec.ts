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
    await expect(page.locator('text=Administration')).toBeVisible();
    await expect(page.locator('a[href="/admin/users"]')).toBeVisible();
    await expect(page.locator('a[href="/admin/groups"]')).toBeVisible();
    await expect(page.locator('a[href="/admin/databases"]')).toBeVisible();

    // 4. Logout
    await page.locator('button:has-text("Logout"), button:has([data-lucide="log-out"]), button:has-text("Sign Out")').first().click();
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
    const proposeBtn = auditorLeadPage.locator('button:has-text("Propose Migration Draft"), a:has-text("Propose Migration Draft")');
    const rollbackBtn = auditorLeadPage.locator('button:has-text("Rollback (1-Step)")');
    const forwardBtn = auditorLeadPage.locator('button:has-text("Forward (Redo)")');

    if (await proposeBtn.count() > 0) {
      await expect(proposeBtn).toBeDisabled();
    }
    if (await rollbackBtn.count() > 0) {
      await expect(rollbackBtn).toBeDisabled();
    }
    if (await forwardBtn.count() > 0) {
      await expect(forwardBtn).toBeDisabled();
    }

    // 3. Directly invoke API mutation with auditor_lead session
    const response = await auditorLeadPage.request.post('/api/v1/me/databases/1/drafts', {
      data: {
        sql_up: 'CREATE TABLE auditor_test (id INT PRIMARY KEY);',
      },
    });

    // 4. Assert backend rejects with 403 Forbidden
    expect([403, 401]).toContain(response.status());
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
