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

function runtime(fetchImpl) {
  const applied = [];
  const listeners = new Map();
  let uuid = 0;
  const form = {
    values: new Map([['Наименование', 'Черновик']]),
    querySelectorAll() { return []; },
    querySelector() { return null; },
    appendChild() {},
  };
  const document = {
    getElementById(id) { return id === 'main-form' ? form : null; },
    createElement() { return {}; },
    addEventListener(type, fn) { listeners.set(type, fn); },
  };
  const context = {
    __cfg: {kind: 'catalog', closeUrl: '/ui/catalog/Тест/form-close-intent', closeTimeoutMs: 5000},
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
    location: {pathname: '/ui/catalog/Тест/new', origin: 'http://onebase.test'},
    parent: null,
  };
  context.window = context;
  context.parent = context;
  context.crypto = {randomUUID() { uuid++; return `00000000-0000-4000-8000-${String(uuid).padStart(12, '0')}`; }};
  context.obManagedApplyTablePartRefOptions = (value) => applied.push(['refs', value]);
  context.applyTableParts = (value) => applied.push(['parts', value]);
  context.confirm = () => true;
  context.__obEmbedded = true;

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
  return {context, form, applied, listeners};
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
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && /Сетевая ошибка/.test(entry[1])));
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
