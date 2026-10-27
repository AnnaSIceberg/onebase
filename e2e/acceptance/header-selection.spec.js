const { test, expect } = require('@playwright/test');
const paths = JSON.parse(process.env.OB_HEADER_SELECTION_PATHS || '{}');
let pageErrors;
test.beforeEach(async ({ page }) => {
  pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
});
test.afterEach(() => expect(pageErrors).toEqual([]));

async function fire(page, command, selected, messages) {
  const [response] = await Promise.all([
    page.waitForResponse(r => r.request().method() === 'POST' && r.url().includes('/form-event')),
    page.locator(`[data-ob-fire-click="${command}"]`).click(),
  ]);
  const sent = new URLSearchParams(response.request().postData());
  expect(sent.get('_tp')).toBe('Lines');
  expect(sent.get('_tp_selected')).toBe(selected);
  const result = await response.json();
  expect(result.ok, result.error).toBe(true);
  expect(result.messages).toEqual(['Lines', String(messages.length), ...messages]);
}

for (const mode of ['header', 'local', 'mixed', 'popup']) {
  test(`${mode}: selected rows survive header placement and redraw`, async ({ page }) => {
    await page.goto(paths[mode]);
    const selectors = page.locator('#tp-body-Lines ._tp-sel');
    await expect(selectors).toHaveCount(2);
    await expect(page.locator('#tp-body-Other ._tp-sel')).toHaveCount(0);
    const columns = await page.locator('table[data-tp="Lines"] thead th').count();
    expect(await page.locator('#tp-body-Lines tr').first().locator('td').count()).toBe(columns);
    expect(await page.locator('#tp-foot-Lines td').count()).toBe(columns);
    await selectors.nth(1).check();
    await fire(page, 'SelectRows', '1', ['B']);
    // The actual event response rebuilds the rows, including their selectors.
    await expect(selectors).toHaveCount(2);
    await fire(page, 'SelectRows', '', []);
    await page.locator('[data-ob-add-tp="Lines"]').click();
    await expect(selectors).toHaveCount(3);
    expect(await page.locator('#tp-body-Lines tr').last().locator('td').count()).toBe(columns);
    await page.locator('[name="tp.Lines.2.Note"]').fill('C');
    await selectors.nth(0).check();
    await selectors.nth(2).check();
    await fire(page, mode === 'mixed' ? 'LocalRows' : 'SelectRows', '0,2', ['A', 'C']);
    await expect(selectors).toHaveCount(3);
    await selectors.nth(2).check();
    await fire(page, 'SelectRows', '2', ['C']);
    // Native form ownership keeps an unrelated external control out of this form.
    await page.evaluate(() => {
      const other = document.createElement('form');
      other.innerHTML = '<input type="checkbox" class="_tp-sel" checked>';
      document.body.appendChild(other);
    });
    await fire(page, 'SelectRows', '', []);
    const url = new URL(paths[mode], 'http://fixture');
    const [, , , entity, id] = url.pathname.split('/');
    const response = await page.request.post(`/ui/catalog/${entity}/form-event`, {
      form: { _id: id, _kind: 'object', _element: 'SelectRows', _event: 'Нажатие', _tp: 'Other', _tp_selected: '0' },
    });
    const rejected = await response.json();
    expect(rejected.ok).toBe(false);
    expect(rejected.messages || []).toEqual([]);
  });
}

test('grid: header command retains marker and existing event context', async ({ page }) => {
  await page.goto(paths.grid);
  await expect(page.locator('[data-sg-tp="Lines"]')).toHaveAttribute('data-sg-cmd', '1');
  await page.waitForFunction(() => window._obGrids?.Lines);
  // Vendored SlickGrid has no RowSelectionModel. Preserve its ordinary empty
  // selection rather than installing a new selection mechanism for this fix.
  await page.locator('#sg-Lines .slick-row').nth(1).locator('.slick-cell').first().click();
  await fire(page, 'SelectRows', '', []);
});

for (const mode of ['readonly', 'readuser', 'hidden', 'none']) {
  test(`${mode}: placement grants no extra action or selection`, async ({ page }) => {
    await page.goto(paths[mode]);
    if (mode === 'none') {
      await expect(page.locator('._tp-sel')).toHaveCount(0);
      await expect(page.locator('[data-ob-el="SelectRows"]')).toHaveCount(0);
    } else if (mode === 'hidden') {
      await expect(page.locator('[data-ob-fire-click="SelectRows"]')).toBeHidden();
      await expect(page.locator('#tp-body-Lines ._tp-sel').first()).toBeHidden();
    } else {
      await expect(page.locator('button[data-ob-el="SelectRows"]')).toBeDisabled();
      await expect(page.locator('#tp-body-Lines ._tp-sel').first()).toBeDisabled();
      const url = new URL(paths[mode], 'http://fixture');
      const entity = url.pathname.split('/')[3];
      const id = url.pathname.split('/')[4];
      const response = await page.request.post(`/ui/catalog/${entity}/form-event${url.search}`, {
        form: { _id: id, _kind: 'object', _element: 'SelectRows', _event: 'Нажатие', _tp: 'Lines', _tp_selected: '0' },
      });
      const data = await response.json();
      expect(data.ok).toBe(false);
      expect(data.messages || []).toEqual([]);
    }
  });
}
