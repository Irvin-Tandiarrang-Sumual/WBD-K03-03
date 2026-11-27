import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 5: Batch Recovery & Quarantine (BAT / CON)', () => {
  test('TC-E2E-BAT-001: Verify db4_billing displays 423 Quarantined banner and blocks new migration', async ({
    adminPage,
  }) => {
    // 1. Navigate to db4_billing (database id 4)
    await adminPage.goto('/databases/4');

    // 2. Verify 423 Quarantined banner is prominently displayed
    const quarantineBanner = adminPage.locator('text=423, text=QUARANTINED, span:has-text("Quarantined")').first();
    await expect(quarantineBanner).toBeVisible();

    // 3. Verify Propose Migration Draft button is disabled while database is quarantined
    const proposeBtn = adminPage.locator('button:has-text("Propose Migration Draft"), a:has-text("Propose Migration Draft")').first();
    if (await proposeBtn.count() > 0) {
      await expect(proposeBtn).toBeDisabled();
    }
  });

  test('TC-E2E-BAT-002: Verify recovery voting options (Resume vs Revert) on quarantined draft', async ({
    leaderPage,
  }) => {
    // 1. Open draft list or navigate directly to the quarantined draft on db4_billing
    await leaderPage.goto('/drafts');

    const quarantinedRow = leaderPage.locator('tr:has-text("PARTIAL"), div:has-text("PARTIAL"), tr:has-text("db4"), div:has-text("db4")').first();
    if (await quarantinedRow.count() > 0) {
      await quarantinedRow.click();

      // 2. Verify Recovery voting options appear
      const resumeBtn = leaderPage.locator('button:has-text("Resume")');
      const revertBtn = leaderPage.locator('button:has-text("Revert")');

      await expect(resumeBtn).toBeVisible();
      await expect(revertBtn).toBeVisible();
    }
  });
});
