const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

// Порядок колонок таблицы формы за пользователем: разбор сохранённого
// порядка, применение к колонкам формы и перенос колонки перетаскиванием.
const source = fs.readFileSync('static/managed.js', 'utf8');
const start = source.indexOf('function obGridParseOrder');
const end = source.indexOf('\n(function(){', start);
if (start < 0 || end < 0) throw new Error('grid column order helpers not found');
const helpers = new Function(source.slice(start, end) +
  '\nreturn {obGridParseOrder, obGridApplyOrder, obGridMoveColumn, obGridColumnIds};')();

const cols = () => [{id: 'Номер'}, {id: 'Адрес'}, {id: 'Мастер'}];

test('сохранённый порядок применяется на месте массива', () => {
  const columns = cols();
  const same = columns;
  assert.equal(helpers.obGridApplyOrder(columns, ['Мастер', 'Номер', 'Адрес']), true);
  assert.equal(columns, same, 'массив тот же: обработчики берут columns[args.cell]');
  assert.deepEqual(helpers.obGridColumnIds(columns), ['Мастер', 'Номер', 'Адрес']);
});

test('новые колонки формы — после переставленных, исчезнувшие — пропускаются', () => {
  const columns = cols();
  assert.equal(helpers.obGridApplyOrder(columns, ['Мастер', 'Удалённая']), true);
  assert.deepEqual(helpers.obGridColumnIds(columns), ['Мастер', 'Номер', 'Адрес']);
});

test('порядок как в форме — без изменений', () => {
  const columns = cols();
  assert.equal(helpers.obGridApplyOrder(columns, ['Номер', 'Адрес', 'Мастер']), false);
  assert.equal(helpers.obGridApplyOrder(columns, []), false);
});

test('разбор data-sg-order: мусор не ломает таблицу', () => {
  assert.deepEqual(helpers.obGridParseOrder('["Адрес","",5,"Номер"]'), ['Адрес', 'Номер']);
  assert.deepEqual(helpers.obGridParseOrder('{"a":1}'), []);
  assert.deepEqual(helpers.obGridParseOrder('не json'), []);
  assert.deepEqual(helpers.obGridParseOrder(null), []);
});

test('перетаскивание переносит колонку на место целевой', () => {
  const columns = cols();
  assert.equal(helpers.obGridMoveColumn(columns, 2, 0), true);
  assert.deepEqual(helpers.obGridColumnIds(columns), ['Мастер', 'Номер', 'Адрес']);
  assert.equal(helpers.obGridMoveColumn(columns, 0, 2), true);
  assert.deepEqual(helpers.obGridColumnIds(columns), ['Номер', 'Адрес', 'Мастер']);
  assert.equal(helpers.obGridMoveColumn(columns, 1, 1), false);
  assert.equal(helpers.obGridMoveColumn(columns, 0, 9), false);
});
