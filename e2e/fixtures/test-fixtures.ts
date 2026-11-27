import { test as base, type Browser, type Page } from '@playwright/test';
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

async function useRolePage(
  browser: Browser,
  baseURL: string | undefined,
  storageState: string,
  use: (page: Page) => Promise<void>,
) {
  const context = await browser.newContext({ baseURL, storageState });
  try {
    await context.route('https://fonts.googleapis.com/**', (route) => route.abort());
    await use(await context.newPage());
  } finally {
    await context.close();
  }
}

export const test = base.extend<CustomFixtures>({
  adminPage: async ({ browser, baseURL }, use) => {
    await useRolePage(browser, baseURL, ADMIN_AUTH, use);
  },

  leaderPage: async ({ browser, baseURL }, use) => {
    await useRolePage(browser, baseURL, LEADER_AUTH, use);
  },

  userPage: async ({ browser, baseURL }, use) => {
    await useRolePage(browser, baseURL, USER_AUTH, use);
  },

  leaderQaPage: async ({ browser, baseURL }, use) => {
    await useRolePage(browser, baseURL, LEADER_QA_AUTH, use);
  },

  auditorLeadPage: async ({ browser, baseURL }, use) => {
    await useRolePage(browser, baseURL, AUDITOR_LEAD_AUTH, use);
  },
});

export { expect } from '@playwright/test';
