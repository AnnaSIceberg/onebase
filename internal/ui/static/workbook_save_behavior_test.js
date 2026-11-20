const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, 'tabular.js'), 'utf8');
const xlsx = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

function fixture(options = {}) {
  const messages = [];
  const stored = options.stored || new Map();
  const picker = {dataset: {obWorkbookPicker: 'xlsx-fixture'}};
  const name = {value: 'Default.xlsx'};
  let chooseClick;
  const choose = {addEventListener: (_type, listener) => chooseClick = listener};
  const status = {hidden: true, querySelector: () => percent};
  const percent = {textContent: ''};
  const button = {
    getAttribute: key => key === 'data-ob-fire-click' ? 'Build' : null,
    hasAttribute: key => key === 'data-ob-workbook-save',
    nextElementSibling: status,
  };
  picker.querySelector = key => key === '[data-ob-workbook-name]' ? name : choose;
  const form = {querySelector: () => picker};
  const window = {};
  if (options.choose) window.showSaveFilePicker = options.choose;
  vm.runInNewContext(source, {
    window, document: {querySelectorAll: () => [button]}, fetch: options.fetch,
    localStorage: {getItem: key => stored.get(key), setItem: (key, value) => stored.set(key, value)},
  });
  const flash = (message, type) => messages.push([message, type]);
  window.obBindWorkbookPicker(form, flash);
  return {window, messages, name, status, percent, choose: () => chooseClick(),
    build: run => window.obRunWorkbookSave(form, 'Build', flash, run)};
}

test('selects once, remembers the filename and reuses the handle', async () => {
  const stored = new Map(), calls = [];
  const handle = {name: 'Chosen.xlsx', requestPermission: async () => 'granted'};
  const f = fixture({stored, choose: async options => { calls.push(options); return handle; }});
  await f.choose();
  await f.build(destination => assert.equal(destination.handle, handle));
  assert.equal(calls.length, 1);
  assert.equal(calls[0].id, 'xlsx-fixture');
  assert.equal(calls[0].suggestedName, 'Default.xlsx');
  assert.equal(fixture({stored}).name.value, 'Chosen.xlsx');
});

test('cancel and denied permission stop generation; unsupported browser keeps download', async () => {
  const cancelled = fixture({choose: async () => {throw Object.assign(new Error('cancel'), {name:'AbortError'});}});
  await cancelled.build(() => assert.fail('generated after cancel'));
  assert.deepEqual(cancelled.messages, []);
  const denied = fixture({choose: async () => ({name:'Denied.xlsx',requestPermission:async () => 'denied'})});
  await denied.build(() => assert.fail('generated with denied permission'));
  assert.equal(denied.messages[0][1], 'err');
  await fixture().build(destination => assert.equal(destination.handle, null));
});

test('workbook progress renders without an iframe and finishes', () => {
  const f = fixture();
  const progress = f.window.obBeginTabularProgress('Build');
  assert.equal(f.status.hidden, false);
  assert.equal(f.percent.textContent, '0%');
  progress.update(50);
  assert.equal(f.percent.textContent, '50%');
  progress.finish();
  assert.equal(f.status.hidden, true);
});

test('write waits for close and aborts on failed write', async () => {
  const order = [];
  const f = fixture({fetch: async () => ({ok: true, headers: {get: () => xlsx}, blob: async () => 'xlsx'})});
  const writer = {
    write: async data => order.push(data),
    close: async () => order.push('close'),
    abort: async () => order.push('abort'),
  };
  const handle = {createWritable: async () => writer};
  await f.window.obWriteWorkbook(handle, '/ui/tabular/id/xlsx');
  assert.deepEqual(order, ['xlsx', 'close']);
  writer.write = async () => {throw new Error('disk full');};
  await assert.rejects(f.window.obWriteWorkbook(handle, '/ui/tabular/id/xlsx'), /disk full/);
  assert.deepEqual(order, ['xlsx', 'close', 'abort']);
});
