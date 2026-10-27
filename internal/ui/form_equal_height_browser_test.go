package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	orun "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Load real YAML and exercise GET, tab clicks and a DSL form event with the
// production styles/scripts. Geometry is measured without invoking layout JS.
func TestEqualHeightGeometry_Browser(t *testing.T) {
	browser := newMobileLayoutBrowser(t, managedLayoutTestBrowser(t))
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		for _, nested := range []bool{false, true} {
			t.Run(fmt.Sprintf("nested=%t", nested), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "ФормаДокумента.form.yaml")
				if err := os.WriteFile(path, []byte(equalHeightFormYAML(nested)), 0o600); err != nil {
					t.Fatal(err)
				}
				module := `Процедура Переключить()
 Объект.Скрыть = НЕ Объект.Скрыть;
КонецПроцедуры
`
				if err := os.WriteFile(filepath.Join(dir, "ФормаДокумента.form.os"), []byte(module), 0o600); err != nil {
					t.Fatal(err)
				}
				fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Геометрия")
				if err != nil {
					t.Fatal(err)
				}
				ent := &metadata.Entity{Name: "Геометрия", Kind: metadata.KindDocument,
					Fields: []metadata.Field{{Name: "Текст", Type: metadata.FieldTypeString}, {Name: "Скрыть", Type: metadata.FieldTypeBool}},
					Forms:  []*metadata.FormModule{fm}}
				if err := db.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
					t.Fatal(err)
				}
				reg := orun.NewRegistry()
				reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{ent}})
				srv := New(reg, db, interpreter.New(), nil, Config{}, nil)
				router := chi.NewRouter()
				srv.MountStatic(router)
				srv.Mount(router)
				// Only append a measurement probe to the actual HTTP card response.
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet && r.URL.Path == "/ui/document/геометрия/new" {
						rec := httptest.NewRecorder()
						router.ServeHTTP(rec, r)
						for k, vs := range rec.Header() {
							w.Header()[k] = vs
						}
						w.WriteHeader(rec.Code)
						_, _ = w.Write([]byte(strings.Replace(rec.Body.String(), "</body>", equalHeightProbe+"</body>", 1)))
						return
					}
					router.ServeHTTP(w, r)
				}))
				defer func() { srv.CloseEventStreams(); server.Close(); srv.Close() }()
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				data, err := browser.measure(ctx, server.URL+"/ui/document/геометрия/new")
				if err != nil {
					t.Fatalf("browser: %v\n%s", err, browser.diagnostics())
				}
				var got struct {
					Failures []string
					Phases   []struct {
						Name        string
						Height, Top float64
					}
				}
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if len(got.Failures) > 0 {
					t.Errorf("geometry/state regression: %s", data)
				}
				if len(got.Phases) < 5 {
					t.Fatalf("missing geometry phases: %s", data)
				}
				t.Logf("equal_height geometry: %s", data)
			})
		}
	})
}

func equalHeightFormYAML(nested bool) string {
	var b strings.Builder
	b.WriteString(`schema: onebase.form/v1
form:
  name: ФормаДокумента
  kind: object
  entity: Геометрия
elements:
  - kind: СтраницыФормы
    name: Внешние
    equal_height: true
    children:
      - kind: Страница
        name: Короткая
        children:
          - kind: ПолеВвода
            name: Первое
            data_path: Объект.Текст
      - kind: Страница
        name: Длинная
        children:
`)
	indent := "          "
	if nested {
		b.WriteString(`          - kind: СтраницыФормы
            name: Внутренние
            equal_height: true
            children:
              - kind: Страница
                name: ВнутренняяКороткая
                children:
                  - kind: ПолеВвода
                    name: Внутреннее
                    data_path: Объект.Текст
              - kind: Страница
                name: ВнутренняяДлинная
                children:
`)
		indent = "                  "
	}
	for i := range 20 {
		fmt.Fprintf(&b, "%s- kind: ПолеВвода\n%s  name: Поле%d\n%s  data_path: Объект.Текст\n%s  hidden_when: 'Скрыть = Истина'\n", indent, indent, i, indent, indent)
	}
	b.WriteString(`  - kind: ПолеВвода
    name: Следующее
    data_path: Объект.Текст
  - kind: Флажок
    name: Скрыть
    data_path: Объект.Скрыть
  - kind: Кнопка
    name: Переключить
    title: {ru: Переключить}
    events: {Нажатие: Переключить}
`)
	return b.String()
}

const equalHeightProbe = `<script>
window.addEventListener('load', async function () {
  var sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
  var publish = (id, value) => { var p=document.createElement('pre'); p.id=id; p.textContent=value; document.body.appendChild(p); };
  try {
    await document.fonts.ready;
    await sleep(350);
    var outer=document.querySelector('[data-tabs="Внешние"]'), inner=document.querySelector('[data-tabs="Внутренние"]');
    var failures=[], phases=[];
    var own=(tabs, selector) => Array.from(tabs.querySelectorAll(selector)).filter(x => x.closest('.managed-tabs')===tabs);
    var measure=name => { var r=outer.getBoundingClientRect(); var x={name:name,height:r.height,top:document.querySelector('[data-ob-el="Следующее"]').getBoundingClientRect().top}; phases.push(x); return x; };
    var stable=(a,b) => { if(Math.abs(a.height-b.height)>1 || Math.abs(a.top-b.top)>1) failures.push(a.name+' -> '+b.name+' moved'); };
    var click=async(tabs,i) => { own(tabs,'.managed-tab-btn')[i].click(); await sleep(200); };
    var initial=measure('initial-hidden-long');
    await click(outer,1); stable(initial,measure('outer-long'));
    if(inner) { await click(inner,1); stable(initial,measure('inner-long')); await click(inner,0); stable(initial,measure('inner-short')); }
    await click(outer,0); stable(initial,measure('outer-short'));
    await click(outer,1); if(inner) await click(inner,1);
    var selection=()=>Array.from(document.querySelectorAll('.managed-tab-btn.active')).map(x=>x.getAttribute('data-tab-page')).join(',');
    var storage=()=>JSON.stringify(Object.entries(sessionStorage).sort());
    var selected=selection(), saved=storage();
    var toggle=async(hidden) => {
      document.querySelector('[data-ob-el="Переключить"]').click();
      var deadline=Date.now()+5000, field=document.querySelector('[data-ob-el="Поле0"]');
      while((getComputedStyle(field).display==='none')!==hidden) { if(Date.now()>deadline)throw new Error('form event did not update visibility'); await sleep(20); }
      await sleep(250);
      if(selection()!==selected || storage()!==saved)failures.push('measurement changed tab selection/storage');
      own(outer,'.managed-tab-content').forEach((x,i)=>{ if((x.style.display==='none')!==(i===0) || x.style.visibility)failures.push('outer display/visibility not restored'); });
      if(inner)own(inner,'.managed-tab-content').forEach((x,i)=>{ if((x.style.display==='none')!==(i===0) || x.style.visibility)failures.push('inner display/visibility not restored'); });
    };
    await toggle(true); var hidden=measure('event-hidden');
    if(hidden.height>=initial.height-100)failures.push('hidden fields still reserve long page');
    if(Array.from(document.querySelectorAll('[data-ob-el^="Поле"]')).some(x=>getComputedStyle(x).display!=='none'))failures.push('measurement revealed rule-hidden field');
    await toggle(false); stable(initial,measure('event-revealed'));
    publish('list-mobile-measure',JSON.stringify({failures:failures,phases:phases}));
  } catch(error) { publish('list-mobile-error',String(error)); }
});
</script>`
