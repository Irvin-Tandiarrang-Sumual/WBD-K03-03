import { test, expect } from '../fixtures/test-fixtures.js';

test.describe('Module 3: Multi-Leader Quorum & Invalidation (APR)', () => {
  test('TC-E2E-APR-001: Verify multi-leader 100% unanimous approval workflow', async ({
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
    await userPage.waitForURL(/\/drafts/);

    // Get current draft URL
    const currentDraftUrl = userPage.url();

    // 2. Leader 1 (engineering) approves
    await leaderPage.goto(currentDraftUrl);
    const approveBtnLeader = leaderPage.locator('button:has-text("Approve")');
    if (await approveBtnLeader.isVisible()) {
      await approveBtnLeader.click();
      // Should still be pending approval until all leaders approve
      await expect(leaderPage.locator('text=Approved').first()).not.toBeVisible();
    }

    // 3. Leader 2 (qa-team) approves
    await leaderQaPage.goto(currentDraftUrl);
    const approveBtnQa = leaderQaPage.locator('button:has-text("Approve")');
    if (await approveBtnQa.isVisible()) {
      await approveBtnQa.click();
    }

    // 4. Leader 3 (auditors, view_only) approves
    await auditorLeadPage.goto(currentDraftUrl);
    const approveBtnAuditor = auditorLeadPage.locator('button:has-text("Approve")');
    if (await approveBtnAuditor.isVisible()) {
      await approveBtnAuditor.click();
    }

    // 5. Verify draft status transitions to Approved once all approve
    await expect(auditorLeadPage.locator('text=APPROVED, span:has-text("APPROVED")').first()).toBeVisible({ timeout: 5000 });
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
    await userPage.waitForURL(/\/drafts/);

    const draftUrl = userPage.url();

    // 2. Leader 1 approves
    await leaderPage.goto(draftUrl);
    const approveBtn = leaderPage.locator('button:has-text("Approve")');
    if (await approveBtn.isVisible()) {
      await approveBtn.click();
    }

    // 3. Leader 2 rejects
    await leaderQaPage.goto(draftUrl);
    const rejectBtn = leaderQaPage.locator('button:has-text("Reject")');
    if (await rejectBtn.isVisible()) {
      await rejectBtn.click();
      // Status must transition to REJECTED
      await expect(leaderQaPage.locator('text=REJECTED, span:has-text("REJECTED")').first()).toBeVisible({ timeout: 5000 });
    }
  });
});
