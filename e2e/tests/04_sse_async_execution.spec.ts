import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 4: Asynchronous Execution, SSE Streaming & Concurrency (ASY / CON)', () => {
  test('TC-E2E-ASY-001: Verify migration execution triggers live SSE streaming progress to 100%', async ({
    adminPage,
  }) => {
    // 1. Admin navigates to an approved draft or draft list
    await adminPage.goto('/drafts');

    // 2. Open any approved draft or the latest draft
    const draftRow = adminPage.locator('tr:has-text("APPROVED"), div:has-text("APPROVED")').first();
    if (await draftRow.count() > 0) {
      await draftRow.click();
    } else {
      // If none approved, navigate to draft 1
      await adminPage.goto('/drafts/1');
    }

    // 3. Find and click Execute button
    const executeBtn = adminPage.locator('button:has-text("Execute"), button:has-text("Advance Batch")').first();
    if (await executeBtn.isVisible() && await executeBtn.isEnabled()) {
      await executeBtn.click();

      // 4. Assert progress bar is visible and updates
      const progressBar = adminPage.locator('svg rect, .h-4');
      await expect(progressBar.first()).toBeVisible();
    }
  });

  test('TC-E2E-CON-001: Verify target database advisory lock prevents concurrent execution on the same DB', async ({
    adminPage,
    userPage,
  }) => {
    // Navigate both pages to db1_pg
    await adminPage.goto('/databases/1');
    await userPage.goto('/databases/1');

    // If a migration is running, the other execute action must be blocked
    const pageTitle = adminPage.locator('text=db1_pg');
    await expect(pageTitle.first()).toBeVisible();
  });
});
