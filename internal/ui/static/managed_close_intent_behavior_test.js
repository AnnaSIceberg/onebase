'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const managed = fs.readFileSync('static/managed.js', 'utf8');
const start = managed.indexOf('  // One close controller for every managed-form adapter.');
const end = managed.indexOf('  // Отслеживание «грязной» формы', start);
assert.ok(start >= 0 && end > start, 'managed close-controller slice not found');
const controller = managed.slice(start, end);
const escapeComment = managed.indexOf('// Esc — отмена незаконченного ввода');
const escapeStart = managed.indexOf('  function consumeManagedEscape', escapeComment);
const escapeEnd = managed.indexOf('  }, true);', escapeStart);
assert.ok(escapeComment >= 0 && escapeStart >= 0 && escapeEnd > escapeStart, 'managed Escape slice not found');
const escapeHandler = managed.slice(escapeStart, escapeEnd + '  }, true);'.length);

class FakeFormData {
  constructor(form) {
    this.values = new Map(form.values);
  }
  set(name, value) { this.values.set(String(name), String(value)); }
  append(name, value) {
    name = String(name);
    value = String(value);
    if (!this.values.has(name)) this.values.set(name, value);
    else this.values.set(name, this.values.get(name) + ',' + value);
  }
  forEach(fn) { for (const [key, value] of this.values) fn(value, key); }
}

function runtime(fetchImpl, cfgOverride = {}) {
  const applied = [];
  const assigned = [];
  const listeners = new Map();
  let uuid = 0;
  const form = {
    values: new Map([['Наименование', 'Черновик']]),
    querySelectorAll() { return []; },
    querySelector() { return null; },
    appendChild() {},
  };
  const cancelLink = {
    dataset: {obCloseReason: 'close'},
    getAttribute(name) { return name === 'href' ? '/ui/catalog/Тест' : null; },
    closest(selector) { return selector === '[data-ob-close-tab]' ? this : null; },
    click() {
      document.dispatch('click', {
        target: this, button: 0, defaultPrevented: false,
        preventDefault() { this.defaultPrevented = true; },
      });
    },
  };
  const document = {
    getElementById(id) { return id === 'main-form' ? form : null; },
    createElement() { return {}; },
    querySelector(selector) { return selector === 'a.btn-cancel' ? cancelLink : null; },
    activeElement: null,
    addEventListener(type, fn) {
      if (!listeners.has(type)) listeners.set(type, []);
      listeners.get(type).push(fn);
    },
    dispatch(type, event) {
      for (const fn of listeners.get(type) || []) {
        fn(event);
        if (event.immediatePropagationStopped) break;
      }
    },
  };
  const context = {
    __cfg: Object.assign({
      kind: 'catalog',
      closeUrl: '/ui/catalog/Тест/form-close-intent',
      closeTimeoutMs: 5000,
      closeMessages: {
        controllerUnavailable: 'Close check is unavailable. The form remains open.',
        invalidResponse: 'The server returned an invalid close response.',
        correlationMismatch: 'The close response did not match the request.',
        formChanged: 'The form changed during the close check.',
        timeout: 'Close check timed out',
        network: 'Network error while closing',
      },
    }, cfgOverride),
    __form: form,
    __applied: applied,
    document,
    FormData: FakeFormData,
    URLSearchParams,
    AbortController,
    Promise,
    Math,
    setTimeout,
    clearTimeout,
    fetch: fetchImpl,
    location: {
      pathname: '/ui/catalog/Тест/new', origin: 'http://onebase.test',
      assign(href) { assigned.push(href); },
    },
    parent: null,
  };
  context.window = context;
  context.parent = context;
  context.crypto = {randomUUID() { uuid++; return `00000000-0000-4000-8000-${String(uuid).padStart(12, '0')}`; }};
  context.obManagedApplyTablePartRefOptions = (value) => applied.push(['refs', value]);
  context.applyTableParts = (value) => applied.push(['parts', value]);
  context.confirm = () => true;
  context.__obEmbedded = true;
  context._obGrids = {};

  const prefix = `
    (function(){
      var cfg = globalThis.__cfg;
      var DOC_ID = '';
      function serviceField(name){ return name; }
      async function awaitCurrentFileReads(){ return true; }
      function applyFormConditionalCSS(value){ globalThis.__applied.push(['css', value]); }
      function applyElementStates(value){ globalThis.__applied.push(['states', value]); }
      function applyValues(value){ globalThis.__applied.push(['values', value]); }
      function applyFormTables(value){ globalThis.__applied.push(['tables', value]); }
      function flash(value, kind){ globalThis.__applied.push(['flash', value, kind]); }
  `;
  vm.runInNewContext(prefix + controller + '\n})();', context, {filename: 'managed-close-controller.js'});
  vm.runInNewContext(escapeHandler, context, {filename: 'managed-close-escape.js'});
  return {context, form, applied, listeners, document, cancelLink, assigned};
}

function response(data, ok = true) {
  return {ok, async json() { return data; }};
}

test('close controller is single-flight and returns only its exact decision', async () => {
  let complete;
  const calls = [];
  const app = runtime((url, options) => {
    calls.push({url, options});
    return new Promise((resolve) => { complete = resolve; });
  });

  const first = app.context.obRequestFormClose({reason: 'cross'});
  const second = app.context.obRequestFormClose({reason: 'escape'});
  assert.equal(first, second);
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, '/ui/catalog/Тест/form-close-intent');
  const intent = calls[0].options.body.get('_close_intent_id');
  assert.equal(calls[0].options.body.get('_close_reason'), 'cross');
  assert.equal(calls[0].options.body.get('Наименование'), 'Черновик');

  complete(response({ok: true, values: {Наименование: 'Проверено'}, messages: ['готово'], close: {
    intentId: intent, allowed: true, saved: false,
  }}));
  const decision = await first;
  assert.equal(decision.allowed, true);
  assert.equal(decision.intentId, intent);
  assert.equal(decision.error, '');
  assert.ok(app.applied.some((entry) => entry[0] === 'values'));
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && entry[1] === 'готово'));
});

test('stale response fails closed and identical retry reuses the intent id', async () => {
  const calls = [];
  const answers = [
    (intent) => response({ok: true, close: {intentId: '00000000-0000-4000-8000-999999999999', allowed: true}}),
    (intent) => response({ok: true, close: {intentId: intent, allowed: true}}),
  ];
  const app = runtime(async (url, options) => {
    const intent = options.body.get('_close_intent_id');
    calls.push(intent);
    return answers.shift()(intent);
  });

  const denied = await app.context.obRequestFormClose({reason: 'cross'});
  assert.equal(denied.allowed, false);
  const allowed = await app.context.obRequestFormClose({reason: 'cross'});
  assert.equal(allowed.allowed, true);
  assert.equal(calls.length, 2);
  assert.equal(calls[1], calls[0], 'retry changed the exactly-once intent id');
});

test('network failure is fail-closed', async () => {
  const app = runtime(async () => { throw new Error('offline'); });
  const result = await app.context.obRequestFormClose({reason: 'close'});
  assert.equal(result.allowed, false);
  assert.equal(result.error, 'network');
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && /Network error while closing/.test(entry[1])));
});

test('missing close endpoint is fail-closed', async () => {
  const app = runtime(async () => { throw new Error('must not fetch'); }, {closeUrl: ''});
  const result = await app.context.obRequestFormClose({reason: 'close'});
  assert.equal(result.allowed, false);
  assert.equal(result.error, 'controller-not-ready');
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && /Close check is unavailable/.test(entry[1])));
});

test('editing after fetch starts rejects the stale decision without applying server state', async () => {
  let complete;
  const app = runtime(() => new Promise((resolve) => { complete = resolve; }));
  const pending = app.context.obRequestFormClose({reason: 'cross'});
  await Promise.resolve();
  await Promise.resolve();
  app.form.values.set('Наименование', 'Новый ввод');
  const intent = '00000000-0000-4000-8000-000000000001';
  complete(response({
    ok: true,
    values: {Наименование: 'Старый ответ'},
    messages: ['не применять'],
    close: {intentId: intent, allowed: true, saved: false},
  }));
  const decision = await pending;
  assert.equal(decision.allowed, false);
  assert.equal(decision.error, 'form-changed');
  assert.equal(app.form.values.get('Наименование'), 'Новый ввод');
  assert.equal(app.applied.some((entry) => entry[0] === 'values'), false);
  assert.equal(app.applied.some((entry) => entry[0] === 'flash' && entry[1] === 'не применять'), false);
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && /form changed/i.test(entry[1])));
});

test('server execution failure is retried with the same intent for the same snapshot', async () => {
  const calls = [];
  let attempt = 0;
  const app = runtime(async (url, options) => {
    const intent = options.body.get('_close_intent_id');
    calls.push(intent);
    attempt++;
    if (attempt === 1) return response({ok: false, error: 'timeout', close: {intentId: intent, allowed: false}});
    return response({ok: true, close: {intentId: intent, allowed: true}});
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, false);
  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, true);
  assert.equal(calls[1], calls[0]);
});

test('standalone bottom link and Escape use the close controller before navigation', async () => {
  const reasons = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    reasons.push(options.body.get('_close_reason'));
    return response({ok: true, close: {intentId, allowed: true}});
  });
  app.context.__obEmbedded = false;

  app.cancelLink.click();
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(reasons, ['close']);
  assert.deepEqual(app.assigned, ['/ui/catalog/Тест']);

  app.assigned.length = 0;
  app.document.dispatch('keydown', {
    key: 'Escape', keyCode: 27,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() { this.propagationStopped = true; },
    stopImmediatePropagation() { this.immediatePropagationStopped = true; },
  });
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(reasons, ['close', 'escape']);
  assert.deepEqual(app.assigned, ['/ui/catalog/Тест']);
});
