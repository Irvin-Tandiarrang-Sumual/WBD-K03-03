import { test as base, Page } from '@playwright/test';
import {
  ADMIN_AUTH,
  LEADER_AUTH,
  USER_AUTH,
  LEADER_QA_AUTH,
  AUDITOR_LEAD_AUTH,
} from '../playwright.config.js';

type CustomFixtures = {
  adminPage: Page;
  leaderPage: Page;
  userPage: Page;
  leaderQaPage: Page;
  auditorLeadPage: Page;
};

export const test = base.extend<CustomFixtures>({
  adminPage: async ({ browser }, use) => {
    const context = await browser.newContext({ storageState: ADMIN_AUTH });
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  leaderPage: async ({ browser }, use) => {
    const context = await browser.newContext({ storageState: LEADER_AUTH });
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  userPage: async ({ browser }, use) => {
    const context = await browser.newContext({ storageState: USER_AUTH });
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  leaderQaPage: async ({ browser }, use) => {
    const context = await browser.newContext({ storageState: LEADER_QA_AUTH });
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  auditorLeadPage: async ({ browser }, use) => {
    const context = await browser.newContext({ storageState: AUDITOR_LEAD_AUTH });
    const page = await context.newPage();
    await use(page);
    await context.close();
  },
});

export { expect } from '@playwright/test';
