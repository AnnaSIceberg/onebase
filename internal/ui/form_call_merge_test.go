package ui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestOperatorCallCloseReconcilesOnlyServiceUpdates(t *testing.T) {
	for _, tc := range []struct {
		name            string
		mode            string
		concurrentEdit  bool
		concurrentTable bool
		wrongProof      bool
	}{
		{name: "save", mode: "save"},
		{name: "close handler writes", mode: "discard"},
		{name: "competing business edit", mode: "save", concurrentEdit: true},
		{name: "competing table part", mode: "save", concurrentTable: true},
		{name: "unverified opening", mode: "save", wrongProof: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entity := &metadata.Entity{
				Name: "А_Звонок", Kind: metadata.KindDocument,
				TableParts: []metadata.TablePart{{Name: "ЛогПрослушки", Fields: []metadata.Field{{Name: "ИмяФайла", Type: metadata.FieldTypeString}}}},
				Fields: []metadata.Field{
					{Name: "Комментарий", Type: metadata.FieldTypeString},
					{Name: "ИдЗаписи", Type: metadata.FieldTypeString},
					{Name: "НомерВ1С", Type: metadata.FieldTypeString},
					{Name: "Блокировка", Type: metadata.FieldTypeBool},
				},
			}
			entity.Forms = []*metadata.FormModule{{
				Name: "Object", Kind: "object", EntityName: entity.Name, LayoutKind: metadata.FormLayoutManaged,
				Elements: []*metadata.FormElement{
					{Kind: metadata.FormElementField, Name: "Comment", DataPath: "Объект.Комментарий"},
					{Kind: metadata.FormElementField, Name: "Export", DataPath: "Объект.НомерВ1С", ReadOnly: true},
				},
				Handlers:   map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"},
				ProgramAST: mustParse(t, "Процедура ПроверитьЗакрытие(Отказ)\nОбъект.Блокировка = Истина;\nОбъект.Записать();\nКонецПроцедуры"),
			}}
			srv, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
			id := uuid.New()
			first := map[string]any{"Комментарий": "Исходный", "НомерВ1С": "старый"}
			if err := srv.store.Upsert(ctx, entity.Name, id, first, entity); err != nil {
				t.Fatal(err)
			}

			operator := &auth.User{ID: uuid.NewString(), Login: "test-call-operator",
				Roles: []*auth.Role{{Name: "ОператорКЦ", Permissions: auth.Permission{
					Documents: map[string][]string{entity.Name: {"read", "write"}},
				}}},
			}
			router := chi.NewRouter()
			srv.Mount(router)
			request := func(method, path, body string) *http.Request {
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.AddCookie(&http.Cookie{Name: "onebase_session", Value: "test-session"})
				return req.WithContext(auth.ContextWithUser(req.Context(), operator))
			}
			base := "/ui/document/" + entity.Name
			opened := httptest.NewRecorder()
			router.ServeHTTP(opened, request(http.MethodGet, base+"/"+id.String(), ""))
			proof := regexp.MustCompile("name=\"_ob_call_business\" value=\"([a-f0-9]{64})\"").FindStringSubmatch(opened.Body.String())
			if opened.Code != http.StatusOK || len(proof) != 2 {
				t.Fatalf("call card did not render opening proof: status %d", opened.Code)
			}
			row, err := srv.store.GetByID(ctx, entity.Name, id, entity)
			if err != nil {
				t.Fatal(err)
			}
			oldVersion := persistedEntityVersion(row)
			second := map[string]any{"Комментарий": "Исходный", "НомерВ1С": "подтверждённый", "ИдЗаписи": "recording"}
			if tc.concurrentEdit {
				second["Комментарий"] = "Чужая правка"
			}
			if err := srv.store.Upsert(ctx, entity.Name, id, second, entity); err != nil {
				t.Fatal(err)
			}

			if tc.concurrentTable {
				table := entity.TableParts[0]
				if err := srv.store.UpsertTablePartRows(ctx, entity.Name, table.Name, id,
					[]map[string]any{{"ИмяФайла": "new-audit-entry"}}, table); err != nil {
					t.Fatal(err)
				}
			}
			body := closeIntentBody(uuid.NewString(), "close", "")
			body.Set("_id", id.String())
			body.Set("_version", strconv.FormatInt(oldVersion, 10))
			body.Set("_close_mode", tc.mode)
			body.Set(callFormProofField, proof[1])
			if tc.wrongProof {
				body.Set(callFormProofField, strings.Repeat("0", 64))
			}
			body.Set("Комментарий", "Правка оператора")
			body.Set("НомерВ1С", "старый")
			ensureEntityCloseIntentSchema(entity, body)
			req := request(http.MethodPost, base+"/form-close-intent", body.Encode())
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			setEntityCloseIntentHeaders(req, body)
			closeResponse := httptest.NewRecorder()
			router.ServeHTTP(closeResponse, req)
			result := decodeCloseIntentResponse(t, closeResponse)
			expectedComment := "Правка оператора"
			if tc.concurrentEdit || tc.concurrentTable || tc.wrongProof {
				if closeResponse.Code != http.StatusConflict || result.Close == nil || result.Close.Allowed {
					t.Fatalf("unsafe rebase accepted: status=%d, result=%+v", closeResponse.Code, result)
				}
				expectedComment = second["Комментарий"].(string)
			} else if closeResponse.Code != http.StatusOK || result.Close == nil || !result.Close.Allowed || !result.Close.Saved {
				t.Fatalf("call did not save and close: status=%d, result=%+v", closeResponse.Code, result)
			}
			row, err = srv.store.GetByID(ctx, entity.Name, id, entity)
			if err != nil || row["Комментарий"] != expectedComment || row["ИдЗаписи"] != "recording" || row["НомерВ1С"] != "подтверждённый" {
				t.Fatalf("durable call state: %v, err=%v", row, err)
			}
		})
	}
}
