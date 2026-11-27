import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Storage state paths for pre-authenticated role contexts
 */
export const AUTH_DIR = path.join(__dirname, '.auth');
export const ADMIN_AUTH = path.join(AUTH_DIR, 'admin.json');
export const LEADER_AUTH = path.join(AUTH_DIR, 'leader.json');
export const USER_AUTH = path.join(AUTH_DIR, 'user.json');
export const LEADER_QA_AUTH = path.join(AUTH_DIR, 'leader_qa.json');
export const AUDITOR_LEAD_AUTH = path.join(AUTH_DIR, 'auditor_lead.json');

export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // Ensure deterministic multi-step state transitions across tests
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // Run sequentially to preserve database seed state order
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report', open: 'never' }]
  ],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 15000,
  },

  projects: [
    // Global Auth Setup Project
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
    },

    // Standard Desktop Chromium testing
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
      dependencies: ['setup'],
    },

    // Mobile Viewport (iPhone 14 / 375x812) testing
    {
      name: 'mobile',
      testMatch: /.*01_auth_access_control\.spec\.ts/,
      use: {
        ...devices['iPhone 14'],
      },
      dependencies: ['setup'],
    },
  ],
});
