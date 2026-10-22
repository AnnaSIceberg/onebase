'use strict';

// obApplyConcealed: раскрытое поле включается, нераскрытое — отключено и пусто.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync('static/managed.js', 'utf8');
const start = source.indexOf('function obApplyConcealed(');
const end = source.indexOf('window.obApplyConcealed = obApplyConcealed;', start);
assert.ok(start >= 0 && end > start, 'managed.js: obApplyConcealed не найдена');

test('раскрытое включается, нераскрытое отключается и стирается', () => {
  const field = (name, value) => ({ value, disabled: true, getAttribute: (k) => (k === 'name' ? name : null) });
  const opis = field('Описание', 'полный текст');
  const other = field('Адрес', 'ул. Тайная');
  other.disabled = false;
  const ctx = { document: { querySelectorAll: () => [opis, other] } };
  vm.runInNewContext(source.slice(start, end) + '\nthis.obApplyConcealed = obApplyConcealed;', ctx);
  ctx.obApplyConcealed(['описание'], ctx.document);
  assert.equal(opis.disabled, false);
  assert.equal(opis.value, 'полный текст');
  assert.equal(other.disabled, true);
  assert.equal(other.value, '');
});

test('раскрытие ограничено своим экземпляром и сохраняет readonly', () => {
 const own={value:'text',disabled:true,readOnly:true,getAttribute:()=> 'Описание'};
 const alien={value:'hidden',disabled:true};
 const form={querySelectorAll:()=>[own]};
 const ctx={document:{querySelectorAll:()=>{throw Error('global DOM access');}}};
 vm.runInNewContext(source.slice(start,end)+'\nthis.apply=obApplyConcealed;',ctx);
 ctx.apply(['описание'],form);
 assert.equal(own.readOnly,true);
 assert.equal(alien.disabled,true);
 ctx.apply([],form);
 assert.equal(own.value,'');
});
