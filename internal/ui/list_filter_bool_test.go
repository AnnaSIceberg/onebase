package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Панель «Отбор» списка: булев реквизит — выбор «— все — / Да / Нет», а не
// текстовое поле, и отбор по нему находит записи. Раньше поле было текстовым,
// а введённое «Да» искалось подстрокой и на SQLite не находило ничего.
func TestUI_EntityList_BoolFilter(t *testing.T) {
	задача := &metadata.Entity{
		Name: "Задача", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Выполнена", Type: metadata.FieldTypeBool},
		},
	}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{задача})
	for _, row := range []map[string]any{
		{"Наименование": "Позвонить клиенту", "Выполнена": true},
		{"Наименование": "Проверить отказ", "Выполнена": false},
	} {
		if err := s.store.Upsert(ctx, задача.Name, uuid.New(), row, задача); err != nil {
			t.Fatal(err)
		}
	}
	список := func(query string) string {
		t.Helper()
		target := "/ui/catalog/" + задача.Name
		if query != "" {
			target += "?" + query
		}
		req := reqWithChi(http.MethodGet, target, nil, map[string]string{"kind": "catalog", "entity": задача.Name})
		req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: "root", Login: "admin", IsAdmin: true}))
		rec := httptest.NewRecorder()
		s.list(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list: status=%d body=%s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	html := список("")
	if !strings.Contains(html, `<select name="f.Выполнена">`) {
		t.Fatalf("булев реквизит в панели «Отбор» — не выбор Да/Нет:\n%s", filterPanelFragment(html, "f.Выполнена"))
	}
	if strings.Contains(html, `<input type="text" name="f.Выполнена"`) {
		t.Fatal("булев реквизит остался текстовым полем")
	}

	да := список("f.Выполнена=true")
	if !strings.Contains(да, "Позвонить клиенту") || strings.Contains(да, "Проверить отказ") {
		t.Fatal("отбор «Да»: только выполненная задача")
	}
	if !strings.Contains(да, `<option value="true" selected>`) {
		t.Fatalf("выбранное «Да» не отмечено в панели:\n%s", filterPanelFragment(да, "f.Выполнена"))
	}

	нет := список("f.Выполнена=false")
	if strings.Contains(нет, "Позвонить клиенту") || !strings.Contains(нет, "Проверить отказ") {
		t.Fatal("отбор «Нет»: только невыполненная задача")
	}
}

func filterPanelFragment(html, name string) string {
	i := strings.Index(html, name)
	if i < 0 {
		return "(поля нет)"
	}
	start := i - 200
	if start < 0 {
		start = 0
	}
	end := i + 400
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}
