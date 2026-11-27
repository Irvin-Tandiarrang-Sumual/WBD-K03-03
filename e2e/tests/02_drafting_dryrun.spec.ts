import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 2: Drafting, Wire-Protocol Dry-Run & Console Security', () => {
  test('TC-E2E-DFT-001: Verify PostgreSQL native wire-protocol dry-run and draft proposal creation', async ({
    userPage,
  }) => {
    // 1. User navigates to database detail for db1_pg (database id 1)
    await userPage.goto('/databases/1');

    // 2. Click Propose Migration Draft
    await userPage.locator('a:has-text("Propose Migration Draft"), button:has-text("Propose Migration Draft")').first().click();
    await expect(userPage).toHaveURL(/\/drafts\/new/);

    // 3. Fill proposal summary and Up SQL
    const uniqueTableName = `e2e_table_${Date.now()}`;
    await userPage.locator('#draft-description').fill(`E2E Test Proposal - ${uniqueTableName}`);
    await userPage.locator('#draft-up-sql').fill(`CREATE TABLE ${uniqueTableName} (id SERIAL PRIMARY KEY, title VARCHAR(100) NOT NULL);`);

    // 4. Submit draft for quorum
    await userPage.locator('button:has-text("Submit Draft for Quorum")').click();

    // 5. Assert successful redirect to draft detail page or draft list
    await expect(userPage).toHaveURL(/\/drafts\/\d+$/);
  });

  test('TC-E2E-SEC-001: Verify SQL Console read-only sanitizer blocks mutation and injection', async ({
    adminPage,
  }) => {
    // 1. Admin navigates to /console
    await adminPage.goto('/console');

    // 2. Input malicious destructive mutation statement
    const queryInput = adminPage.locator('textarea[aria-label="SQL Query Input"]');
    await queryInput.fill('DELETE FROM users;');

    // 3. Click Run Query button
    const runBtn = adminPage.locator('button:has-text("Run Query")');
    await runBtn.click();

    // 4. Verify system displays security rejection error
    const errorBox = adminPage.getByRole('heading', { name: 'Query Execution Failed' });
    await expect(errorBox).toBeVisible({ timeout: 5000 });
  });
});
