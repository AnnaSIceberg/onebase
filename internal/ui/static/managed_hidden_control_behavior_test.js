'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');

const source = fs.readFileSync('static/managed.js', 'utf8');
const start = source.indexOf('function applyElementStates(');
assert.ok(start >= 0);
let depth = 0;
let end = -1;
for (let i = source.indexOf('{', start); i < source.length; i++) {
  if (source[i] === '{') depth++;
  if (source[i] === '}' && --depth === 0) { end = i + 1; break; }
}
assert.ok(end > start);

function fieldset(initiallyHidden, value) {
  return {
    style: {display: initiallyHidden ? 'none' : ''},
    disabled: initiallyHidden,
    value,
    getAttribute(name) { return name === 'data-ob-control-fieldset' ? '1' : null; },
    querySelectorAll() { return []; },
  };
}

test('hidden_when excludes duplicate controls from submission and required validation', () => {
  const fields = {
    Old: fieldset(true, 'old value'),
    Current: fieldset(false, 'new value'),
  };
  const document = {
    querySelector(selector) {
      const match = /^\[data-ob-el="([^"]+)"\]$/.exec(selector);
      return match ? fields[match[1]] : null;
    },
  };
  const window = {CSS: null};
  const applyElementStates = new Function('window', 'document', 'CSS',
    source.slice(start, end) + '\nreturn applyElementStates;')(window, document, null);
  const submitted = () => Object.values(fields).filter((field) => !field.disabled).map((field) => field.value);

  assert.deepEqual(submitted(), ['new value']);
  applyElementStates({hidden: {Old: false, Current: true}});
  assert.equal(fields.Old.style.display, '');
  assert.equal(fields.Current.style.display, 'none');
  assert.equal(fields.Old.disabled, false);
  assert.equal(fields.Current.disabled, true);
  assert.deepEqual(submitted(), ['old value']);
});
