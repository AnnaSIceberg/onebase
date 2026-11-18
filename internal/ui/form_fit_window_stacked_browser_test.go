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

// fit_window у страниц и правой панели (ГруппаФормы) одной строки, когда
// окно уже, чем они вместе: строка переносит панель ПОД страницы. Подгонка
// «рядом» в такой раскладке ужимала и страницы, и панель до 120px — обе
// обрезаны, страница в 400px прокручивалась в окне 120px (заявка
// колл-центра при развёрнутом левом меню или окне уже ~1410px). Перенесённую
// раскладку подгонка не трогает: форма естественной высоты. В широком окне —
// прежняя подгонка: страницы — остаток окна, панель — их высоты.
func TestFitWindowStackedGroupNotSqueezed_Browser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	type result struct {
		Tab       string  `json:"tab"`
		Group     string  `json:"group"`
		Stacked   bool    `json:"stacked"`
		TabHeight float64 `json:"tabHeight"`
		TabScroll float64 `json:"tabScroll"`
	}
	var fields strings.Builder
	var entFields []metadata.Field
	for _, n := range []string{"Клиент", "Телефон", "Адрес", "Улица", "Дом", "Квартира", "Подъезд", "Этаж", "Код", "Метро", "Примечание", "Источник"} {
		fields.WriteString("              - kind: ПолеВвода\n                name: Поле" + n + "\n                data_path: Объект." + n + "\n")
		entFields = append(entFields, metadata.Field{Name: n, Type: metadata.FieldTypeString})
	}
	src := "schema: onebase.form/v1\nform:\n  name: ФормаДокумента\n  kind: object\n  entity: Заявка\n" +
		"elements:\n  - kind: ГруппаФормы\n    name: Строка\n    orientation: horizontal\n    children:\n" +
		"      - kind: СтраницыФормы\n        name: Закладки\n        fit_window: true\n        width: 620\n        children:\n" +
		"          - kind: Страница\n            name: Заказчик\n            children:\n" + fields.String() +
		"      - kind: ГруппаФормы\n        name: Панель\n        width: 345\n        fit_window: true\n        children:\n" +
		"          - kind: ПолеВвода\n            name: ПолеРеклама\n            data_path: Объект.Источник\n"
	measure := func(width string) result {
		t.Helper()
		db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "fit.db"))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "ФормаДокумента.form.yaml")
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Заявка")
		if err != nil {
			t.Fatal(err)
		}
		ent := &metadata.Entity{Name: "Заявка", Kind: metadata.KindDocument, Posting: true, Fields: entFields, Forms: []*metadata.FormModule{fm}}
		if err := db.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
			t.Fatal(err)
		}
		reg := orun.NewRegistry()
		reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{ent}})
		srv := New(reg, db, interpreter.New(), auth.NewRepo(db), Config{}, nil)
		router := chi.NewRouter()
		srv.MountStatic(router)
		srv.Mount(router)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/document/заявка/new", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("form: %d", rec.Code)
		}
		page := rec.Body.String()
		// Замер после подгонки (obManagedReady) и её отложенных повторов.
		script := `<script>setTimeout(function(){var c=document.querySelector('.managed-tab-content');var g=document.querySelector('[data-ob-el="Панель"]');var tb=c.closest('.managed-tabs');` +
			`var p=document.createElement('pre');p.id='fit-measure';p.textContent=JSON.stringify({tab:c.style.height,group:g.style.height,stacked:g.getBoundingClientRect().top>=tb.getBoundingClientRect().bottom-1,tabHeight:c.offsetHeight,tabScroll:c.scrollHeight});document.body.appendChild(p);},2500);</script>`
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
			"--window-size=" + width + ",712", "--virtual-time-budget=6000", "--user-data-dir=" + t.TempDir(), "--dump-dom", server.URL}
		if runtime.GOOS == "linux" {
			args = append([]string{"--no-sandbox"}, args...)
		}
		cmd := exec.CommandContext(t.Context(), browser, args...) //nolint:gosec // browser selected from existing test allow-list
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		dump, err := cmd.Output()
		srv.BeginShutdown()
		db.Close()
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
		return res
	}

	wide := measure("1440")
	if wide.Stacked || wide.Tab == "" || wide.Group == "" {
		t.Fatalf("широкое окно: панель рядом и обе части подогнаны, получено %+v", wide)
	}
	narrow := measure("760")
	if !narrow.Stacked {
		t.Fatalf("узкое окно не перенесло панель — тест не проверяет перенос: %+v", narrow)
	}
	if narrow.Tab != "" || narrow.Group != "" || narrow.TabScroll > narrow.TabHeight+1 {
		t.Errorf("перенесённая раскладка ужата подгонкой: страница %q (%.0f из %.0f px), панель %q",
			narrow.Tab, narrow.TabHeight, narrow.TabScroll, narrow.Group)
	}
}
