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

	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
)

// renderHeaderInlineForm открывает форму документа из YAML штатным
// обработчиком; header — строки блока form: (header_inline, density).
func renderHeaderInlineForm(t *testing.T, header string) string {
	t.Helper()
	src := "schema: onebase.form/v1\nform:\n  name: ФормаДокумента\n  kind: object\n  entity: Заявка\n" + header +
		"elements:\n  - kind: ГруппаФормы\n    name: ГруппаШапка\n    title:\n      ru: Заказчик\n    children:\n" +
		"      - kind: ПолеВвода\n        name: ПолеКлиент\n        data_path: Объект.Клиент\n"
	path := filepath.Join(t.TempDir(), "ФормаДокумента.form.yaml")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Заявка")
	if err != nil {
		t.Fatal(err)
	}
	ent := &metadata.Entity{Name: "Заявка", Kind: metadata.KindDocument, Posting: true,
		Fields: []metadata.Field{{Name: "Клиент", Type: metadata.FieldTypeString}},
		Forms:  []*metadata.FormModule{fm}}
	srv, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
	req := reqWithChi(http.MethodGet, "/ui/document/заявка/new", nil, map[string]string{"entity": "заявка"})
	rec := httptest.NewRecorder()
	srv.form(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("form: %d %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

// header_inline ставит класс на <html> только этой форме и вместе с density
// (оба класса в одном атрибуте); без ключа страница прежняя.
func TestManagedFormHeaderInlineClass(t *testing.T) {
	for _, tc := range []struct {
		header, want string
	}{
		{"", `<html lang="ru">`},
		{"  header_inline: true\n", `<html lang="ru" class="ob-header-inline">`},
		{"  header_inline: true\n  density: compact\n", `<html lang="ru" class="ob-density-compact ob-header-inline">`},
	} {
		page := renderHeaderInlineForm(t, tc.header)
		if !strings.Contains(page, tc.want) {
			t.Errorf("form %q: нет %s", tc.header, tc.want)
		}
	}
}

// Геометрия в настоящем браузере: с header_inline заголовок, строка команд и
// «×» стоят в одной строке, содержимое формы — ниже; без ключа строка команд
// под заголовком.
func TestManagedFormHeaderInline_Browser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	type box struct{ Top, Bottom, Left, Right float64 }
	type geometry struct{ Title, Actions, Close, Card box }
	measure := func(page string) geometry {
		t.Helper()
		const script = `<script>
window.addEventListener('load', () => {
 const r = (sel) => { const el = document.querySelector(sel); if (!el) return null; const b = el.getBoundingClientRect(); return {top:b.top,bottom:b.bottom,left:b.left,right:b.right}; };
 const out = document.createElement('pre'); out.id = 'header-inline-measure';
 out.textContent = JSON.stringify({title:r('main h2'),actions:r('main>.ob-form-actions'),close:r('[data-ob-close-tab]'),card:r('main>.card')});
 document.body.appendChild(out);
});
</script>`
		end := strings.LastIndex(page, "</body>")
		if end < 0 {
			t.Fatal("HTML has no body")
		}
		page = page[:end] + script + page[end:]
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
		}))
		defer server.Close()
		args := []string{"--headless=new", "--disable-gpu", "--disable-background-networking", "--disable-extensions", "--no-first-run",
			"--window-size=1440,784", "--user-data-dir=" + t.TempDir(), "--dump-dom", server.URL}
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
		marker := []byte(`<pre id="header-inline-measure">`)
		start := bytes.Index(dump, marker)
		if start < 0 {
			t.Fatalf("browser did not measure: %s", stderr.String())
		}
		start += len(marker)
		stop := bytes.Index(dump[start:], []byte("</pre>"))
		var g geometry
		if err := json.Unmarshal([]byte(html.UnescapeString(string(dump[start:start+stop]))), &g); err != nil {
			t.Fatal(err)
		}
		return g
	}

	inline := measure(renderHeaderInlineForm(t, "  header_inline: true\n"))
	if inline.Actions.Top >= inline.Title.Bottom || inline.Title.Top >= inline.Actions.Bottom {
		t.Errorf("строка команд не в строке заголовка: заголовок %+v, команды %+v", inline.Title, inline.Actions)
	}
	if inline.Actions.Left < inline.Title.Right {
		t.Errorf("строка команд не справа от заголовка: заголовок %+v, команды %+v", inline.Title, inline.Actions)
	}
	if inline.Close.Left < inline.Actions.Left {
		t.Errorf("«×» не в конце строки: команды %+v, × %+v", inline.Actions, inline.Close)
	}
	if inline.Card.Top < inline.Actions.Bottom || inline.Card.Top < inline.Title.Bottom {
		t.Errorf("содержимое формы залезло в строку заголовка: %+v", inline)
	}

	plain := measure(renderHeaderInlineForm(t, ""))
	if plain.Actions.Top < plain.Title.Bottom {
		t.Errorf("без ключа строка команд должна быть под заголовком: заголовок %+v, команды %+v", plain.Title, plain.Actions)
	}
	if saved := plain.Card.Top - inline.Card.Top; saved < 20 {
		t.Errorf("header_inline сэкономил по высоте %.0f px, ожидалась хотя бы одна строка", saved)
	}
}
