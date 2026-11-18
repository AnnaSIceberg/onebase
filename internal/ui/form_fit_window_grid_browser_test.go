package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	orun "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// fit_window у ТабличнаяЧасть (сетка SlickGrid): таблица результатов
// занимает остаток высоты окна — форма не прокручивается, строки
// прокручиваются внутри сетки, вьюпорт SlickGrid пересчитан под новую высоту.
// Раньше ключ на ТЧ молча игнорировался: сетка 300px, под формой пусто
// (поиск заявок колл-центра). Подгонка трогает сетку, только когда высота
// меняется, — иначе перерисовка строк будила бы наблюдатель подгонки снова:
// после того как всё улеглось, высота сетки больше не меняется.
func TestFitWindowTablePartGrid_Browser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	type result struct {
		Grid      string  `json:"grid"`
		GridH     float64 `json:"gridH"`
		Viewport  float64 `json:"viewport"`
		Scroll    float64 `json:"scroll"`
		Client    float64 `json:"client"`
		LateMoves int     `json:"lateMoves"`
	}
	db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "grid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	src := "schema: onebase.form/v1\nform:\n  name: ФормаДокумента\n  kind: object\n  entity: Поиск\n" +
		"actions:\n  attachments: { visible: false }\nelements:\n" +
		"  - kind: ПолеВвода\n    name: ПолеСтрока\n    data_path: Объект.Строка\n" +
		"  - kind: ТабличнаяЧасть\n    name: ТЧНайденные\n    data_path: Объект.Найденные\n    fit_window: true\n"
	path := filepath.Join(t.TempDir(), "ФормаДокумента.form.yaml")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Поиск")
	if err != nil {
		t.Fatal(err)
	}
	tp := metadata.TablePart{Name: "Найденные", Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}}}
	ent := &metadata.Entity{Name: "Поиск", Kind: metadata.KindCatalog,
		Fields:     []metadata.Field{{Name: "Строка", Type: metadata.FieldTypeString}},
		TableParts: []metadata.TablePart{tp}, Forms: []*metadata.FormModule{fm}}
	if err := db.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	if err := db.Upsert(t.Context(), ent.Name, id, map[string]any{"Строка": "заявки"}, ent); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	for i := 1; i <= 30; i++ {
		rows = append(rows, map[string]any{"Номер": fmt.Sprintf("Пл%07d", i)})
	}
	if err := db.UpsertTablePartRows(t.Context(), ent.Name, tp.Name, id, rows, tp); err != nil {
		t.Fatal(err)
	}
	reg := orun.NewRegistry()
	reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{ent}})
	srv := New(reg, db, interpreter.New(), auth.NewRepo(db), Config{}, nil)
	defer srv.BeginShutdown()
	router := chi.NewRouter()
	srv.MountStatic(router)
	srv.Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/catalog/поиск/"+id.String(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("form: %d", rec.Code)
	}
	page := rec.Body.String()
	if !strings.Contains(page, `data-ob-fit-grid="1"`) {
		t.Fatal("у сетки ТЧ с fit_window нет признака data-ob-fit-grid")
	}
	// Высоты сетки после того, как всё улеглось (2,5 с), не должны меняться:
	// изменения за следующие 1,5 с считаются.
	script := `<script>var moves=0,settled=false;setTimeout(function(){settled=true;var g=document.querySelector('[data-ob-fit-grid="1"]');new MutationObserver(function(){moves++;}).observe(g,{attributes:true,attributeFilter:['style']});},2500);` +
		`setTimeout(function(){var g=document.querySelector('[data-ob-fit-grid="1"]');var m=document.querySelector('main');var v=g.querySelector('.slick-viewport');` +
		`var p=document.createElement('pre');p.id='fit-measure';p.textContent=JSON.stringify({grid:g.style.height,gridH:g.offsetHeight,viewport:v?v.offsetHeight:0,scroll:m.scrollHeight,client:m.clientHeight,lateMoves:moves});document.body.appendChild(p);},4000);</script>`
	end := strings.LastIndex(page, "</body>")
	page = page[:end] + script + page[end:]
	// Статика — настоящим роутером; остальное (SSE, события формы) не
	// нужно и не держит страницу открытой.
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
		"--window-size=1440,1000", "--virtual-time-budget=8000", "--user-data-dir=" + t.TempDir(), "--dump-dom", server.URL}
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
	marker := []byte(`<pre id="fit-measure">`)
	start := bytes.Index(dump, marker)
	if start < 0 {
		t.Fatalf("browser did not measure: %s", stderr.String())
	}
	start += len(marker)
	stop := bytes.Index(dump[start:], []byte("</pre>"))
	var res result
	if err := json.Unmarshal([]byte(html.UnescapeString(string(dump[start:start+stop]))), &res); err != nil {
		t.Fatal(err)
	}
	if res.Grid == "" || res.GridH <= 300 {
		t.Fatalf("сетка не заняла остаток окна (по умолчанию 300px): %+v", res)
	}
	if res.Scroll > res.Client {
		t.Errorf("форма выше окна на %.0f px: %+v", res.Scroll-res.Client, res)
	}
	if res.Viewport <= 0 || res.Viewport > res.GridH || res.GridH-res.Viewport > 80 {
		t.Errorf("вьюпорт SlickGrid не пересчитан под высоту сетки: %+v", res)
	}
	if res.LateMoves != 0 {
		t.Errorf("подгонка сетки не успокоилась: %d изменений высоты после 2,5 с", res.LateMoves)
	}
}
