package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ui.открытьФорму с отбором и параметрами: список открывается с отбором
// (f.<Поле>=…), отчёт — с параметрами и сразу сформированным (__run=1). Раньше
// formURL строил только путь, и кнопка формы не могла открыть «Жалобы на
// мастере» или «Штрафы на мастере» с отбором, как в 1С.
func TestFormURLCarriesFilterAndReportParams(t *testing.T) {
	runFormURLNode(t, `
const assert = require('node:assert');
assert.strictEqual(formURL({'вид': 'document', 'сущность': 'А_Жалоба'}), '/ui/document/' + encodeURIComponent('а_жалоба'));
assert.strictEqual(formURL({'вид': 'document', 'сущность': 'А_Жалоба', 'отбор': {'Мастер': 'm-1'}}),
  '/ui/document/' + encodeURIComponent('а_жалоба') + '?' + encodeURIComponent('f.Мастер') + '=m-1');
assert.strictEqual(formURL({'вид': 'document', 'сущность': 'А_Жалоба', 'id': 'x', 'отбор': {'Мастер': 'm-1'}}),
  '/ui/document/' + encodeURIComponent('а_жалоба') + '/x', 'у формы объекта отбора нет');
assert.strictEqual(formURL({'вид': 'report', 'сущность': 'Штрафы', 'параметры': {'Мастер': 'm-1', 'Пусто': null}, 'сформировать': true}),
  '/ui/report/' + encodeURIComponent('Штрафы') + '?' + encodeURIComponent('Мастер') + '=m-1&__run=1');
assert.strictEqual(formURL({'вид': 'report', 'сущность': 'Штрафы'}), '/ui/report/' + encodeURIComponent('Штрафы'));
assert.strictEqual(formURL({'вид': 'processor', 'сущность': 'Обр', 'параметры': {'А': 1, 'Б': false}, 'сформировать': true}),
  '/ui/processor/' + encodeURIComponent('Обр') + '?' + encodeURIComponent('А') + '=1&' + encodeURIComponent('Б') + '=false');

const location = {origin: 'http://onebase.local'};
let legacy = false;
const document = {body:{hasAttribute: () => legacy}};
const normal = [];
const result = new Map();
const window = {
  obOpenTab: url => normal.push(url),
  obOpenReportResultFromTab: (source,key,url) => {
    result.set(source + key, new URL(url, location.origin));
    return 'result-frame';
  },
};
function generate(period, slot) {
  openFormTab({'вид':'report', 'сущность':'Штрафы', 'параметры':{'начало':period}, 'сформировать':true, 'ключрезультата':slot}, 'обработка');
}
generate('2026-09-01');
generate('2026-10-01');
assert.strictEqual(normal.length, 2);
assert.notStrictEqual(normal[0], normal[1]);
assert(!normal.some(url => url.includes('__result')));
assert.strictEqual(result.size, 0);
generate('2026-09-01', 'основной');
generate('2026-09-01', 'дополнительный');
generate('2026-10-01', 'основной');
assert.strictEqual(result.size, 2);
assert.strictEqual(result.get('обработкаreport:/ui/report/Штрафы:основной').searchParams.get('начало'), '2026-10-01');
assert.strictEqual(result.get('обработкаreport:/ui/report/Штрафы:дополнительный').searchParams.get('начало'), '2026-09-01');
legacy = true;
generate('2026-11-01', 'основной');
assert.strictEqual(normal.length, 3);
assert(!normal[2].includes('__result'));
console.log('ok');
`)
}

func runFormURLNode(t *testing.T, script string) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	src := string(uiJS)
	start := strings.Index(src, "  function formURL(link) {")
	if start < 0 {
		t.Fatal("в ui.js нет formURL")
	}
	open := strings.Index(src[start:], "  function openFormTab(")
	if open < 0 {
		t.Fatal("в ui.js нет openFormTab")
	}
	end := strings.Index(src[start+open:], "\n  }\n")
	if end < 0 {
		t.Fatal("не найден конец openFormTab")
	}
	fn := src[start : start+open+end+4]
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "t.js"), []byte(fn+"\n"+script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "t.js") //nolint:gosec // test-only executable resolved by exec.LookPath
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}
