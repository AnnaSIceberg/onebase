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

// fit_window в настоящем браузере с managed.js: страницы получают остаток окна
// и форма не прокручивается — даже когда часть высоты main не видна по
// прямоугольникам его блоков (здесь — псевдоэлемент main::after 6px, как
// отступ, которого подгонка не видит). Без сверки со scrollHeight форма
// оказывалась выше окна ровно на эти пиксели.
func TestFitWindowNoFormScroll_Browser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	type result struct {
		After     string  `json:"after"`
		Disp      string  `json:"disp"`
		Ov        string  `json:"ov"`
		MB        float64 `json:"mb"`
		CB        float64 `json:"cb"`
		WH        float64 `json:"wh"`
		Tab       string  `json:"tab"`
		Scroll    float64 `json:"scroll"`
		Client    float64 `json:"client"`
		TabHeight float64 `json:"tabHeight"`
	}
	measure := func(extraCSS string, late bool) result {
		t.Helper()
		db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "fit.db"))
		if err != nil {
			t.Fatal(err)
		}
		src := "schema: onebase.form/v1\nform:\n  name: ФормаДокумента\n  kind: object\n  entity: Заявка\n" +
			"elements:\n  - kind: СтраницыФормы\n    name: Закладки\n    fit_window: true\n    children:\n" +
			"      - kind: Страница\n        name: Заказчик\n        children:\n" +
			"          - kind: ПолеВвода\n            name: ПолеКлиент\n            data_path: Объект.Клиент\n"
		path := filepath.Join(t.TempDir(), "ФормаДокумента.form.yaml")
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Заявка")
		if err != nil {
			t.Fatal(err)
		}
		ent := &metadata.Entity{Name: "Заявка", Kind: metadata.KindDocument, Posting: true,
			Fields: []metadata.Field{{Name: "Клиент", Type: metadata.FieldTypeString}}, Forms: []*metadata.FormModule{fm}}
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
		// Замер после подгонки (obManagedReady) и её отложенного повтора.
		script := `<style>` + extraCSS + `</style><script>setTimeout(function(){if(window.__late){var d=document.createElement('div');d.style.height='60px';d.textContent='вторая строка шапки';var cd=document.querySelector('main>.card');cd.insertBefore(d,cd.firstChild);}},300);var t0=Date.now();var poll=function(){var m=document.querySelector('main');if(m.scrollHeight>m.clientHeight&&Date.now()-t0<5000){setTimeout(poll,200);return;}var c=document.querySelector('.managed-tab-content');` +
			`var p=document.createElement('pre');p.id='fit-measure';p.textContent=JSON.stringify({tab:c.style.height,scroll:m.scrollHeight,client:m.clientHeight,tabHeight:c.offsetHeight,after:getComputedStyle(m,'::after').height,disp:getComputedStyle(m).display,ov:getComputedStyle(m).overflowY,mb:Math.round(m.getBoundingClientRect().bottom),cb:Math.round(document.querySelector('main>.card').getBoundingClientRect().bottom),wh:innerHeight});document.body.appendChild(p);};setTimeout(poll,1000);</script>`
		end := strings.LastIndex(page, "</body>")
		if late {
			script = `<script>window.__late=true</script>` + script
		}
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
			"--window-size=1440,712", "--virtual-time-budget=8000", "--user-data-dir=" + t.TempDir(), "--dump-dom", server.URL}
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

	plain := measure("", false)
	if plain.Tab == "" || plain.Scroll > plain.Client {
		t.Fatalf("fit_window не подогнал страницы под окно: %+v", plain)
	}
	// Высота, которой подгонка не видит по прямоугольникам блоков main, —
	// больше естественного зазора: без сверки со scrollHeight форма
	// прокручивалась бы.
	hidden := measure("main::after{content:'';display:block;height:60px}", false)
	if hidden.Scroll > hidden.Client {
		t.Errorf("невидимая подгонке высота: форма выше окна на %.0f px: %+v", hidden.Scroll-hidden.Client, hidden)
	}
	// Содержимое, появившееся после открытия формы (вторая строка шапки,
	// отложенная группа): подгонка пересчитывается без resize окна.
	late := measure("", true)
	if late.Scroll > late.Client {
		t.Errorf("позднее содержимое: форма выше окна на %.0f px: %+v", late.Scroll-late.Client, late)
	}
	if d := plain.TabHeight - late.TabHeight; d < 50 {
		t.Errorf("страницы не уступили место позднему блоку 60px: ужались на %.0f px", d)
	}
}
