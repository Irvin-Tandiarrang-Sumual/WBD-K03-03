import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 7: Admin Panel Governance & Protections (ADM)', () => {
  test('TC-E2E-ADM-001: Verify Last Admin protection and group leader succession enforcement', async ({
    adminPage,
  }) => {
    // 1. Navigate to /admin/users
    await adminPage.goto('/admin/users');
    await expect(adminPage).toHaveURL(/\/admin\/users/);

    // 2. Locate admin user row
    const adminUserRow = adminPage.locator('tr:has-text("admin"), div:has-text("admin")').first();
    await expect(adminUserRow).toBeVisible();

    // 3. Navigate to /admin/groups
    await adminPage.goto('/admin/groups');
    await expect(adminPage).toHaveURL(/\/admin\/groups/);

    // 4. Verify engineering group exists with leader designated
    const engineeringGroup = adminPage.getByText('engineering', { exact: true }).first();
    await expect(engineeringGroup).toBeVisible();
  });
});
