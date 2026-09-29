// Скрытые поля реквизитов формы без своего контрола (managed.js).
//
// Сервер реквизиты формы между событиями не хранит: событие видит только то,
// что прислал браузер. Реквизит без контрола (флаг для hidden_when) возвращался
// пустым, и условие откатывалось при первом постороннем событии. Скрытое поле
// возит значение туда и обратно; реквизиту со своим контролом оно не нужно —
// одноимённая пара дала бы серверу два значения.

const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync('static/managed.js', 'utf8');
const start = source.indexOf('// BEGIN onebase-form-attr-carriers');
const end = source.indexOf('// END onebase-form-attr-carriers', start);
if (start < 0 || end < 0) throw new Error('form attribute carrier block is absent from managed.js');
const block = source.slice(start, end);

function fakeForm(controls) {
  const appended = [];
  return {
    appended,
    elements: {namedItem: (name) => (controls.includes(name) || appended.some((el) => el.name === name) ? {name} : null)},
    appendChild(el) { appended.push(el); },
  };
}

function carriers(form, names) {
  const document = {
    createElement(tag) {
      const attrs = {};
      return {tag, attrs, setAttribute(k, v) { attrs[k] = v; }};
    },
  };
  const sandbox = {document, String};
  vm.createContext(sandbox);
  vm.runInContext(block, sandbox, {filename: 'managed.js#form-attr-carriers'});
  return sandbox.obEnsureFormAttrCarriers(form, names);
}

test('скрытое поле — только реквизиту без своего контрола', () => {
  const form = fakeForm(['Склад']);
  const created = carriers(form, ['Склад', 'СкрытьПоле', 'ПользовательАдмин']);
  assert.deepEqual(Array.from(created), ['СкрытьПоле', 'ПользовательАдмин']);
  assert.equal(form.appended.length, 2);
  for (const el of form.appended) {
    assert.equal(el.tag, 'input');
    assert.equal(el.type, 'hidden');
    assert.equal(el.attrs['data-ob-form-attr-carrier'], '1');
  }
});

test('повторный вызов не плодит дубликаты', () => {
  const form = fakeForm([]);
  carriers(form, ['СкрытьПоле']);
  const again = carriers(form, ['СкрытьПоле']);
  assert.equal(again.length, 0);
  assert.equal(form.appended.length, 1);
});

test('без формы или без реквизитов — ничего', () => {
  assert.equal(carriers(null, ['СкрытьПоле']).length, 0);
  assert.equal(carriers(fakeForm([]), []).length, 0);
});

test('поля заводятся до восстановления реквизитов и до автособытия ПриОткрытии', () => {
  const call = source.indexOf("obEnsureFormAttrCarriers(document.getElementById('main-form'), FORM_ATTRS)");
  assert.ok(call > 0, 'вызов obEnsureFormAttrCarriers не найден');
  const restore = source.indexOf('restoreFormAttrs();', call);
  assert.ok(restore > call, 'restoreFormAttrs должен идти после заведения полей');
  // Автособытие ПриОткрытии отложено setTimeout в своём obManagedReady ниже
  // по файлу: к его запуску поля уже есть.
  const autoOpen = source.indexOf("obFire('', 'ПриОткрытии')");
  assert.ok(autoOpen > restore, 'автособытие ПриОткрытии должно регистрироваться позже');
});
