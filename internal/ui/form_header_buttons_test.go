package ui

import (
	"fmt"
	"github.com/ivantit66/onebase/internal/auth"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Кнопка с placement: header рисуется в строке команд документа — сразу после
// «История» у записанного объекта, в конце строки у нового — и не рисуется в
// теле формы. Карточка открыта штатными обработчиками формы.
func TestManagedFormHeaderPlacementButton(t *testing.T) {
	btn := func() *metadata.FormElement {
		return &metadata.FormElement{
			Kind: metadata.FormElementButton, Name: "КнУведомить", Placement: metadata.FormPlacementHeader,
			TitleMap: map[string]string{"ru": "Уведомить тех. отдел"},
			Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Уведомить"},
		}
	}
	check := func(t *testing.T, body string, after string) {
		t.Helper()
		if n := strings.Count(body, `data-ob-fire-click="КнУведомить"`); n != 1 {
			t.Fatalf("кнопка должна быть ровно одна, найдено %d:\n%s", n, body)
		}
		at := strings.Index(body, `data-ob-fire-click="КнУведомить"`)
		if form := strings.Index(body, `<form id="main-form"`); form < 0 || at > form {
			t.Fatalf("кнопка не в строке команд (после начала тела формы):\n%s", body)
		}
		if after != "" {
			if h := strings.Index(body, after); h < 0 || h > at {
				t.Fatalf("кнопка не стоит после %q:\n%s", after, body)
			}
		}
	}

	t.Run("новая запись", func(t *testing.T) {
		ent := layoutTestEntity(btn())
		srv, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
		req := reqWithChi(http.MethodGet, "/ui/catalog/"+ent.Name+"/new", nil, map[string]string{"entity": ent.Name})
		rec := httptest.NewRecorder()
		srv.form(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("form: %d", rec.Code)
		}
		check(t, rec.Body.String(), "")
	})
	t.Run("записанная — после «История»", func(t *testing.T) {
		srv, ent := setupManagedEventsServer(t, "Процедура Уведомить()\nКонецПроцедуры\n", nil, []*metadata.FormElement{
			{Kind: metadata.FormElementField, Name: "Наименование", DataPath: "Объект.Наименование"}, btn(),
		})
		id := insertContragent(t, srv, ent, "Ромашка")
		rec := executeFormEditGET(t, srv, ent, id)
		if rec.Code != http.StatusOK {
			t.Fatalf("formEdit: %d", rec.Code)
		}
		check(t, rec.Body.String(), "/history\" class=\"btn btn-sm btn-secondary\">")
	})
}

// The popup route must expose the same command exactly once for both a new
// object and an existing object. Native form ownership also makes it a hotkey
// candidate even though the command bar is outside the form subtree.
func TestManagedHeaderButtonPopupAndFormOwnership(t *testing.T) {
	for _, popup := range []bool{false, true} {
		for _, saved := range []bool{false, true} {
			t.Run(fmt.Sprintf("popup=%t/saved=%t", popup, saved), func(t *testing.T) {
				button := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "HeaderAction", Placement: metadata.FormPlacementHeader, HotKey: "F9", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Run"}}
				srv, ent := setupManagedEventsServer(t, `Процедура Run()
 Сообщить("done");
КонецПроцедуры`, nil, []*metadata.FormElement{button})
				path := "/ui/catalog/" + ent.Name + "/new"
				params := map[string]string{"kind": "catalog", "entity": ent.Name}
				if saved {
					id := insertContragent(t, srv, ent, "Example")
					path = "/ui/catalog/" + ent.Name + "/" + id.String()
					params["id"] = id.String()
				}
				if popup {
					path += "?_popup=1"
				}
				rec := httptest.NewRecorder()
				req := reqWithChi(http.MethodGet, path, nil, params)
				if saved {
					srv.formEdit(rec, req)
				} else {
					srv.form(rec, req)
				}
				if rec.Code != http.StatusOK {
					t.Fatalf("GET: %d %s", rec.Code, rec.Body.String())
				}
				body := rec.Body.String()
				if count := strings.Count(body, `data-ob-fire-click="HeaderAction"`); count != 1 {
					t.Fatalf("command count=%d", count)
				}
				anchor := managedAnchorNode(t, body, "HeaderAction")
				if owner, _ := managedHTMLAttr(anchor, "form"); owner != "main-form" {
					t.Fatalf("native form owner=%q", owner)
				}
				if key, _ := managedHTMLAttr(anchor, "data-ob-hotkey"); key != "F9" {
					t.Fatalf("hotkey=%q", key)
				}
			})
		}
	}
}

func TestManagedHeaderButtonHiddenAncestorsAndEventUpdates(t *testing.T) {
	button := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "HeaderAction", Placement: metadata.FormPlacementHeader, Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Run"}}
	elements := []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "Name", DataPath: "Объект.Наименование"},
		{Kind: metadata.FormElementGroupBox, Name: "Outer", HiddenWhen: `Наименование = "outer"`, Children: []*metadata.FormElement{
			{Kind: metadata.FormElementGroupBox, Name: "Inner", HiddenWhen: `Наименование = "inner"`, Children: []*metadata.FormElement{button}},
		}},
		{Kind: metadata.FormElementButton, Name: "Refresh", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Run"}},
	}
	srv, ent := setupManagedEventsServer(t, "Процедура Run()\nКонецПроцедуры", nil, elements)
	id := insertContragent(t, srv, ent, "outer")
	rec := executeFormEditGET(t, srv, ent, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: %d", rec.Code)
	}
	anchor := managedAnchorNode(t, rec.Body.String(), "HeaderAction")
	if anchor == nil || !managedStyleHidden(anchor) {
		t.Fatal("header command escaped its hidden ancestor")
	}
	path, _ := managedHTMLAttr(anchor, "data-ob-el-path")
	for _, value := range []string{"visible", "inner", "outer", "visible"} {
		body := url.Values{"_id": {id.String()}, "_element": {"Refresh"}, "_event": {string(metadata.FormEventOnClick)}, "_kind": {"object"}, "Наименование": {value}}
		response := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
		if !response.OK || response.ElementStates == nil {
			t.Fatalf("form-event: %+v", response)
		}
		hidden, present := response.ElementStates.Hidden[path]
		if !present || hidden != (value != "visible") {
			t.Fatalf("%q: command hidden=%t present=%t", value, hidden, present)
		}
	}
}

func TestManagedHeaderTableCommandKeepsContextOnce(t *testing.T) {
	for _, valueTable := range []bool{false, true} {
		for _, noGrid := range []bool{false, true} {
			for _, readOnly := range []bool{false, true} {
				t.Run(fmt.Sprintf("valueTable=%t/noGrid=%t/readonly=%t", valueTable, noGrid, readOnly), func(t *testing.T) {
					button := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "TableAction", Placement: metadata.FormPlacementHeader, Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Run"}}
					table := &metadata.FormElement{Kind: metadata.FormElementTablePart, Name: "Table", DataPath: "Объект.lines", NoGrid: noGrid, ReadOnly: readOnly, Children: []*metadata.FormElement{button}}
					srv, ent := setupManagedEventsServer(t, `Процедура Run()
Сообщить("table action");
КонецПроцедуры`, nil, []*metadata.FormElement{table})
					if valueTable {
						ent.Forms[0].Attributes = []*metadata.FormAttribute{{Name: "Lines", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{{Name: "Note", TypeRef: "string"}}}}
					} else {
						ent.TableParts = []metadata.TablePart{{Name: "Lines", Fields: []metadata.Field{{Name: "Note", Type: metadata.FieldTypeString}}}}
						if err := srv.store.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
							t.Fatal(err)
						}
					}
					req := reqWithChi(http.MethodGet, "/ui/catalog/"+ent.Name+"/new", nil, map[string]string{"entity": ent.Name})
					rec := httptest.NewRecorder()
					srv.form(rec, req)
					if rec.Code != http.StatusOK {
						t.Fatalf("GET %d: %s", rec.Code, rec.Body.String())
					}
					rendered := rec.Body.String()
					if count := strings.Count(rendered, `data-ob-el="TableAction"`); count != 1 {
						t.Fatalf("command count=%d", count)
					}
					command := managedAnchorNode(t, rendered, "TableAction")
					_, disabled := managedHTMLAttr(command, "disabled")
					if disabled != readOnly {
						t.Fatalf("table readonly not preserved: disabled=%t", disabled)
					}
					if !readOnly {
						if count := strings.Count(rendered, `data-ob-fire-click="TableAction"`); count != 1 {
							t.Fatalf("duplicate command actions: %d", count)
						}
						if tp, _ := managedHTMLAttr(command, "data-ob-fire-tp"); tp != "Lines" {
							t.Fatalf("table context=%q", tp)
						}
						body := url.Values{"_element": {"TableAction"}, "_event": {string(metadata.FormEventOnClick)}, "_kind": {"object"}, "_tp": {"Lines"}, "Наименование": {"Example"}}
						response := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
						if !response.OK || len(response.Messages) != 1 || response.Messages[0] != "table action" {
							t.Fatalf("table event: %+v", response)
						}
					}
				})
			}
		}
	}
}

func TestManagedHeaderTableCommandReadOnlyUser(t *testing.T) {
	button := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "TableAction", Placement: metadata.FormPlacementHeader, Width: 180, Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Run"}}
	table := &metadata.FormElement{Kind: metadata.FormElementTablePart, Name: "Table", DataPath: "Form.Lines", Children: []*metadata.FormElement{button}}
	srv, ent := setupManagedEventsServer(t, "Процедура Run()\nКонецПроцедуры", nil, []*metadata.FormElement{table})
	ent.Forms[0].Attributes = []*metadata.FormAttribute{{Name: "Lines", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{{Name: "Note", TypeRef: "string"}}}}
	id := insertContragent(t, srv, ent, "Example")
	user := &auth.User{Roles: []*auth.Role{{Permissions: auth.Permission{Catalogs: map[string][]string{ent.Name: {"read"}}}}}}
	req := reqWithChi(http.MethodGet, "/ui/catalog/"+ent.Name+"/"+id.String(), nil, map[string]string{"entity": ent.Name, "id": id.String()})
	req = req.WithContext(auth.ContextWithUser(req.Context(), user))
	rec := httptest.NewRecorder()
	srv.formEdit(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: %d %s", rec.Code, rec.Body.String())
	}
	anchor := managedAnchorNode(t, rec.Body.String(), "TableAction")
	command := managedDescendantButton(anchor)
	if command == nil {
		t.Fatal("missing table command")
	}
	if _, disabled := managedHTMLAttr(command, "disabled"); !disabled {
		t.Fatal("read-only user received enabled table command")
	}
	if denied, _ := managedHTMLAttr(command, "data-ob-write-denied"); denied != "1" {
		t.Fatal("write restriction must survive client state updates")
	}
}
