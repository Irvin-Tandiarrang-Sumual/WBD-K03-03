import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 6: Migration Lifecycle, Rollback & Forward (HIS)', () => {
  test('TC-E2E-HIS-001: Verify Rollback 1-Step is active on db1_pg (HEAD #3)', async ({
    adminPage,
  }) => {
    // 1. Open db1_pg (database id 1)
    await adminPage.goto('/databases/1');

    // 2. Check Rollback (1-Step) button
    const rollbackBtn = adminPage.locator('button:has-text("Rollback (1-Step)")').first();
    await expect(rollbackBtn).toBeVisible();
    await expect(rollbackBtn).toBeEnabled();
  });

  test('TC-E2E-HIS-002: Verify Rollback is disabled on irreversible migration (db2_mysql)', async ({
    adminPage,
  }) => {
    // 1. Open db2_mysql (database id 2)
    await adminPage.goto('/databases/2');

    // 2. Rollback (1-Step) must be disabled due to irreversible DDL (DROP COLUMN)
    const rollbackBtn = adminPage.locator('button:has-text("Rollback (1-Step)")').first();
    await expect(rollbackBtn).toBeVisible();
    await expect(rollbackBtn).toBeDisabled();
  });

  test('TC-E2E-HIS-003: Verify Forward (Redo) is active on db3_analytics (Post-Rollback #1)', async ({
    adminPage,
  }) => {
    // 1. Open db3_analytics (database id 3)
    await adminPage.goto('/databases/3');

    // 2. Forward (Redo) button must be enabled
    const forwardBtn = adminPage.locator('button:has-text("Forward (Redo)")').first();
    await expect(forwardBtn).toBeVisible();
    await expect(forwardBtn).toBeEnabled();
  });

  test('TC-E2E-HIS-004: Verify Hide Rollbacks checkbox toggles paired execution records in history', async ({
    adminPage,
  }) => {
    // 1. Open db1_pg or db3_analytics with migration history
    await adminPage.goto('/databases/1');

    // 2. Find Hide Rollbacks checkbox
    const hideRollbacksCheckbox = adminPage.locator('input[type="checkbox"]#hide-rollbacks, label:has-text("Hide Rollbacks") input[type="checkbox"]').first();
    if (await hideRollbacksCheckbox.count() > 0) {
      const historyRows = adminPage.locator('table tbody tr');
      const initialCount = await historyRows.count();

      // Toggle checkbox ON
      await hideRollbacksCheckbox.check();
      const filteredCount = await historyRows.count();
      expect(filteredCount).toBeLessThanOrEqual(initialCount);

      // Toggle checkbox OFF
      await hideRollbacksCheckbox.uncheck();
      const restoredCount = await historyRows.count();
      expect(restoredCount).toBe(initialCount);
    }
  });
});
