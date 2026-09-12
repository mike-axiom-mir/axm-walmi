const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const path = require('node:path');

const html = fs.readFileSync(path.join(__dirname, '..', 'cmd', 'workshop', 'ui.html'), 'utf8');

function workshopState() {
  return {
    Version: 2,
    Identities: [{ ID: 'waldo', Name: 'Waldo', Model: 'waldo' }],
    Sessions: [{
      ID: 'session-one',
      IdentityID: 'waldo',
      Title: 'Waldo Mirror',
      Heartbeat: 'paused',
      RuntimeMode: 'paused',
      HeartbeatEverySec: 300,
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
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(state) });
      return;
    }
    if (request.method() === 'POST' && url.pathname === '/api/chat') {
      const body = request.postDataJSON();
      const messages = state.Sessions[0].Messages;
      if (!body.RetryMessageID) {
        counters.initial += 1;
        messages.push({
          ID: 'msg-failed',
          Role: 'user',
          Text: body.Text,
          CreatedAt: '2026-09-09T16:57:00Z',
		  Delivery: 'pending',
        });
		await new Promise(resolve => setTimeout(resolve, 3500));
		messages[0].Delivery = 'failed';
		messages[0].DeliveryError = 'AI endpoint unavailable: connection refused';
        await route.fulfill({
          status: 502,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'AI endpoint unavailable: connection refused' }),
        });
        return;
      }
      counters.retries += 1;
      const original = messages.find(message => message.ID === body.RetryMessageID);
      original.Delivery = 'answered';
      original.DeliveryError = '';
      messages.push({
        ID: 'msg-answer',
        Role: 'assistant',
        Text: 'The local model is ready now.',
        CreatedAt: '2026-09-09T16:57:05Z',
      });
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

for (const viewport of [
  { name: 'desktop', width: 1280, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`failed delivery remains visible and retryable on ${viewport.name}`, async ({ page }) => {
    const state = workshopState();
    const counters = { initial: 0, retries: 0 };
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await installWorkshopRoutes(page, state, counters);
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto('http://workshop.test/');
	if (viewport.name === 'mobile') {
	  await page.locator('#toggle').click();
	}

    await page.locator('#text').fill('Give me the next concrete step.');
    await page.locator('#send').click();

	await expect(page.locator('.delivery.pending')).toContainText('Waiting for Waldo');
	await expect(page.locator('#messages')).toHaveAttribute('aria-busy', 'true');
    const failed = page.locator('.delivery.failed');
    await expect(failed).toContainText('Response failed.');
    await expect(failed).toContainText('connection refused');
    await expect(page.locator('[data-retry="msg-failed"]')).toBeVisible();
    await page.waitForTimeout(4200);
    await expect(failed).toBeVisible();
    await expect(page.locator('#notice')).toHaveText('');
    expect(counters.initial).toBe(1);
    expect(state.Sessions[0].Messages).toHaveLength(1);
    expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
    await page.screenshot({
      path: `test-results/workshop-failed-delivery-${viewport.name}.png`,
      fullPage: true,
    });

    await page.locator('[data-retry="msg-failed"]').click();
    await expect(page.locator('.delivery.failed')).toHaveCount(0);
    await expect(page.locator('.msg.assistant')).toContainText('The local model is ready now.');
    expect(counters.retries).toBe(1);
    expect(state.Sessions[0].Messages).toHaveLength(2);
    expect(pageErrors).toEqual([]);
  });
}
