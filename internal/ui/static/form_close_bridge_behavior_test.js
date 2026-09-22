'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const ui = fs.readFileSync('static/ui.js', 'utf8');
const marker = '// Same-origin request/decision protocol shared by shell tabs and reference';
const start = ui.indexOf(marker);
const end = ui.indexOf('if (window.__obEmbedded)', start);
assert.ok(start >= 0 && end > start, 'form close bridge slice not found');
const bridge = ui.slice(start, end);

function runtime(managed = false) {
  const listeners = [];
  let uuid = 0;
  const context = {
    Promise,
    Math,
    Date,
    location: {origin: 'http://onebase.test'},
    document: {getElementById(id) { return managed && id === 'ob-managed-config' ? {} : null; }},
    crypto: {randomUUID() { uuid++; return `id-${uuid}`; }},
    setTimeout() { return 1; },
    clearTimeout() {},
    addEventListener(type, fn) { if (type === 'message') listeners.push(fn); },
  };
  context.window = context;
  vm.runInNewContext(bridge, context, {filename: 'form-close-bridge.js'});
  return {context, message(ev) { for (const listener of listeners) listener(ev); }};
}

test('parent accepts a close decision only from the requested same-origin frame', async () => {
  const app = runtime();
  const sent = [];
  const child = {postMessage(data, origin) { sent.push({data, origin}); }};
  const frame = {contentWindow: child};
  const decisionPromise = app.context.obRequestFrameClose(frame, 'escape');
  assert.equal(sent.length, 1);
  assert.equal(sent[0].origin, 'http://onebase.test');
  assert.equal(sent[0].data.reason, 'escape');
  const correlation = sent[0].data.correlation;

  app.message({origin: 'https://evil.test', source: child, data: {
    source: 'obFormCloseDecision', correlation, allowed: true,
  }});
  app.message({origin: 'http://onebase.test', source: {}, data: {
    source: 'obFormCloseDecision', correlation, allowed: true,
  }});
  let settled = false;
  decisionPromise.then(() => { settled = true; });
  await Promise.resolve();
  assert.equal(settled, false, 'spoofed decision resolved the request');

  app.message({origin: 'http://onebase.test', source: child, data: {
    source: 'obFormCloseDecision', correlation, allowed: false, intentId: 'server-id', error: 'cancelled',
  }});
  const decision = await decisionPromise;
  assert.equal(decision.allowed, false);
  assert.equal(decision.intentId, 'server-id');
  assert.equal(decision.error, 'cancelled');
});

test('child delegates to the managed controller and replies to the exact requester origin', async () => {
  const app = runtime(true);
  app.context.obRequestFormClose = async ({reason}) => ({allowed: reason === 'cross', intentId: 'intent-1'});
  const replies = [];
  const parent = {postMessage(data, origin) { replies.push({data, origin}); }};
  app.message({origin: 'http://onebase.test', source: parent, data: {
    source: 'obRequestFormClose', correlation: 'corr-1', reason: 'cross',
  }});
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(replies.length, 1);
  assert.equal(replies[0].origin, 'http://onebase.test');
  assert.equal(replies[0].data.source, 'obFormCloseDecision');
  assert.equal(replies[0].data.correlation, 'corr-1');
  assert.equal(replies[0].data.allowed, true);
  assert.equal(replies[0].data.intentId, 'intent-1');
});

test('managed page without a ready controller fails closed', async () => {
  const app = runtime(true);
  const replies = [];
  const parent = {postMessage(data, origin) { replies.push({data, origin}); }};
  app.message({origin: 'http://onebase.test', source: parent, data: {
    source: 'obRequestFormClose', correlation: 'corr-2', reason: 'cross',
  }});
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(replies[0].data.allowed, false);
  assert.equal(replies[0].data.error, 'controller-not-ready');
});
