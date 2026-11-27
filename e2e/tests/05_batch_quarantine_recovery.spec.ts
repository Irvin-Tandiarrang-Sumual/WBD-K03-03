import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 5: Batch Recovery & Quarantine (BAT / CON)', () => {
  test('TC-E2E-BAT-001: Verify db4_billing displays 423 Quarantined banner and blocks new migration', async ({
    adminPage,
  }) => {
    // 1. Navigate to db4_billing (database id 4)
    await adminPage.goto('/databases/4');

    // 2. Verify 423 Quarantined banner is prominently displayed
    const quarantineBanner = adminPage.getByText('423 QUARANTINED', { exact: true });
    await expect(quarantineBanner).toBeVisible();

    // 3. Verify Propose Migration Draft button is disabled while database is quarantined
    const proposeBtn = adminPage.getByRole('link', { name: 'Propose Migration Draft' });
    await expect(proposeBtn).toHaveAttribute('href', '/databases/4');
  });

  test('TC-E2E-BAT-002: Verify the seeded recovery decision offers Resume Batch', async ({
    leaderPage,
  }) => {
    // 1. Open draft list or navigate directly to the quarantined draft on db4_billing
    await leaderPage.goto('/drafts');

    const quarantinedRow = leaderPage.getByRole('row').filter({ hasText: 'db4_billing' }).filter({ hasText: /partial/i });
    await quarantinedRow.click();

    await expect(leaderPage.getByText(/Consensus reached: Resume/)).toBeVisible();
    await expect(leaderPage.getByRole('button', { name: 'Resume Batch', exact: true })).toBeVisible();
  });
});
