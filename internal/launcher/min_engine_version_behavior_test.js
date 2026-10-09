const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');
const html = fs.readFileSync(process.env.ONEBASE_START_ERROR_HTML, 'utf8');
const start = html.slice(html.indexOf('function startBase(el, id)'), html.indexOf('// Esc закрывает'));
const isolated = html.slice(html.indexOf('function startIsolated('), html.indexOf('function cleanProfiles('));
for (const mode of ['startBase', 'startBaseNative', 'startIsolated']) {
  for (const warning of [false, true]) {
    test(`${mode}: successful launch returns to ${warning ? 'warning banner' : 'base list'}`, async () => {
      const baseWindow = {location: {}, document: {write() {}, close() {}}};
      const launcherWindow = {location: {}, open() { return baseWindow; }};
      const result = {url: 'http://127.0.0.1:8080', ok: true};
      if (warning) result.launcher_url = '/?sel=base-control&flash=warning-key';
      const context = {
        _nativeOK: false,
        window: launcherWindow,
        document: {getElementById() { return null; }},
        suppressEvent() {}, startButton() { return null; }, setStartButtonHTML() {},
        setTimeout(fn) { fn(); }, fetch() { return Promise.resolve({json: () => Promise.resolve(result)}); },
        showStartError() { throw new Error('warning became a startup error'); },
        showStartErrorModal() { throw new Error('warning became a startup error'); }
      };
      vm.createContext(context);
      vm.runInContext(start + '\n' + isolated, context);
      context[mode](null, 'base-control', '');
      await new Promise(resolve => setImmediate(resolve));
      assert.equal(launcherWindow.location.href, result.launcher_url || '/?sel=base-control');
      if (mode === 'startBase') assert.equal(baseWindow.location.href, result.url);
    });
  }
}
