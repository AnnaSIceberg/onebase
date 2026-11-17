'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'ui.js'), 'utf8').replace(/\r\n/g, '\n');
function fn(name) {
  const start = source.indexOf('function ' + name + '(');
  const end = source.indexOf('\n}', start);
  assert.ok(start >= 0 && end > start, 'production function ' + name);
  return source.slice(start, end + 2);
}
function fixture() {
  const url = '/ui/document/fineprobe/11111111-1111-1111-1111-111111111111';
  const row = {dataset: {openUrl: url}, closest() { return null; }};
  const listeners = [];
  const sandbox = {
    window: {location: {href: ''}}, URLSearchParams,
    document: {addEventListener(kind, callback) { if (kind === 'keydown') listeners.push(callback); }},
    obInitListFocusSelection() {}, obHasBlockingModal() { return false; },
    obHandleDOMTableShortcut() { return false; }, obHandleListDeleteShortcut() {},
    obIsInteractiveTarget(target) { return target.tagName === 'INPUT'; },
    obListRows() { return [row]; }, obListCurrentRow() { return row; }, listSel() { return row; },
  };
  const begin = source.indexOf('// BEGIN onebase-row-url');
  const end = source.indexOf('// END onebase-row-url');
  vm.runInNewContext(source.slice(begin, end) + '\n' + ['listOpen', 'listActivateRow', 'listRowDblClick', 'obInitKeyboardShortcuts'].map(fn).join('\n'), sandbox);
  return {sandbox, row, url, listeners};
}
test('journal row without an action link still opens on double click', () => {
  const f = fixture();
  f.sandbox.listRowDblClick({target: {tagName: 'TD'}}, f.row);
  assert.equal(f.sandbox.window.location.href, f.url);
});
test('journal row without an action link still opens on Enter', () => {
  const f = fixture();
  f.sandbox.obInitKeyboardShortcuts();
  const event = {key: 'Enter', target: {tagName: 'TD'}, preventDefault() { this.defaultPrevented = true; }};
  f.listeners.forEach(callback => callback(event));
  assert.equal(f.sandbox.window.location.href, f.url);
  assert.equal(event.defaultPrevented, true);
});
test('Enter in an input does not open the selected journal row', () => {
  const f = fixture();
  f.sandbox.obInitKeyboardShortcuts();
  const event = {key: 'Enter', target: {tagName: 'INPUT'}, preventDefault() { this.defaultPrevented = true; }};
  f.listeners.forEach(callback => callback(event));
  assert.equal(f.sandbox.window.location.href, '');
  assert.equal(event.defaultPrevented, undefined);
});
