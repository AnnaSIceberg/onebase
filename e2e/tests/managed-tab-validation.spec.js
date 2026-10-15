const { test, expect } = require('@playwright/test');
const path = require('node:path');

// The Go regression covers rendering and the form-event hidden map. These
// browser cases cover the next public entry point: native constraint validation
// in a real DOM, with the complete production runtime and actual required inputs.
// display:none on a button is the state applied by the form-event response;
// display:none on content alone is an ordinary inactive, accessible page.
async function mount(page, { hidden = true, nested = false } = {}) {
  const inner = nested ? `
    <div class="managed-tabs" data-tabs="inner">
      <div><button type="button" class="managed-tab-btn active" data-tab-idx="0">Inner</button></div>
      <div class="managed-tab-content" data-tab-content="0" style="display:block">
        <input id="conditional" required>
      </div>
    </div>` : '<input id="conditional" required>';
  await page.setContent(`
    <form id="form">
      <div class="managed-tabs" data-tabs="outer">
        <div>
          <button id="conditional-tab" type="button" class="managed-tab-btn" data-tab-idx="0" style="display:${hidden ? 'none' : ''}">Conditional</button>
          <button id="home-tab" type="button" class="managed-tab-btn active" data-tab-idx="1">Home</button>
          <button id="available-tab" type="button" class="managed-tab-btn" data-tab-idx="2">Available</button>
        </div>
        <div id="conditional-page" class="managed-tab-content" data-tab-content="0" style="display:none">${inner}</div>
        <div id="home-page" class="managed-tab-content" data-tab-content="1" style="display:block">Home</div>
        <div id="available-page" class="managed-tab-content" data-tab-content="2" style="display:none"><input id="available" required></div>
      </div>
      <button type="submit">Save</button>
    </form>`);
  await page.addScriptTag({ path: path.join(__dirname, '../../internal/ui/static/managed.js') });
}

for (const nested of [false, true]) {
  test(`native validation keeps a conditional page hidden${nested ? ' through its ancestor' : ''}`, async ({ page }) => {
    await mount(page, { nested });
    // Native validation visits the conditionally hidden input first. It must
    // still reveal and focus the later invalid input on an available tab.
    expect(await page.locator('#form').evaluate((form) => form.reportValidity())).toBe(false);
    await expect(page.locator('#conditional-tab')).toBeHidden();
    await expect(page.locator('#conditional-tab')).not.toHaveClass(/active/);
    await expect(page.locator('#conditional-page')).toBeHidden();
    await expect(page.locator('#available-page')).toBeVisible();
    await expect(page.locator('#available')).toBeFocused();

    // Repeat via the actual submit button in a fresh validation pass.
    await page.locator('#home-tab').click();
    await page.locator('button[type="submit"]').click();
    await expect(page.locator('#conditional-page')).toBeHidden();
    await expect(page.locator('#available')).toBeFocused();
  });
}

test('native validation opens an ordinary inactive page and preserves the first invalid field', async ({ page }) => {
  await mount(page, { hidden: false });
  expect(await page.locator('#form').evaluate((form) => form.reportValidity())).toBe(false);
  await expect(page.locator('#conditional-page')).toBeVisible();
  await expect(page.locator('#conditional')).toBeFocused();
  await expect(page.locator('#available-page')).toBeHidden();

  await page.locator('#conditional').fill('complete');
  await page.locator('button[type="submit"]').click();
  await expect(page.locator('#available')).toBeFocused();
});
