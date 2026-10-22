package ui

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	orun "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// placement: header / header_end у Кнопка, ПолеВвода и Флажок: header — в
// строке команд перед «? Справка», header_end — после неё. Поле рисуется
// «Подпись: значение» только для чтения, сам контрол остаётся в теле формы
// невидимым — с ним работают запись, события и hidden_when. Форма — из YAML
// штатным маршрутом, поведение — в браузере с managed.js.
func TestManagedHeaderFields_Browser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "hdr.db"))
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("Ошибка выгрузки в 1С: нет связи с сервером. ", 6)
	src := "schema: onebase.form/v1\nform:\n  name: ФормаОбъекта\n  kind: object\n  entity: Задача\nelements:\n" +
		"  - kind: Кнопка\n    name: КнИстория1С\n    placement: header\n    title:\n      ru: Журнал\n" +
		"  - kind: ГруппаФормы\n    name: ГруппаОбменС1С\n    children:\n" +
		"      - kind: ПолеВвода\n        name: ПолеНомерВ1С\n        data_path: Объект.НомерВ1С\n        readonly: true\n        placement: header_end\n        title:\n          ru: Номер в 1С\n" +
		"      - kind: ПолеВвода\n        name: ПолеОшибка\n        data_path: Объект.Ошибка\n        readonly: true\n        placement: header_end\n        title:\n          ru: Не выгружена в 1С\n" +
		"      - kind: Флажок\n        name: ПолеВОТК\n        data_path: Объект.ВОТК\n        placement: header_end\n        title:\n          ru: Передать в ОТК\n" +
		"      - kind: Кнопка\n        name: КнВыгрузить\n        placement: header_end\n        title:\n          ru: Выгрузить в 1С\n" +
		"  - kind: ПолеВвода\n    name: ПолеТема\n    data_path: Объект.Тема\n"
	path := filepath.Join(t.TempDir(), "ФормаОбъекта.form.yaml")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Задача")
	if err != nil {
		t.Fatal(err)
	}
	ent := &metadata.Entity{Name: "Задача", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "НомерВ1С", Type: metadata.FieldTypeString}, {Name: "Ошибка", Type: metadata.FieldTypeString},
		{Name: "ВОТК", Type: metadata.FieldTypeBool}, {Name: "Тема", Type: metadata.FieldTypeString},
	}, Forms: []*metadata.FormModule{fm}}
	if err := db.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	reg := orun.NewRegistry()
	reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{ent}})
	srv := New(reg, db, interpreter.New(), auth.NewRepo(db), Config{}, nil)
	defer func() { srv.BeginShutdown(); db.Close() }()
	router := chi.NewRouter()
	srv.MountStatic(router)
	srv.Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/catalog/задача/new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("form: %d", rec.Code)
	}
	page := rec.Body.String()
	// Значения задаём в теле формы, как их принёс бы сервер, и прогоняем
	// штатные applyValues / applyElementStates.
	script := `<script>setTimeout(function(){
var q=function(s){return document.querySelector(s)};
var mirror=function(n){return q('.ob-header-field[data-ob-header-field="'+n+'"]')};
var f=q('#main-form');
var setv=function(n,v){var i=f.querySelector('[name="'+n+'"]'); if(i.type==='checkbox'){i.checked=v;}else{i.value=v;}};
setv('НомерВ1С','ЗД-000123'); setv('Ошибка',` + "`" + long + "`" + `); setv('ВОТК',true);
window.obRefreshHeaderFields();
var m1=mirror('ПолеНомерВ1С'), m2=mirror('ПолеОшибка'), m3=mirror('ПолеВОТК');
var v2=m2.querySelector('.ob-header-field-value');
var groups=[].slice.call(document.querySelectorAll('.ob-header-commands'));
var res={
 number:m1.textContent.replace(/\s+/g,' ').trim(),
 otk:m3.textContent.replace(/\s+/g,' ').trim(),
 errTitle:m2.title,
 errTruncated:v2.scrollWidth>v2.clientWidth,
 errWidth:m2.getBoundingClientRect().width,
 headerBtnFirst:!!groups[0].querySelector('[data-ob-el="КнИстория1С"]'),
 endInSecond:!!(groups[1]&&groups[1].contains(m1)&&groups[1].querySelector('[data-ob-el="КнВыгрузить"]')),
 sourceHidden:getComputedStyle(f.querySelector('[data-ob-el="ПолеНомерВ1С"]')).display==='none',
 bodyButtons:f.querySelectorAll('[data-ob-el="КнВыгрузить"],[data-ob-el="КнИстория1С"]').length,
 themeVisible:getComputedStyle(f.querySelector('[data-ob-el="ПолеТема"]')).display!=='none'
};
var srcPath=f.querySelector('[data-ob-el="ПолеОшибка"]').getAttribute('data-ob-el-path');
var hidden={}; hidden[srcPath]=true;
window.applyElementStates({hidden:hidden});
res.hiddenAfterState=getComputedStyle(m2).display==='none';
var p=document.createElement('pre');p.id='hdr-measure';p.textContent=JSON.stringify(res);document.body.appendChild(p);
},1200);</script>`
	end := strings.LastIndex(page, "</body>")
	page = page[:end] + script + page[end:]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
		case strings.HasPrefix(r.URL.Path, "/static/"), strings.HasPrefix(r.URL.Path, "/vendor/"):
			router.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	args := []string{"--headless=new", "--disable-gpu", "--disable-background-networking", "--disable-extensions", "--no-first-run",
		"--window-size=1440,784", "--virtual-time-budget=5000", "--user-data-dir=" + t.TempDir(), "--dump-dom", server.URL}
	if runtime.GOOS == "linux" {
		args = append([]string{"--no-sandbox"}, args...)
	}
	cmd := exec.CommandContext(t.Context(), browser, args...) //nolint:gosec // browser selected from existing test allow-list
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	dump, err := cmd.Output()
	if err != nil {
		t.Fatalf("browser: %v: %s", err, stderr.String())
	}
	marker := []byte(`<pre id="hdr-measure">`)
	start := bytes.Index(dump, marker)
	if start < 0 {
		t.Fatalf("browser did not measure: %s", stderr.String())
	}
	start += len(marker)
	stop := bytes.Index(dump[start:], []byte("</pre>"))
	var r struct {
		Number, Otk, ErrTitle                                   string
		ErrTruncated                                            bool
		ErrWidth                                                float64
		HeaderBtnFirst, EndInSecond, SourceHidden, ThemeVisible bool
		BodyButtons                                             int
		HiddenAfterState                                        bool
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(string(dump[start:start+stop]))), &r); err != nil {
		t.Fatal(err)
	}
	if r.Number != "Номер в 1С: ЗД-000123" {
		t.Errorf("поле в строке команд: %q", r.Number)
	}
	if r.Otk != "Передать в ОТК: ✓" {
		t.Errorf("флажок в строке команд: %q", r.Otk)
	}
	if !r.ErrTruncated || r.ErrWidth > 421 || !strings.Contains(r.ErrTitle, strings.TrimSpace(long)) {
		t.Errorf("длинный текст ошибки: обрезан=%v ширина=%.0f подсказка=%q", r.ErrTruncated, r.ErrWidth, r.ErrTitle)
	}
	if !r.HeaderBtnFirst || !r.EndInSecond {
		t.Errorf("порядок строки команд: header перед «? Справка»=%v, header_end после=%v", r.HeaderBtnFirst, r.EndInSecond)
	}
	if !r.SourceHidden || r.BodyButtons != 0 || !r.ThemeVisible {
		t.Errorf("тело формы: источник скрыт=%v, кнопок шапки в теле=%d, обычное поле видно=%v", r.SourceHidden, r.BodyButtons, r.ThemeVisible)
	}
	if !r.HiddenAfterState {
		t.Error("hidden_when (applyElementStates) не скрыл поле в строке команд")
	}
}
