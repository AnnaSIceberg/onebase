const { test, expect } = require('@playwright/test');

// «Настройка списка» прямо в шапке: край заголовка меняет ширину колонки,
// перетаскивание заголовка — порядок; оба сохраняются за пользователем и
// переживают перезагрузку. «Стандартные настройки» возвращают список.
test('list header resize and drag are kept per user', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 750 });
  expect((await page.request.post('/auth/login', { data: { login: 'admin', password: 'secret123' } })).ok()).toBe(true);
  await page.goto('/ui/document/orders', { waitUntil: 'domcontentloaded' });
  // Список перерисовывается перезагрузкой: опрос переживает навигацию.
  const heads = () => page.locator('th[data-ob-ls-field]').evaluateAll((ths) => ths.map((th) => th.getAttribute('data-ob-ls-field'))).catch(() => null);
  expect(await heads()).toEqual(['Клиент', 'Сумма', 'Комментарий']);

  // Ширина «Клиент»: тянем правый край на +120px.
  const th = page.locator('th[data-ob-ls-field="Клиент"]');
  const before = Math.round((await th.boundingBox()).width);
  const grip = th.locator('[data-ob-ls-resize]');
  const g = await grip.boundingBox();
  const saved = page.waitForResponse((r) => r.url().includes('/list-settings/save') && r.status() === 204);
  await page.mouse.move(g.x + g.width / 2, g.y + g.height / 2);
  await page.mouse.down();
  await page.mouse.move(g.x + g.width / 2 + 120, g.y + g.height / 2, { steps: 6 });
  await page.mouse.up();
  await saved;
  expect(Math.abs(Math.round((await th.boundingBox()).width) - (before + 120))).toBeLessThanOrEqual(2);
  // Щелчок по краю не сортирует.
  expect(page.url()).not.toContain('sort=');
  await page.reload({ waitUntil: 'domcontentloaded' });
  expect(Math.abs(Math.round((await page.locator('th[data-ob-ls-field="Клиент"]').boundingBox()).width) - (before + 120))).toBeLessThanOrEqual(2);

  // Порядок: «Комментарий» перетаскиваем на место «Клиент».
  await page.locator('th[data-ob-ls-field="Комментарий"]').dragTo(page.locator('th[data-ob-ls-field="Клиент"]'));
  await expect.poll(heads).toEqual(['Комментарий', 'Клиент', 'Сумма']);
  await page.reload({ waitUntil: 'domcontentloaded' });
  expect(await heads()).toEqual(['Комментарий', 'Клиент', 'Сумма']);

  // Окно настройки: скрыть «Сумма», сохранить; затем «Стандартные настройки».
  await page.locator('[aria-controls="ob-list-settings-panel"]').click();
  await page.locator('.ls-col-row[data-field="Сумма"] .ls-visible').uncheck();
  await page.locator('details[data-block="list-settings"] button[type="submit"]:not([formaction])').click();
  await expect.poll(heads).toEqual(['Комментарий', 'Клиент']);
  await page.locator('details[data-block="list-settings"]').evaluate((d) => { d.open = true; });
  await page.locator('details[data-block="list-settings"] button[formaction$="/reset"]').click();
  await expect.poll(heads).toEqual(['Клиент', 'Сумма', 'Комментарий']);
});

test('settings scroll inside the list and keep Save visible', async ({ page }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  expect((await page.request.post('/auth/login', { data: { login: 'admin', password: 'secret123' } })).ok()).toBe(true);
  for (const entity of ['manyorders', 'manycalls', 'manytasks']) {
    await page.goto('/ui/document/' + entity, { waitUntil: 'domcontentloaded' });
    await page.locator('[aria-controls="ob-list-settings-panel"]').click();
    const panel = page.locator('#ob-list-settings-panel');
    const scroll = panel.locator('.ob-list-settings-scroll');
    await expect(scroll).toBeVisible();
    await expect(scroll.locator('tbody tr')).toHaveCount(46);
    const dimensions = await scroll.evaluate(el => [el.clientHeight, el.scrollHeight, window.innerHeight, getComputedStyle(el).overflowY]);
    expect(dimensions[0]).toBeLessThan(dimensions[1]);
    expect(dimensions[0]).toBeLessThan(dimensions[2] / 2);
    expect(dimensions[3]).toEqual('auto');

    await scroll.hover();
    await page.mouse.wheel(0, 420);
    await expect.poll(() => scroll.evaluate(el => el.scrollTop)).toBeGreaterThan(0);
    await scroll.focus();
    await page.keyboard.press('End');
    await expect.poll(() => scroll.evaluate(el => el.scrollHeight - el.scrollTop - el.clientHeight)).toBeLessThan(2);
    const last = scroll.locator('tbody tr').last();
    await expect(last).toBeInViewport();
    await expect(panel.locator('button[type="submit"]:not([formaction])')).toBeInViewport();
  }
});
