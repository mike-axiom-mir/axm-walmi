const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const path = require('node:path');

const html = fs.readFileSync(path.join(__dirname, '..', 'cmd', 'workshop', 'ui.html'), 'utf8');

function workshopState() {
  return {
    Version: 1,
    Identities: [{ ID: 'identity-waldo', Name: 'Waldo', Model: 'waldo' }],
    Sessions: [{
      ID: 'session-one',
      IdentityID: 'identity-waldo',
      Title: 'Build session',
      Goal: 'Saved goal',
      Heartbeat: 'active',
      RuntimeMode: 'active',
      HeartbeatEverySec: 300,
      NextHeartbeat: '2026-09-09T12:00:00Z',
      PulseCount: 0,
      Messages: [],
    }],
    Memories: [],
    Consents: [],
    Media: [],
  };
}

async function installWorkshopRoutes(page, state, counters) {
  await page.route('http://workshop.test/**', async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === 'GET' && url.pathname === '/') {
      await route.fulfill({ status: 200, contentType: 'text/html', body: html });
      return;
    }
    if (request.method() === 'GET' && url.pathname === '/api/state') {
      counters.polls += 1;
      state.Sessions[0].PulseCount = counters.polls;
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(state) });
      return;
    }
    if (request.method() === 'POST' && url.pathname === '/api/op') {
      const body = request.postDataJSON();
      if (body.Op === 'goal') {
        state.Sessions[0].Goal = String(body.Goal || '').trim();
        counters.saves += 1;
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(state.Sessions[0]) });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }
    await route.fulfill({ status: 404, body: 'not found' });
  });
}

test('live polling preserves an unsaved goal draft until the user saves it', async ({ page }) => {
  const state = workshopState();
  const counters = { polls: 0, saves: 0 };
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await installWorkshopRoutes(page, state, counters);
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto('http://workshop.test/');

  const goal = page.locator('#goal');
  const goalState = page.locator('#goalState');
  const save = page.locator('#savegoal');

  await expect(goal).toHaveValue('Saved goal');
  await expect(goalState).toHaveText('Saved');
  await expect(save).toBeDisabled();

  await goal.fill('Draft that must survive background refresh');
  await expect(goalState).toHaveText('Unsaved draft');
  await expect(save).toBeEnabled();
  await expect.poll(() => counters.polls, { timeout: 5000 }).toBeGreaterThanOrEqual(2);
  await page.waitForTimeout(2100);

  await expect(goal).toHaveValue('Draft that must survive background refresh');
  await expect(goalState).toHaveText('Unsaved draft');
  await page.screenshot({ path: 'test-results/workshop-goal-draft-desktop.png', fullPage: true });

  await goal.press('Control+S');
  await expect.poll(() => counters.saves).toBe(1);
  await expect.poll(() => state.Sessions[0].Goal).toBe('Draft that must survive background refresh');
  await expect(goalState).toHaveText('Saved');
  await expect(save).toBeDisabled();
  expect(pageErrors).toEqual([]);
});

test('goal draft state remains readable on a phone-sized viewport', async ({ page }) => {
  const state = workshopState();
  const counters = { polls: 0, saves: 0 };
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await installWorkshopRoutes(page, state, counters);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('http://workshop.test/');

  await page.locator('#toggle').click();
  await page.locator('#goal').fill('Phone-sized draft');
  await expect(page.locator('#goalState')).toHaveText('Unsaved draft');
  await expect(page.locator('#savegoal')).toBeEnabled();
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow).toBeLessThanOrEqual(0);
  expect(pageErrors).toEqual([]);
  await page.screenshot({ path: 'test-results/workshop-goal-draft-mobile.png', fullPage: true });
});
