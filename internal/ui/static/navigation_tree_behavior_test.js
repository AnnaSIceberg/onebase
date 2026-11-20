const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync(path.join(__dirname, 'navigation-tree.js'), 'utf8');

function openFolders(query, formVariant) {
  const saved = new Map([['full', '0'], ['total', '0'], ['base', '0']]);
  const folders = [...saved.keys()].map(key => {
    const summary = {setAttribute() {}, addEventListener() {}};
    return {dataset: {navKey: key}, open: true, parentElement: null,
      querySelector: () => summary, addEventListener() {}, matches: () => true};
  });
  const links = folders.map((folder, index) => ({parentElement: folder,
    href: 'http://menu.test/ui/report/work?subsystem=OTK' + (index < 2 ? '&__variant=' + encodeURIComponent(index ? 'Итоговый' : 'Полный') : '')}));
  const root = {dataset: {}, querySelectorAll: selector => selector === 'details.navtree' ? folders : links, addEventListener() {}};
  const document = {readyState: 'complete', documentElement: {},
    getElementById: () => root, querySelector: () => formVariant === undefined ? null : {value: formVariant},
    createElement: () => ({}), head: {appendChild() {}}};
  vm.runInNewContext(source, {window: {}, document, location: {href: 'http://menu.test/ui/report/work?subsystem=OTK' + query}, URL,
    localStorage: {getItem: key => saved.get(key), setItem: (key, value) => saved.set(key, value)},
    MutationObserver: class {observe() {}}, setTimeout});
  assert.deepEqual([...saved.values()], ['0', '0', '0'], 'opening active ancestors must not overwrite saved choices');
  return folders.filter(folder => folder.open).map(folder => folder.dataset.navKey);
}

test('menu report variant expands only its own branch', () => {
  assert.deepEqual(openFolders('&__variant=' + encodeURIComponent('Полный')), ['full']);
});
test('POST result uses the rendered variant when the URL has no query selection', () => {
  assert.deepEqual(openFolders('', 'Итоговый'), ['total']);
});
test('explicit base selection keeps other variants collapsed', () => {
  assert.deepEqual(openFolders('&__variant=', 'Итоговый'), ['base']);
});

test('typed menu reopens manual choices only in the current boot and login', () => {
  const local = new Map([['section', '1'], ['child', '1']]);
  const session = new Map();
  const store = map => ({getItem: key => map.get(key), setItem: (key, value) => map.set(key, value)});
  const init = scope => {
    const folders = ['section', 'child'].map(key => {
      const handlers = {};
      const element = {
        dataset: {navKey: key}, open: !scope, matches: () => true, addEventListener() {},
        querySelector: () => ({setAttribute() {}, addEventListener: (kind, fn) => handlers[kind] = fn}),
        toggle: () => handlers.keydown({key: element.open ? 'ArrowLeft' : 'ArrowRight', preventDefault() {}}),
      };
      return element;
    });
    const root = {dataset: {navSession: scope}, addEventListener() {},
      querySelectorAll: selector => selector === 'details.navtree' ? folders : []};
    const document = {readyState: 'complete', documentElement: {}, getElementById: () => root,
      querySelector: () => null, createElement: () => ({}), head: {appendChild() {}}};
    vm.runInNewContext(source, {window: {}, document, location: {href: 'http://menu.test/ui/app'}, URL,
      localStorage: store(local), sessionStorage: store(session), MutationObserver: class {observe() {}}, setTimeout});
    return folders;
  };
  const states = folders => folders.map(folder => folder.open);
  const first = init('boot-a:login-a');
  assert.deepEqual(states(first), [false, false]);
  first.forEach(folder => folder.toggle());
  assert.deepEqual(states(init('boot-a:login-a')), [true, true]);
  assert.deepEqual(states(init('boot-a:login-b')), [false, false]);
  assert.deepEqual(states(init('boot-b:login-a')), [false, false]);
  assert.deepEqual(states(init('')), [true, true]);
});
