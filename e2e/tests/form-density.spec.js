const { test, expect } = require('@playwright/test');
const { login, open } = require('./helpers');

// Load real YAML forms through the HTTP handler: CSS text alone cannot prove
// that compact rules override the group/page inline defaults (#2277).
for (const compact of [false, true]) {
  test(`form group spacing: ${compact ? 'compact' : 'default'}`, async ({ page }) => {
    await login(page);
    await open(page, `/ui/document/${compact ? 'ПлотнаяФорма' : 'ОбычнаяФорма'}/new`);
    expect(await page.locator('html').evaluate((el) => el.classList.contains('ob-density-compact'))).toBe(compact);
    for (const name of ['Group', 'Nested', 'Page']) {
      const container = page.locator(`[data-ob-el="${name}"]`);
      await expect(container).toBeVisible();
      await expect(container).toHaveCSS('padding-top', compact ? '6px' : '12px');
      await expect(container).toHaveCSS('padding-right', compact ? '10px' : '14px');
      await expect(container).toHaveCSS('padding-bottom', compact ? '8px' : '12px');
      await expect(container).toHaveCSS('padding-left', compact ? '10px' : '14px');
      await expect(container).toHaveCSS('margin-bottom', compact ? '8px' : '14px');
    }
  });
}
