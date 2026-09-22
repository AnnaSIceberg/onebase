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
	const dirtyReports = [];
	let embeddedDirty = false;
  const listeners = new Map();
  let uuid = 0;
  const idInput = {value: ''};
  const versionInput = {value: ''};
  const form = {
    id: 'main-form',
    values: new Map([['Наименование', 'Черновик']]),
    querySelectorAll() { return []; },
    querySelector(selector) {
      if (selector === '[name="_id"]') return idInput;
      if (selector === '[name="_version"]') return versionInput;
      return null;
    },
    appendChild() {},
  };
  const closeButton = {
    tagName: 'BUTTON',
    dataset: {obCloseReason: 'close'},
    getAttribute(name) {
      if (name === 'data-ob-form-close') return '';
      if (name === 'data-ob-close-href') return '/ui/catalog/Тест';
      return null;
    },
    hasAttribute(name) { return name === 'data-ob-form-close'; },
    closest(selector) { return selector.indexOf('[data-ob-form-close]') >= 0 ? this : null; },
    click() {
      Promise.resolve(context.obRequestFormClose({reason: this.dataset.obCloseReason || 'close'})).then((decision) => {
        if (!decision || !decision.allowed) return;
        context.obFinalizeFormClose();
        context.location.assign(this.getAttribute('data-ob-close-href'));
      });
    },
  };
  function fakeElement(tag) {
    const attrs = new Map();
    const elementListeners = new Map();
    return {
      tagName: String(tag || '').toUpperCase(), id: '', type: '', textContent: '', style: {}, children: [], parentElement: null,
      setAttribute(name, value) { attrs.set(String(name), String(value)); if (name === 'id') this.id = String(value); },
      getAttribute(name) { return attrs.has(String(name)) ? attrs.get(String(name)) : null; },
      appendChild(child) { this.children.push(child); child.parentElement = this; return child; },
      addEventListener(type, fn) { if (!elementListeners.has(type)) elementListeners.set(type, []); elementListeners.get(type).push(fn); },
      dispatch(type, event) { for (const fn of elementListeners.get(type) || []) fn(event); },
      closest(selector) { return selector === '[data-ob-close-choice]' && attrs.has('data-ob-close-choice') ? this : null; },
      focus() { document.activeElement = this; },
      remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter((child) => child !== this); this.parentElement = null; },
    };
  }
  const body = fakeElement('body');
  function findById(root, id) {
    if (!root) return null;
    if (root.id === id) return root;
    for (const child of root.children || []) { const found = findById(child, id); if (found) return found; }
    return null;
  }
  const document = {
    body,
    getElementById(id) { return id === 'main-form' ? form : findById(body, id); },
    createElement(tag) { return fakeElement(tag); },
    querySelector(selector) {
      return selector === 'button[data-ob-form-close=""], [data-ob-close-tab], a.btn-cancel' ? closeButton : null;
    },
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
	obSetEmbeddedDirty(value) {
	  value = !!value;
	  if (value === embeddedDirty) return;
	  embeddedDirty = value;
	  dirtyReports.push(value);
	},
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
	return {context, form, idInput, versionInput, applied, listeners, document, closeButton, assigned, dirtyReports};
}

function response(data, ok = true, status = ok ? 200 : 500) {
  return {ok, status, async json() { return data; }};
}

test('close controller is single-flight and returns only its exact decision', async () => {
  let complete;
  const calls = [];
  const app = runtime((url, options) => {
    calls.push({url, options});
    return new Promise((resolve) => { complete = resolve; });
  });

  const first = app.context.obRequestFormClose({reason: 'cross', mode: 'discard'});
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

test('dirty close modal maps Cancel/No/Yes to zero HTTP, discard and save', async () => {
  const modes = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    modes.push(options.body.get('_close_mode'));
    return response({ok: true, close: {intentId, allowed: true, saved: modes.at(-1) === 'save'}});
  });
  app.context._obFormDirty = true;

  const escaped = app.context.obRequestFormClose({reason: 'cross'});
  await Promise.resolve();
  app.document.dispatch('keydown', {
    key: 'Escape', keyCode: 27,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() { this.propagationStopped = true; },
    stopImmediatePropagation() { this.immediatePropagationStopped = true; },
  });
  assert.equal((await escaped).error, 'user-cancelled');
  assert.deepEqual(modes, []);

  const cancelled = app.context.obRequestFormClose({reason: 'cross'});
  await Promise.resolve();
  let modal = app.document.getElementById('ob-managed-close-confirm');
  assert.ok(modal);
  let row = modal.children[0].children[2];
  const buttons = row.children;
  row.dispatch('click', {target: buttons[2]});
  assert.equal((await cancelled).error, 'user-cancelled');
  assert.deepEqual(modes, []);

  const discarded = app.context.obRequestFormClose({reason: 'close'});
  await Promise.resolve();
  modal = app.document.getElementById('ob-managed-close-confirm');
  row = modal.children[0].children[2];
  row.dispatch('click', {target: row.children[1]});
  assert.equal((await discarded).allowed, true);

  app.context._obFormDirty = true;
  const saved = app.context.obRequestFormClose({reason: 'close'});
  await Promise.resolve();
  modal = app.document.getElementById('ob-managed-close-confirm');
  row = modal.children[0].children[2];
  row.dispatch('click', {target: row.children[0]});
  assert.equal((await saved).allowed, true);
  assert.deepEqual(modes, ['discard', 'save']);
});

test('dirty processor offers an explicit close-without-saving choice and never sends save', async () => {
	const modes = [];
	const app = runtime(async (url, options) => {
	  const intentId = options.body.get('_close_intent_id');
	  modes.push(options.body.get('_close_mode'));
	  return response({ok: true, close: {intentId, allowed: true, saved: false}});
	}, {kind: 'processor', closeMessages: {
	  processorQuestion: 'Close without saving?',
	  processorDiscard: 'Close',
	  cancel: 'Cancel',
	}});
	app.context._obFormDirty = true;
	const pending = app.context.obRequestFormClose({reason: 'close'});
	await Promise.resolve();
	const modal = app.document.getElementById('ob-managed-close-confirm');
	assert.equal(modal.children[0].children[1].textContent, 'Close without saving?');
	const row = modal.children[0].children[2];
	assert.equal(row.children.length, 2, 'processor prompt exposed a save button');
	assert.deepEqual(row.children.map((button) => button.textContent), ['Close', 'Cancel']);
	row.dispatch('click', {target: row.children[0]});
	assert.equal((await pending).allowed, true);
	assert.deepEqual(modes, ['discard']);
});

test('clean discard becomes dirty when denied BeforeClose returns unsaved state', async () => {
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    return response({
      ok: true,
      values: {Наименование: 'Изменено обработчиком'},
      dirty: true,
      close: {intentId, allowed: false, saved: false},
    });
  });
  assert.equal(app.context._obFormDirty, undefined);
  const decision = await app.context.obRequestFormClose({reason: 'close', mode: 'discard'});
  assert.equal(decision.allowed, false);
  assert.equal(app.context._obFormDirty, true);
  assert.deepEqual(app.dirtyReports, [true]);

  const choice = app.context.obManagedCloseChoice();
  await Promise.resolve();
  assert.ok(app.document.getElementById('ob-managed-close-confirm'), 'next close did not protect handler state');
  const modal = app.document.getElementById('ob-managed-close-confirm');
  const row = modal.children[0].children[2];
  row.dispatch('click', {target: row.children[2]});
  assert.equal(await choice, 'cancel');
});

test('unsaved dirty=false response cannot clear pre-existing dirty state', async () => {
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    return response({ok: true, dirty: false, close: {intentId, allowed: false, saved: false}});
  });
  app.context._obFormDirty = true;
  const pending = app.context.obRequestFormClose({reason: 'close'});
  await Promise.resolve();
  const modal = app.document.getElementById('ob-managed-close-confirm');
  const row = modal.children[0].children[2];
  row.dispatch('click', {target: row.children[1]});
  await pending;
  assert.equal(app.context._obFormDirty, true);
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

test('network failure is fail-closed and its unknown result keeps the intent id', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    calls.push(intentId);
    if (calls.length === 1) throw new Error('offline');
    return response({ok: true, close: {intentId, allowed: true}});
  });

  const result = await app.context.obRequestFormClose({reason: 'close'});
  assert.equal(result.allowed, false);
  assert.equal(result.error, 'network');
  assert.ok(app.applied.some((entry) => entry[0] === 'flash' && /Network error while closing/.test(entry[1])));
  assert.equal((await app.context.obRequestFormClose({reason: 'close'})).allowed, true);
  assert.equal(calls[1], calls[0]);
});

test('unknown discard cannot authorize a later explicit save request', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) throw new Error('response lost');
    return response({ok: true, close: {intentId: body.get('_close_intent_id'), allowed: true, saved: body.get('_close_mode') === 'save'}});
  });
  assert.equal((await app.context.obRequestFormClose({reason: 'close', mode: 'discard'})).allowed, false);
  const recovered = await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(recovered.allowed, false, 'old discard decision authorized the new save adapter');
  assert.equal(recovered.error, 'form-changed');
  assert.deepEqual(calls.slice(0, 2).map((body) => body.get('_close_mode')), ['discard', 'discard']);
  assert.equal(calls[1].get('_close_intent_id'), calls[0].get('_close_intent_id'));
  await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(calls[2].get('_close_mode'), 'save');
  assert.notEqual(calls[2].get('_close_intent_id'), calls[1].get('_close_intent_id'));
});

test('unknown new close-save blocks native submit until exact replay adopts identity', async () => {
  const calls = [];
  const savedId = '11111111-1111-4111-8111-111111111111';
  const app = runtime(async (url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) throw new Error('response lost after commit');
    return response({
      ok: true, savedId, version: 1, dirty: false,
      close: {intentId: body.get('_close_intent_id'), allowed: true, saved: true},
    });
  });
  assert.equal((await app.context.obRequestFormClose({reason: 'ok', mode: 'save'})).allowed, false);

  const submit = {
    target: app.form, defaultPrevented: false, immediatePropagationStopped: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() { this.propagationStopped = true; },
    stopImmediatePropagation() { this.immediatePropagationStopped = true; },
  };
  app.document.dispatch('submit', submit);
  assert.equal(submit.defaultPrevented, true, 'native /new submit was not blocked');
  for (let i = 0; i < 8 && calls.length < 2; i++) await Promise.resolve();
  assert.equal(calls.length, 2, 'unknown close intent was not replayed');
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls[1].get('_close_intent_id'), calls[0].get('_close_intent_id'));
  assert.equal(app.idInput.value, savedId);
  assert.equal(app.versionInput.value, '1');

  const nextSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', nextSubmit);
  assert.equal(nextSubmit.defaultPrevented, false, 'submit stayed blocked after terminal identity recovery');
});

test('unknown discard blocks native submit until handler-written identity is recovered', async () => {
  const calls = [];
  const savedId = '22222222-2222-4222-8222-222222222222';
  const app = runtime(async (url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) throw new Error('discard response lost after BeforeClose write');
    return response({
      ok: true, savedId, version: 1, dirty: false,
      close: {intentId: body.get('_close_intent_id'), allowed: false, saved: true},
    });
  });
  assert.equal((await app.context.obRequestFormClose({reason: 'close', mode: 'discard'})).allowed, false);

  const submit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', submit);
  assert.equal(submit.defaultPrevented, true, 'native /new submit bypassed unknown discard outcome');
  for (let i = 0; i < 8 && calls.length < 2; i++) await Promise.resolve();
  assert.equal(calls.length, 2, 'unknown discard intent was not replayed');
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls[1].get('_close_mode'), 'discard');
  assert.equal(calls[1].get('_close_intent_id'), calls[0].get('_close_intent_id'));
  assert.equal(calls[1].toString(), calls[0].toString(), 'discard replay body changed');
  assert.equal(app.idInput.value, savedId);
  assert.equal(app.versionInput.value, '1');

  const nextSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', nextSubmit);
  assert.equal(nextSubmit.defaultPrevented, false, 'submit stayed blocked after discard identity recovery');
});

test('native submit cannot race an in-flight close save', async () => {
  let finishRequest;
  let closeBody;
  const savedId = '33333333-3333-4333-8333-333333333333';
  const app = runtime(async (url, options) => {
    closeBody = new URLSearchParams(options.body.toString());
    return new Promise((resolve) => {
      finishRequest = () => resolve(response({
        ok: true, savedId, version: 1, dirty: false,
        close: {intentId: closeBody.get('_close_intent_id'), allowed: false, saved: true},
      }));
    });
  });
  const closeRequest = app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  for (let i = 0; i < 8 && !finishRequest; i++) await Promise.resolve();
  assert.equal(typeof finishRequest, 'function', 'close request did not enter fetch');

  const racingSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', racingSubmit);
  assert.equal(racingSubmit.defaultPrevented, true, 'parallel native submit raced the close intent');

  finishRequest();
  const decision = await closeRequest;
  assert.equal(decision.allowed, false);
  assert.equal(app.idInput.value, savedId);
  assert.equal(app.versionInput.value, '1');

  const nextSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', nextSubmit);
  assert.equal(nextSubmit.defaultPrevented, false, 'native submit stayed blocked after terminal close response');
});

test('native submit cannot race the embedded parent handoff window', async () => {
  let resolveFetch;
  let body;
  const app = runtime(async (_url, options) => {
    body = new URLSearchParams(options.body.toString());
    return new Promise((resolve) => { resolveFetch = resolve; });
  });
  app.context.obBeginManagedCloseHandoff();
  const racingSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', racingSubmit);
  assert.equal(racingSubmit.defaultPrevented, true, 'native submit escaped before parent close request arrived');

  const request = app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  for (let i = 0; i < 8 && !resolveFetch; i++) await Promise.resolve();
  assert.equal(typeof resolveFetch, 'function');
  resolveFetch(response({
    ok: true, dirty: false, savedId: 'handoff-saved', version: 1,
    close: {intentId: body.get('_close_intent_id'), allowed: false, saved: true},
  }));
  await request;

  const nextSubmit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() {},
  };
  app.document.dispatch('submit', nextSubmit);
  assert.equal(nextSubmit.defaultPrevented, false, 'handoff flag survived the terminal controller response');
});

test('managed popup implicit submit uses the exact save-and-select controller', async () => {
  const calls = [];
  const posted = [];
  const app = runtime(async (_url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    return response({
      ok: true, savedId: 'popup-created', savedLabel: 'Created', version: 1, dirty: false,
      close: {intentId: body.get('_close_intent_id'), allowed: true, saved: true},
    });
  });
  app.context.location.search = '?_popup=1';
  app.context.obManagedPostRefSaved = decision => { posted.push(decision); return true; };
  const submit = {
    target: app.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() { this.immediatePropagationStopped = true; },
  };
  app.document.dispatch('submit', submit);
  assert.equal(submit.defaultPrevented, true, 'popup Enter fell through to legacy HTML POST');
  for (let i = 0; i < 8 && !posted.length; i++) await Promise.resolve();
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls.length, 1);
  assert.equal(calls[0].get('_close_mode'), 'save_and_select');
  assert.equal(calls[0].get('_close_reason'), 'ok');
  assert.equal(posted.length, 1);
  assert.equal(posted[0].intentId, calls[0].get('_close_intent_id'));
  assert.equal(posted[0].savedId, 'popup-created');

  const deniedPosts = [];
  const denied = runtime(async (_url, options) => response({
    ok: true, dirty: true,
    close: {intentId: options.body.get('_close_intent_id'), allowed: false, saved: false},
  }));
  denied.context.location.search = '?_popup=1';
  denied.context.obManagedPostRefSaved = decision => { deniedPosts.push(decision); return true; };
  const deniedSubmit = {
    target: denied.form, defaultPrevented: false,
    preventDefault() { this.defaultPrevented = true; },
    stopPropagation() {}, stopImmediatePropagation() { this.immediatePropagationStopped = true; },
  };
  denied.document.dispatch('submit', deniedSubmit);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(deniedSubmit.defaultPrevented, true);
  assert.equal(deniedPosts.length, 0, 'denied popup save published a selection result');
});

test('popup implicit submit publishes an exact recovered save-and-select result', async () => {
  const calls = [];
  const posted = [];
  const app = runtime(async (_url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) throw new Error('popup response lost after commit');
    return response({
      ok: true, savedId: 'popup-recovered', savedLabel: 'Recovered', version: 1, dirty: false,
      close: {intentId: body.get('_close_intent_id'), allowed: true, saved: true},
    });
  });
  app.context.location.search = '?_popup=1';
  app.context.obManagedPostRefSaved = decision => { posted.push(decision); return true; };
  function submit() {
    const event = {
      target: app.form, defaultPrevented: false,
      preventDefault() { this.defaultPrevented = true; },
      stopPropagation() {}, stopImmediatePropagation() { this.immediatePropagationStopped = true; },
    };
    app.document.dispatch('submit', event);
    return event;
  }

  assert.equal(submit().defaultPrevented, true);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls.length, 1);
  assert.equal(posted.length, 0);
  assert.equal(submit().defaultPrevented, true);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls.length, 2);
  assert.equal(calls[1].toString(), calls[0].toString(), 'popup replay changed UUID/body');
  assert.equal(posted.length, 1);
  assert.equal(posted[0].savedId, 'popup-recovered');
});

test('save modes require a confirmed saved result before allowing destruction', async () => {
  for (const mode of ['save', 'post', 'save_and_select']) {
    const app = runtime(async (url, options) => response({
      ok: true,
      close: {intentId: options.body.get('_close_intent_id'), allowed: true, saved: false},
    }));
    const decision = await app.context.obRequestFormClose({reason: 'ok', mode});
    assert.equal(decision.allowed, false, `${mode} accepted allowed=true without saved=true`);
  }
  const discard = runtime(async (url, options) => response({
    ok: true,
    close: {intentId: options.body.get('_close_intent_id'), allowed: true, saved: false},
  }));
  assert.equal((await discard.context.obRequestFormClose({reason: 'close', mode: 'discard'})).allowed, true);
});

test('unknown new save is recovered before edited snapshot and retry updates its identity', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) throw new Error('response lost after commit');
    return response({
      ok: true, dirty: false, savedId: 'saved-after-loss', version: calls.length,
      close: {intentId: body.get('_close_intent_id'), allowed: true, saved: true}
    });
  });
  assert.equal((await app.context.obRequestFormClose({reason: 'ok', mode: 'save'})).allowed, false);
  app.form.values.set('Наименование', 'edited after unknown');
  const recovered = await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(recovered.allowed, false);
  assert.equal(calls[1].get('_close_intent_id'), calls[0].get('_close_intent_id'));
  await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(calls[2].get('_id'), 'saved-after-loss');
  assert.notEqual(calls[2].get('_close_intent_id'), calls[1].get('_close_intent_id'));
});

test('client timeout keeps the intent id because the result is unknown', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    calls.push(intentId);
    if (calls.length === 1) {
      const err = new Error('aborted');
      err.name = 'AbortError';
      throw err;
    }
    return response({ok: true, close: {intentId, allowed: true}});
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'close'})).error, 'timeout');
  assert.equal((await app.context.obRequestFormClose({reason: 'close'})).allowed, true);
  assert.equal(calls[1], calls[0]);
});

test('received invalid JSON keeps the unknown intent id', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    calls.push(intentId);
    if (calls.length === 1) {
      return {ok: false, status: 500, async json() { throw new SyntaxError('invalid JSON'); }};
    }
    return response({ok: true, close: {intentId, allowed: true}});
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'close'})).allowed, false);
  assert.equal((await app.context.obRequestFormClose({reason: 'close'})).allowed, true);
  assert.equal(calls[1], calls[0], 'unparseable response lost the exactly-once intent id');
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
  const pending = app.context.obRequestFormClose({reason: 'cross', mode: 'discard'});
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

test('editing during save keeps the edited field, merges untouched server fields and adopts identity', async () => {
  let complete;
  const calls = [];
  const app = runtime((url, options) => {
    calls.push(options.body);
    if (calls.length === 1) return new Promise((resolve) => { complete = resolve; });
    const intentId = options.body.get('_close_intent_id');
    return Promise.resolve(response({ok: true, close: {intentId, allowed: true, saved: true}, savedId: 'saved-42', version: 2}));
  });
	app.form.values.set('Статус', 'Исходный');
	app.form.values.set('tp_json.ЛокальноИзменённая', '[{"Значение":"до"}]');
  const pending = app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  await Promise.resolve(); await Promise.resolve();
  app.form.values.set('Наименование', 'Ввод после клика');
	app.form.values.set('tp_json.ЛокальноИзменённая', '[{"Значение":"после"}]');
  const firstIntent = calls[0].get('_close_intent_id');
  complete(response({
	ok: true,
	values: {Наименование: 'Сохранённый снимок', Статус: 'Изменён сервером'},
	refOptions: {Наименование: [{id: 'stale'}], Статус: [{id: 'fresh'}]},
	tpRefOptions: {ЛокальноИзменённая: {Ссылка: [{id: 'stale'}]}},
	elementStates: {readonly: {Наименование: true}}, conditionalCss: '.stale{display:none}',
	savedId: 'saved-42', version: 1,
    close: {intentId: firstIntent, allowed: true, saved: true, formUrl: '/ui/catalog/Тест/saved-42'},
  }));
  const decision = await pending;
  assert.equal(decision.allowed, false);
  assert.equal(decision.error, 'form-changed');
  assert.equal(app.form.values.get('Наименование'), 'Ввод после клика');
	const merged = app.applied.find((entry) => entry[0] === 'values');
	assert.equal(JSON.stringify(merged && merged[1]), JSON.stringify({Статус: 'Изменён сервером'}));
	assert.equal(app.applied.some((entry) => entry[0] === 'css' || entry[0] === 'states'), false, 'stale derived state was applied after concurrent edit');
	const refs = app.applied.find((entry) => entry[0] === 'refs');
	assert.equal(JSON.stringify(refs && refs[1]), '{}', 'stale table-part ref options were applied after concurrent grid edit');

  await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(calls[1].get('_id'), 'saved-42', 'retry would create a duplicate instead of updating saved record');
});

test('saved denied close clears shell dirty only for durable state and next edit reports dirty again', async () => {
	const app = runtime(async (url, options) => {
	  const intentId = options.body.get('_close_intent_id');
	  return response({ok: true, dirty: false, close: {intentId, allowed: false, saved: true}, savedId: 'saved-1', version: 1});
	});
	app.context._obFormDirty = true;
	app.context.obSetEmbeddedDirty(true);
	const decision = await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
	assert.equal(decision.allowed, false);
	assert.equal(app.context._obFormDirty, false);
	app.context.obSetManagedFormDirty(true);
	assert.deepEqual(app.dirtyReports, [true, false, true]);
});

test('saved close keeps returned unsaved BeforeClose mutations dirty', async () => {
	const app = runtime(async (url, options) => {
	  const intentId = options.body.get('_close_intent_id');
	  return response({ok: false, dirty: true, values: {Статус: 'Не записан'}, error: 'boom', close: {intentId, allowed: false, saved: true}, savedId: 'saved-1', version: 1}, false, 500);
	});
	app.context._obFormDirty = true;
	await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
	assert.equal(app.context._obFormDirty, true);
	assert.ok(app.applied.some((entry) => entry[0] === 'values' && entry[1].Статус === 'Не записан'));
});

test('denied popup save keeps popup marker when adopting the created identity', async () => {
  let replaced = '';
  const bodies = [];
  const app = runtime(async (_url, options) => {
    bodies.push(options.body);
    const intentId = options.body.get('_close_intent_id');
    return response({
      ok: true, dirty: false, savedId: 'saved-popup', version: 1,
      close: {intentId, allowed: false, saved: true},
    });
  });
  app.context.location.search = '?_popup=1';
  app.context.history = {replaceState(_state, _title, url) { replaced = url; }};

  const decision = await app.context.obRequestFormClose({reason: 'ok', mode: 'save_and_select'});
  assert.equal(decision.allowed, false);
  assert.equal(replaced, '/ui/catalog/Тест/saved-popup?_popup=1');
  assert.equal(app.idInput.value, 'saved-popup');
  assert.equal(app.versionInput.value, '1');

  await app.context.obRequestFormClose({reason: 'ok', mode: 'save_and_select'});
  assert.equal(bodies[1].get('_close_mode'), 'save_and_select');
  assert.equal(bodies[1].get('_id'), 'saved-popup');
});

test('terminal saved identity survives a throwing response renderer', async () => {
  const bodies = [];
  let replaced = '';
  const app = runtime(async (_url, options) => {
    bodies.push(options.body);
    const intentId = options.body.get('_close_intent_id');
    return response({
      ok: true, tableparts: {Rows: []}, dirty: false,
      savedId: 'saved-before-render', version: 7,
      close: {intentId, allowed: false, saved: true},
    });
  });
  app.context.history = {replaceState(_state, _title, url) { replaced = url; }};
  app.context.applyTableParts = () => { throw new Error('renderer failed'); };

  const first = await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(first.allowed, false);
  assert.equal(first.error, 'network');
  assert.equal(app.idInput.value, 'saved-before-render');
  assert.equal(app.versionInput.value, '7');
  assert.equal(replaced, '/ui/catalog/Тест/saved-before-render');
  assert.equal(app.dirtyReports.at(-1), true, 'render failure did not leave the saved form fail-safe dirty');

  await app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  assert.equal(bodies[1].get('_id'), 'saved-before-render');
});

test('authoritative dirty survives a throwing close response renderer', async () => {
  const app = runtime(async (_url, options) => response({
    ok: false, dirty: true, tableparts: {Rows: []}, error: 'denied',
    close: {intentId: options.body.get('_close_intent_id'), allowed: false, saved: false},
  }, false, 500));
  app.context._obFormDirty = false;
  app.context.applyTableParts = () => { throw new Error('renderer failed after unsaved mutation'); };

  const decision = await app.context.obRequestFormClose({reason: 'close', mode: 'discard'});
  assert.equal(decision.allowed, false);
  assert.equal(decision.error, 'network');
  assert.equal(app.context._obFormDirty, true);
  assert.equal(app.dirtyReports.at(-1), true, 'renderer exception erased authoritative dirty state');
});

test('known terminal failure releases the intent under real replay semantics', async () => {
  const calls = [];
  const replay = new Map();
  let executions = 0;
  const app = runtime(async (url, options) => {
    const intent = options.body.get('_close_intent_id');
    calls.push(intent);
    if (!replay.has(intent)) {
      executions++;
      replay.set(intent, executions === 1
        ? {ok: false, error: 'handler failed', close: {intentId: intent, allowed: false}}
        : {ok: true, close: {intentId: intent, allowed: true}});
    }
    return response(replay.get(intent));
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, false);
  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, true);
  assert.notEqual(calls[1], calls[0], 'new click reused a terminal replay UUID');
  assert.equal(executions, 2, 'new click did not execute a fresh close intent');
});

test('correlated terminal HTTP error releases the intent id', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    calls.push(intentId);
    if (calls.length === 1) {
      return response({ok: false, error: 'handler failed', close: {intentId, allowed: false}}, false, 500);
    }
    return response({ok: true, close: {intentId, allowed: true}});
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, false);
  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, true);
  assert.notEqual(calls[1], calls[0]);
});

test('pending-leader timeout keeps the intent because no terminal replay exists yet', async () => {
  const calls = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    calls.push(intentId);
    if (calls.length === 1) {
      return response({
        ok: false,
        error: 'waiting for the original close request was cancelled',
        close: {intentId, allowed: false},
      }, false, 408);
    }
    return response({ok: true, close: {intentId, allowed: true}});
  });

  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, false);
  assert.equal((await app.context.obRequestFormClose({reason: 'cross'})).allowed, true);
  assert.equal(calls[1], calls[0], 'pending replay wait changed the close intent id');
});

test('408 plus concurrent edit retains the original intent until terminal replay', async () => {
  const calls = [];
  let complete;
  const app = runtime(async (url, options) => {
    const body = new URLSearchParams(options.body.toString());
    calls.push(body);
    if (calls.length === 1) return new Promise((resolve) => { complete = resolve; });
    return response({ok: true, close: {intentId: body.get('_close_intent_id'), allowed: true, saved: false}});
  });
  const first = app.context.obRequestFormClose({reason: 'cross', mode: 'discard'});
  await Promise.resolve(); await Promise.resolve();
  app.form.values.set('Наименование', 'edited while waiting');
  const intent = calls[0].get('_close_intent_id');
  complete(response({ok: false, error: 'still pending', close: {intentId: intent, allowed: false, saved: false}}, false, 408));
  assert.equal((await first).allowed, false);
  const recovered = await app.context.obRequestFormClose({reason: 'cross', mode: 'discard'});
  assert.equal(recovered.allowed, false, 'terminal replay authorized a changed snapshot');
  assert.equal(calls[1].get('_close_intent_id'), intent, '408 lost the original UUID after edit');
  await app.context.obRequestFormClose({reason: 'cross', mode: 'discard'});
  assert.notEqual(calls[2].get('_close_intent_id'), intent, 'terminal replay did not release the UUID');
});

test('failed current re-snapshot adopts saved identity without applying stale mutable state', async () => {
  let complete;
  const app = runtime(() => new Promise((resolve) => { complete = resolve; }));
  const pending = app.context.obRequestFormClose({reason: 'ok', mode: 'save'});
  await Promise.resolve(); await Promise.resolve();
  app.context.obGridSync = () => false;
  complete(response({
    ok: true, values: {Наименование: 'stale'}, tableparts: {Lines: [{Value: 'stale'}]},
    elementStates: {readonly: {Наименование: true}}, conditionalCss: '.stale{display:none}',
    savedId: 'saved-null-snapshot', version: 1,
    close: {intentId: '00000000-0000-4000-8000-000000000001', allowed: true, saved: true}
  }));
  const decision = await pending;
  assert.equal(decision.allowed, false);
  assert.equal(decision.error, 'form-changed');
  assert.equal(app.applied.some((entry) => ['values', 'parts', 'states', 'css'].includes(entry[0])), false);
  assert.equal(app.context._obFormDirty, true);
});

test('standalone bottom link and Escape use the close controller before navigation', async () => {
  const reasons = [];
  const app = runtime(async (url, options) => {
    const intentId = options.body.get('_close_intent_id');
    reasons.push(options.body.get('_close_reason'));
    return response({ok: true, close: {intentId, allowed: true}});
  });
  app.context.__obEmbedded = false;

  app.closeButton.click();
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
