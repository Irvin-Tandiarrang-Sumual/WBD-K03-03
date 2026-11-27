import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 3: Multi-Leader Quorum & Invalidation (APR)', () => {
  test('TC-E2E-APR-001: Verify multi-leader 100% unanimous approval workflow', async ({
    adminPage,
    leaderPage,
    leaderQaPage,
    auditorLeadPage,
    userPage,
  }) => {
    // 1. User creates a draft on db1_pg
    await userPage.goto('/databases/1');
    await userPage.locator('a:has-text("Propose Migration Draft"), button:has-text("Propose Migration Draft")').first().click();

    const proposalTitle = `Quorum Test ${Date.now()}`;
    await userPage.locator('#draft-description').fill(proposalTitle);
    await userPage.locator('#draft-up-sql').fill('CREATE TABLE quorum_demo (id INT PRIMARY KEY);');
    await userPage.locator('button:has-text("Submit Draft for Quorum")').click();
    await userPage.waitForURL(/\/drafts\/\d+$/);

    // Get current draft URL
    const currentDraftUrl = userPage.url();

    // 2. Leader 1 (engineering) approves
    await leaderPage.goto(currentDraftUrl);
    const approveBtnLeader = leaderPage.getByRole('button', { name: 'Approve', exact: true });
    await approveBtnLeader.click();
    // Should still be pending approval until all leaders approve
    await expect(leaderPage.locator('h1 + div')).toHaveText('PENDING_APPROVAL');

    // 3. Leader 2 (qa-team) approves
    await leaderQaPage.goto(currentDraftUrl);
    await leaderQaPage.getByRole('button', { name: 'Approve', exact: true }).click();

    // 4. Leader 3 (auditors, view_only) approves
    await auditorLeadPage.goto(currentDraftUrl);
    await auditorLeadPage.getByRole('button', { name: 'Approve', exact: true }).click();
    await adminPage.goto(currentDraftUrl);
    await adminPage.getByRole('button', { name: 'Approve', exact: true }).click();

    // 5. Verify draft status transitions to Approved once all approve
    await expect(adminPage.locator('h1 + div')).toHaveText('APPROVED');
  });

  test('TC-E2E-APR-002: Verify single leader veto immediately marks draft as Rejected', async ({
    userPage,
    leaderPage,
    leaderQaPage,
  }) => {
    // 1. User creates a draft
    await userPage.goto('/databases/1');
    await userPage.locator('a:has-text("Propose Migration Draft"), button:has-text("Propose Migration Draft")').first().click();

    const title = `Veto Test ${Date.now()}`;
    await userPage.locator('#draft-description').fill(title);
    await userPage.locator('#draft-up-sql').fill('CREATE TABLE veto_demo (id INT PRIMARY KEY);');
    await userPage.locator('button:has-text("Submit Draft for Quorum")').click();
    await userPage.waitForURL(/\/drafts\/\d+$/);

    const draftUrl = userPage.url();

    // 2. Leader 1 approves
    await leaderPage.goto(draftUrl);
    await leaderPage.getByRole('button', { name: 'Approve', exact: true }).click();

    // 3. Leader 2 rejects
    await leaderQaPage.goto(draftUrl);
    await leaderQaPage.getByRole('button', { name: 'Reject', exact: true }).click();
    // Status must transition to REJECTED
    await expect(leaderQaPage.locator('h1 + div')).toHaveText('REJECTED');
  });
});
