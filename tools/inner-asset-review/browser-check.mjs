import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { chromium } from 'playwright-core';

const [reviewPath, evidenceDir] = process.argv.slice(2);
if (!reviewPath || !evidenceDir) {
  throw new Error('usage: node browser-check.mjs <review.html> <evidence-dir>');
}

const chromeCandidates = [
  process.env.CHROME_BIN,
  '/usr/bin/google-chrome',
  '/usr/bin/google-chrome-stable',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
].filter(Boolean);
const executablePath = chromeCandidates.find((candidate) => fs.existsSync(candidate));
if (!executablePath) {
  throw new Error(`no system Chromium found; checked ${chromeCandidates.join(', ')}`);
}

fs.mkdirSync(evidenceDir, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath, args: ['--no-sandbox'] });
const receipt = { executablePath, surfaces: [] };

async function exercise(name, viewport, preferredView) {
  const context = await browser.newContext({ viewport });
  const page = await context.newPage();
  const pageErrors = [];
  const consoleErrors = [];
  const failedRequests = [];
  page.on('pageerror', (error) => pageErrors.push(String(error)));
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text());
  });
  page.on('requestfailed', (request) => failedRequests.push(`${request.method()} ${request.url()}`));

  await page.goto(pathToFileURL(path.resolve(reviewPath)).href, { waitUntil: 'load' });
  await page.waitForFunction(() => [...document.images].every((image) => image.complete && image.naturalWidth > 0));

  const guardText = await page.locator('.guard').innerText();
  if (!guardText.includes('DISPLAY ≠ APPROVAL') || !guardText.includes('NO AUTO-MATERIALIZE')) {
    throw new Error(`${name}: authority boundary is not visible`);
  }
  const buttons = page.locator('[data-view-button]');
  if (await buttons.count() !== 2) throw new Error(`${name}: expected exactly two realization controls`);
  const minTarget = await buttons.evaluateAll((items) => Math.min(...items.map((item) => item.getBoundingClientRect().height)));
  if (minTarget < 44) throw new Error(`${name}: realization control below 44px (${minTarget})`);

  await page.locator(`[data-view-button="${preferredView}"]`).click();
  const selected = page.locator(`[data-view-button="${preferredView}"]`);
  if ((await selected.getAttribute('aria-pressed')) !== 'true') throw new Error(`${name}: selected realization did not expose pressed state`);
  const selectedFigure = page.locator(`[data-view="${preferredView}"]`);
  if (await selectedFigure.getAttribute('hidden') !== null) throw new Error(`${name}: selected realization stayed hidden`);
  const opposite = preferredView === 'primary' ? 'preview' : 'primary';
  if (await page.locator(`[data-view="${opposite}"]`).getAttribute('hidden') === null) throw new Error(`${name}: inactive realization stayed visible`);
  const liveText = await page.locator('[data-view-status]').innerText();
  if (!liveText.toLowerCase().includes(preferredView)) throw new Error(`${name}: live feedback did not name selected realization`);

  const geometry = await page.evaluate(() => ({
    viewportWidth: innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    candidate: document.body.dataset.candidateSha,
    state: document.body.dataset.state,
    visibleImage: document.querySelector('[data-view]:not([hidden]) img')?.getBoundingClientRect().toJSON(),
  }));
  const overflow = Math.max(0, geometry.scrollWidth - geometry.viewportWidth);
  if (overflow !== 0) throw new Error(`${name}: ${overflow}px horizontal overflow`);
  if (!geometry.candidate || !geometry.state) throw new Error(`${name}: exact candidate identity/state missing from rendered surface`);
  if (!geometry.visibleImage || geometry.visibleImage.width <= 0 || geometry.visibleImage.height <= 0) throw new Error(`${name}: selected verified image has no visible geometry`);
  if (pageErrors.length || consoleErrors.length || failedRequests.length) {
    throw new Error(`${name}: runtime errors ${JSON.stringify({ pageErrors, consoleErrors, failedRequests })}`);
  }

  const screenshot = path.join(evidenceDir, `inner-asset-review-${name}.png`);
  await page.screenshot({ path: screenshot, fullPage: true });
  receipt.surfaces.push({
    name,
    viewport,
    preferredView,
    minTarget,
    overflow,
    candidate: geometry.candidate,
    state: geometry.state,
    liveText,
    pageErrors,
    consoleErrors,
    failedRequests,
    screenshot,
  });
  await context.close();
}

try {
  await exercise('desktop-primary', { width: 1440, height: 1000 }, 'primary');
  await exercise('phone-preview', { width: 390, height: 844 }, 'preview');
  fs.writeFileSync(path.join(evidenceDir, 'browser-receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`);
  console.log(JSON.stringify(receipt, null, 2));
} finally {
  await browser.close();
}
