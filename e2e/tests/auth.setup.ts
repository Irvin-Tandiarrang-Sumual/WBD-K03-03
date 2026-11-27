import { test as setup, expect } from '@playwright/test';
import {
  ADMIN_AUTH,
  LEADER_AUTH,
  USER_AUTH,
  LEADER_QA_AUTH,
  AUDITOR_LEAD_AUTH,
} from '../playwright.config.js';

const SEED_ACCOUNTS = [
  { username: 'admin', password: 'admin123', path: ADMIN_AUTH },
  { username: 'leader', password: 'leader123', path: LEADER_AUTH },
  { username: 'user', password: 'user123', path: USER_AUTH },
  { username: 'leader_qa', password: 'qa123', path: LEADER_QA_AUTH },
  { username: 'auditor_lead', password: 'auditor123', path: AUDITOR_LEAD_AUTH },
];

for (const account of SEED_ACCOUNTS) {
  setup(`authenticate as ${account.username}`, async ({ page }) => {
    await page.goto('/login');

    // Fill login form
    await page.locator('#login-username').fill(account.username);
    await page.locator('#login-password').fill(account.password);

    // Submit form
    await page.locator('button[type="submit"]').click();

    // Verify redirected to /databases
    await expect(page).toHaveURL(/\/databases/);

    // Save signed-in state
    await page.context().storageState({ path: account.path });
  });
}
