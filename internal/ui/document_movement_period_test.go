package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/widget"
)

// Use the mounted document route: a formatter-only test would miss either
// template forgetting to format SQLite's TEXT period (issue #2120).
func movementPeriodServer(t *testing.T, db *storage.DB, managed bool) (*registerMaskFixture, http.Handler) {
	t.Helper()
	ctx := context.Background()
	doc := &metadata.Entity{Name: "PeriodDocument", Kind: metadata.KindDocument, Posting: true,
		Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}}}
	if managed {
		doc.Forms = []*metadata.FormModule{{Name: "ФормаОбъекта", Kind: "object", EntityName: doc.Name,
			LayoutKind: metadata.FormLayoutManaged,
			Elements:   []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "Номер", DataPath: "Объект.Номер"}}}}
	}
	reg := &metadata.Register{Name: "PeriodMovements",
		Dimensions: []metadata.Field{{Name: "Область", Type: metadata.FieldTypeString}},
		Resources:  []metadata.Field{{Name: "Количество", Type: metadata.FieldTypeNumber}},
		Attributes: []metadata.Field{{Name: "Метка", Type: metadata.FieldTypeString}}}
	if err := db.Migrate(ctx, []*metadata.Entity{doc}); err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateRegisters(ctx, []*metadata.Register{reg}); err != nil {
		t.Fatal(err)
	}
	registry := runtime.NewRegistry()
	registry.Load(runtime.LoadOptions{Entities: []*metadata.Entity{doc}, Registers: []*metadata.Register{reg}})
	interp := interpreter.New()
	interp.LookupProc = registry.GetModuleProc
	s := &Server{store: db, reg: registry, interp: interp, lockMgr: runtime.NewLockManager(),
		messages: NewMessageStore(), widgetCache: widget.NewCache(time.Minute), aiChatLimit: newAIWindowLimiter(1000, time.Minute)}
	s.entitySvc = s.newEntityService(nil)
	id := uuid.New()
	if err := db.Upsert(ctx, doc.Name, id, map[string]any{"Номер": "PERIOD-2120"}, doc); err != nil {
		t.Fatal(err)
	}
	if err := db.SetPosted(ctx, doc.Name, id, true); err != nil {
		t.Fatal(err)
	}
	f := &registerMaskFixture{s: s, ctx: ctx, doc: doc, reg: reg, recorderID: id, goods: &metadata.Entity{Name: "UnusedCatalog"}}
	r := chi.NewRouter()
	s.Mount(r)
	return f, r
}

func writePeriodMovements(t *testing.T, f *registerMaskFixture, period time.Time) {
	t.Helper()
	if err := f.s.store.WriteMovements(f.ctx, f.reg.Name, f.doc.Name, f.recorderID, []map[string]any{
		{"Область": "visible", "Количество": 1, "Метка": "VISIBLE-PERIOD-ROW"},
		{"Область": "hidden", "Количество": 2, "Метка": "HIDDEN-PERIOD-ROW"},
	}, f.reg, &period); err != nil {
		t.Fatal(err)
	}
}

func getMovementPeriodForm(t *testing.T, f *registerMaskFixture, router http.Handler, user *auth.User) string {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/ui/document/"+url.PathEscape(f.doc.Name)+"/"+f.recorderID.String(), nil)
	r = r.WithContext(auth.ContextWithUser(r.Context(), user))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET form: status %d: %s", w.Code, w.Body.String())
	}
	page := w.Body.String()
	if len(f.doc.Forms) > 0 {
		if !strings.Contains(page, "data-ob-grid-sync") {
			t.Fatal("managed template not rendered")
		}
	} else if !strings.Contains(page, "data-ob-dirty-watch") {
		t.Fatal("ordinary template not rendered")
	}
	return page
}

func movementPeriodDSL(t *testing.T, f *registerMaskFixture) []any {
	t.Helper()
	prog := mustParse(t, `Функция Тест()
  А = РегистрыНакопления.PeriodMovements.Выбрать()[0].Период;
  Б = РегистрыНакопления.PeriodMovements.Движения()[0].Период;
  В = РегистрыНакопления.PeriodMovements.ВыбратьПоРегистратору(Док)[0].Период;
  Возврат [ТипЗнч(А), А, ТипЗнч(Б), Б, ТипЗнч(В), В];
КонецФункции`)
	var result any
	if err := f.s.interp.RunWithResult(prog.Procedures[0], nil, &result, map[string]any{
		"РегистрыНакопления": newAccumRegsRoot(f.s, interpreter.NewTxState(f.ctx)),
		"Док":                &interpreter.Ref{UUID: f.recorderID.String(), Type: f.doc.Name},
	}); err != nil {
		t.Fatal(err)
	}
	return result.(*interpreter.Array).Iterate()
}

func requireMovementDSLPeriod(t *testing.T, values []any, want time.Time) {
	t.Helper()
	for i, method := range []string{"Выбрать", "Движения", "ВыбратьПоРегистратору"} {
		period, ok := values[2*i+1].(time.Time)
		if values[2*i] != "Дата" || !ok || !period.Equal(want) {
			t.Fatalf("%s DSL period = %v (%T); want Date at %v", method, values[2*i+1], values[2*i+1], want)
		}
	}
}

func TestDocumentMovementPeriodLocalTimeMatrix(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })
	for _, zone := range []struct {
		name   string
		offset int
	}{{"UTCplus3", 3 * 3600}, {"UTCminus5", -5 * 3600}} {
		t.Run(zone.name, func(t *testing.T) {
			time.Local = time.FixedZone(zone.name, zone.offset)
			for _, managed := range []bool{false, true} {
				t.Run(fmt.Sprintf("managed=%t", managed), func(t *testing.T) {
					dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
						f, router := movementPeriodServer(t, db, managed)
						reader := f.reader("period-reader", true, nil, nil)
						for _, tc := range []struct {
							name  string
							local time.Time
							want  string
						}{
							{"seconds", time.Date(2026, 8, 15, 15, 33, 22, 0, time.Local), "15.08.2026 15:33:22"},
							{"after-midnight", time.Date(2026, 8, 15, 0, 0, 7, 0, time.Local), "15.08.2026 00:00:07"},
							{"before-midnight", time.Date(2026, 8, 15, 23, 59, 58, 0, time.Local), "15.08.2026 23:59:58"},
							{"midnight", time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local), "15.08.2026"},
							{"fractional-seconds", time.Date(2026, 8, 15, 15, 33, 22, 123456000, time.Local), "15.08.2026 15:33:22"},
						} {
							t.Run(tc.name, func(t *testing.T) {
								// Pass a UTC instant into storage, independently of the host zone.
								writePeriodMovements(t, f, tc.local.UTC())
								before := movementPeriodDSL(t, f)
								want := tc.local
								if db.IsSQLite() {
									// The existing SQLite writer stores whole seconds; PostgreSQL
									// stores microseconds. Reading must retain the saved precision.
									want = want.Truncate(time.Second)
								}
								requireMovementDSLPeriod(t, before, want)
								rawBefore, err := db.GetMovements(f.ctx, f.reg.Name, f.reg, storage.RegFilter{})
								if err != nil {
									t.Fatal(err)
								}
								page := getMovementPeriodForm(t, f, router, reader)
								if !strings.Contains(page, "<td>"+tc.want+"</td>") {
									t.Errorf("movement period cell missing %q", tc.want)
								}
								if !reflect.DeepEqual(before, movementPeriodDSL(t, f)) {
									t.Fatal("form rendering changed DSL period type or value")
								}
								rawAfter, err := db.GetMovements(f.ctx, f.reg.Name, f.reg, storage.RegFilter{})
								if err != nil {
									t.Fatal(err)
								}
								if !reflect.DeepEqual(rawBefore, rawAfter) {
									t.Fatal("form rendering changed storage read payload")
								}
							})
						}
						period := time.Date(2026, 8, 15, 15, 33, 22, 0, time.Local)
						writePeriodMovements(t, f, period.UTC())
						t.Run("access", func(t *testing.T) {
							denied := getMovementPeriodForm(t, f, router, f.reader("no-register-read", false, nil, nil))
							if strings.Contains(denied, "PERIOD-ROW") {
								t.Fatal("movements disclosed without register read")
							}
							policy := auth.RowPolicy{Field: "Область", Op: "eq", Value: auth.RowValue{Literal: "visible"}}
							filtered := getMovementPeriodForm(t, f, router, f.reader("row-filter", true, nil, &policy))
							if !strings.Contains(filtered, "VISIBLE-PERIOD-ROW") || strings.Contains(filtered, "HIDDEN-PERIOD-ROW") || !strings.Contains(filtered, "<td>15.08.2026 15:33:22</td>") {
								t.Fatal("period formatting bypassed or broke RLS")
							}
							for _, strategy := range []string{"mask_all", "hide"} {
								masked := getMovementPeriodForm(t, f, router, f.reader("period-"+strategy, true, auth.FieldPolicies{"period": {Read: strategy}}, nil))
								if !strings.Contains(masked, "VISIBLE-PERIOD-ROW") {
									t.Fatal("period policy removed unrelated movement")
								}
								if strings.Contains(masked, "15.08.2026") || strings.Contains(masked, "2026-08-15") {
									t.Fatalf("period disclosed under %s", strategy)
								}
								if strategy == "hide" && strings.Contains(masked, "<th>period</th>") {
									t.Fatal("hidden period header remains")
								}
							}
						})
					})
				})
			}
		})
	}
}

func TestDocumentMovementPeriodSQLiteTextFormats(t *testing.T) {
	saved := time.Local
	time.Local = time.FixedZone("UTCplus3", 3*3600)
	t.Cleanup(func() { time.Local = saved })
	// Existing and legacy SQLite spellings should follow the common UI parser;
	// offset-free values are local wall time and must not receive a second shift.
	for _, managed := range []bool{false, true} {
		t.Run(fmt.Sprintf("managed=%t", managed), func(t *testing.T) {
			dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
				if !db.IsSQLite() {
					t.Skip("SQLite TEXT transport formats")
				}
				f, router := movementPeriodServer(t, db, managed)
				writePeriodMovements(t, f, time.Date(2026, 8, 15, 12, 33, 22, 0, time.UTC))
				for _, raw := range []string{
					"2026-08-15 12:33:22+00:00", "2026-08-15T12:33:22Z", "2026-08-15T12:33:22.123456Z",
					"2026-08-15 12:33:22 +0000 UTC", "2026-08-15T15:33:22+03:00", "2026-08-15 15:33:22",
				} {
					t.Run(raw, func(t *testing.T) {
						if _, err := db.Exec(f.ctx, "UPDATE "+metadata.RegisterTableName(f.reg.Name)+" SET period = ?", raw); err != nil {
							t.Fatal(err)
						}
						page := getMovementPeriodForm(t, f, router, f.reader("text-period", true, nil, nil))
						if !strings.Contains(page, "<td>15.08.2026 15:33:22</td>") {
							t.Fatalf("raw period %q not rendered as local time", raw)
						}
						// Keep the existing DSL parser's interpretation of machine
						// timestamps: offsets identify instants, no offset means local
						// wall time. The legacy Go spelling is covered above for display.
						if raw != "2026-08-15 12:33:22 +0000 UTC" {
							want := time.Date(2026, 8, 15, 15, 33, 22, 0, time.Local)
							if raw == "2026-08-15T12:33:22.123456Z" {
								want = want.Add(123456 * time.Microsecond)
							}
							requireMovementDSLPeriod(t, movementPeriodDSL(t, f), want)
						}
					})
				}
			})
		})
	}
}

func TestMovementPeriodDSLAccessMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f, router := movementPeriodServer(t, db, false)
		period := time.Date(2026, 8, 15, 12, 33, 22, 0, time.UTC)
		writePeriodMovements(t, f, period)
		policy := auth.RowPolicy{Field: "Область", Op: "eq", Value: auth.RowValue{Literal: "visible"}}
		for _, call := range []string{"Выбрать()", "Движения()", "ВыбратьПоРегистратору(Док)"} {
			t.Run(call, func(t *testing.T) {
				prog := mustParse(t, `Функция Тест()
  Строки = РегистрыНакопления.PeriodMovements.`+call+`;
  Возврат [Строки.Количество(), ТипЗнч(Строки[0].Период), Строки[0].Период,
    ТипЗнч(Строки[0].Метка), Строки[0].Метка];
КонецФункции`)
				for _, strategy := range []string{"", "mask_all", "hide", "denied"} {
					t.Run("period-policy="+strategy, func(t *testing.T) {
						var fields auth.FieldPolicies
						if strategy == "mask_all" || strategy == "hide" {
							fields = auth.FieldPolicies{"period": {Read: strategy}}
						}
						user := f.reader(call+strategy, strategy != "denied", fields, &policy)
						ctx := auth.ContextWithUser(f.ctx, user)
						var result any
						err := f.s.interp.RunWithResult(prog.Procedures[0], nil, &result, map[string]any{
							"РегистрыНакопления": newAccumRegsRoot(f.s, interpreter.NewTxState(ctx)),
							"Док":                &interpreter.Ref{UUID: f.recorderID.String(), Type: f.doc.Name},
						})
						if strategy == "denied" {
							if err == nil {
								t.Fatal("DSL disclosed movements without register read")
							}
							return
						}
						if err != nil {
							t.Fatal(err)
						}
						values := result.(*interpreter.Array).Iterate()
						if values[0] != float64(1) || values[3] != "Строка" || values[4] != "VISIBLE-PERIOD-ROW" {
							t.Fatalf("RLS or unrelated string field changed: %v", values)
						}
						switch strategy {
						case "":
							got, ok := values[2].(time.Time)
							if values[1] != "Дата" || !ok || !got.Equal(period) {
								t.Fatalf("DSL period under RLS = %v; want Date at %v", values, period)
							}
						case "mask_all":
							if values[1] != "Строка" || values[2] != "••••••" {
								t.Fatalf("masked period disclosed: %v", values)
							}
						case "hide":
							if values[2] != nil {
								t.Fatalf("hidden period disclosed: %v", values)
							}
						}
						// The register list consumes the same storage rows. Its access
						// filters must remain intact after changing the period payload.
						r := httptest.NewRequest(http.MethodGet, "/ui/register/"+strings.ToLower(f.reg.Name), nil)
						r = r.WithContext(ctx)
						w := httptest.NewRecorder()
						router.ServeHTTP(w, r)
						if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "VISIBLE-PERIOD-ROW") || strings.Contains(w.Body.String(), "HIDDEN-PERIOD-ROW") {
							t.Fatalf("register list broke RLS: status %d", w.Code)
						}
					})
				}
			})
		}
	})
}

func TestDocumentMovementExplicitRowPeriodMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f, _ := movementPeriodServer(t, db, false)
		documentDate := time.Date(2026, 11, 20, 9, 0, 0, 0, time.Local)
		movementDate := documentDate.Add(48 * time.Hour)

		write := func(dsl string, vars map[string]any) error {
			t.Helper()
			mc := runtime.NewMovementsCollector(f.doc.Name, f.recorderID).WillPersist()
			proc := mustParse(t, "Процедура Заполнение()\n"+dsl+"\nКонецПроцедуры")
			vars["Движения"] = mc
			if err := f.s.interp.Run(proc.Procedures[0], nil, vars); err != nil {
				t.Fatal(err)
			}
			for name, rows := range mc.All() {
				if strings.EqualFold(name, f.reg.Name) {
					return db.WriteMovements(f.ctx, f.reg.Name, f.doc.Name, f.recorderID, rows, f.reg, &documentDate)
				}
			}
			t.Fatal("DSL did not collect rows for the register")
			return nil
		}
		check := func() {
			t.Helper()
			rows, err := db.GetMovements(f.ctx, f.reg.Name, f.reg, storage.RegFilter{})
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 {
				t.Fatalf("got %d movement rows, want 2", len(rows))
			}
			want := []time.Time{documentDate, movementDate}
			for i, row := range rows {
				got := storage.ParseDBTime(row["period"])
				if !got.Equal(want[i]) {
					t.Fatalf("row %d period = %v, want %v", i, row["period"], want[i])
				}
			}
		}
		if err := write("А = Движения.PeriodMovements.Добавить();\nБ = Движения.PeriodMovements.Добавить(ДатаРасхода);", map[string]any{"ДатаРасхода": movementDate}); err != nil {
			t.Fatal(err)
		}
		check()

		if err := write("А = Движения.PeriodMovements.Добавить(\"ошибка\");", map[string]any{}); err == nil {
			t.Fatal("invalid period was accepted")
		}
		check()

		if err := db.SavePostingLockDate(f.ctx, documentDate.AddDate(0, 0, -1)); err != nil {
			t.Fatal(err)
		}
		if err := write("А = Движения.PeriodMovements.Добавить(ЗакрытыйДень);", map[string]any{"ЗакрытыйДень": documentDate.Add(-24 * time.Hour)}); err == nil {
			t.Fatal("explicit period bypassed posting lock")
		}
		check()
	})
}

func TestDocumentMovementReplacementClosedPeriodMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f, _ := movementPeriodServer(t, db, false)
		closed := time.Date(2026, 9, 30, 23, 59, 59, 0, time.Local)
		open := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
		if storage.PostingFrozen(closed, open.UTC()) || !storage.PostingFrozen(closed, closed.UTC()) {
			t.Fatal("posting lock must use the local calendar boundary")
		}
		f.doc.Fields = append(f.doc.Fields, metadata.Field{Name: "Дата", Type: metadata.FieldTypeDate})
		if err := db.Migrate(f.ctx, []*metadata.Entity{f.doc}); err != nil {
			t.Fatal(err)
		}
		if err := db.Upsert(f.ctx, f.doc.Name, f.recorderID, map[string]any{"Дата": open}, f.doc); err != nil {
			t.Fatal(err)
		}
		before := []map[string]any{
			{metadata.ExplicitMovementPeriodKey: closed, "Количество": 1},
			{metadata.ExplicitMovementPeriodKey: open, "Количество": 2},
		}
		if err := db.WriteMovements(f.ctx, f.reg.Name, f.doc.Name, f.recorderID, before, f.reg, &open); err != nil {
			t.Fatal(err)
		}
		if err := db.SavePostingLockDate(f.ctx, closed); err != nil {
			t.Fatal(err)
		}
		assertUnchanged := func() {
			t.Helper()
			rows, err := db.GetMovements(f.ctx, f.reg.Name, f.reg, storage.RegFilter{})
			if err != nil || len(rows) != 2 {
				t.Fatalf("saved movements lost on refusal: %#v, %v", rows, err)
			}
			row, err := db.GetByID(f.ctx, f.doc.Name, f.recorderID, f.doc)
			if err != nil || fmt.Sprint(row["posted"]) == "false" || fmt.Sprint(row["posted"]) == "0" {
				t.Fatalf("posted header lost on refusal: %#v, %v", row, err)
			}
		}
		result, err := f.s.entitySvc.Unpost(f.ctx, f.doc, f.recorderID)
		if err != nil || !strings.Contains(result.DSLError, "свёрнутому периоду") {
			t.Fatalf("unpost must report closed movement: %+v, %v", result, err)
		}
		assertUnchanged()
		if err := f.s.entitySvc.Repost(f.ctx, f.doc.Name, f.recorderID); err == nil || !strings.Contains(err.Error(), "свёрнутому периоду") {
			t.Fatalf("repost of open header must check the previous, closed movement: %v", err)
		}
		assertUnchanged()
		if err := db.WriteMovements(f.ctx, f.reg.Name, f.doc.Name, f.recorderID, []map[string]any{
			{metadata.ExplicitMovementPeriodKey: open, "Количество": 99},
		}, f.reg, &open); err == nil || !strings.Contains(err.Error(), "свёрнутому периоду") {
			t.Fatalf("replacement must check the previous, closed movement: %v", err)
		}
		assertUnchanged()
	})
}

func TestWarehouseRollupDocumentMovementBoundaryMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		doc := &metadata.Entity{Name: "WorkshopRollupDocument", Kind: metadata.KindDocument, Posting: true,
			Fields: []metadata.Field{{Name: "Дата", Type: metadata.FieldTypeDate}}}
		stock := &metadata.Register{Name: "WorkshopRollupStock",
			Resources: []metadata.Field{{Name: "Количество", Type: metadata.FieldTypeNumber}}}
		debts := &metadata.Register{Name: "WorkshopRollupDebts",
			Resources: []metadata.Field{{Name: "Количество", Type: metadata.FieldTypeNumber}}}
		regs := []*metadata.Register{stock, debts}
		docs := []*metadata.Entity{doc}
		if err := db.Migrate(ctx, docs); err != nil {
			t.Fatal(err)
		}
		if err := db.MigrateRegisters(ctx, regs); err != nil {
			t.Fatal(err)
		}
		cutoff := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		early := cutoff.Add(-time.Hour)
		late := cutoff
		id := uuid.New()
		if err := db.Upsert(ctx, doc.Name, id, map[string]any{"Дата": early}, doc); err != nil {
			t.Fatal(err)
		}
		if err := db.SetPosted(ctx, doc.Name, id, true); err != nil {
			t.Fatal(err)
		}
		write := func(reg *metadata.Register, dates ...time.Time) {
			t.Helper()
			rows := make([]map[string]any, 0, len(dates))
			for _, date := range dates {
				rows = append(rows, map[string]any{metadata.ExplicitMovementPeriodKey: date, "Количество": 1})
			}
			if err := db.WriteMovements(ctx, reg.Name, doc.Name, id, rows, reg, &early); err != nil {
				t.Fatal(err)
			}
		}
		assertBlocked := func(opts storage.RollupOptions, want int) {
			t.Helper()
			if _, err := db.RollupPreview(ctx, regs, docs, nil, nil, opts); err == nil || !strings.Contains(err.Error(), doc.Name) {
				t.Fatalf("unsafe rollup preview must identify recorder: %v", err)
			}
			if _, err := db.Rollup(ctx, regs, docs, nil, nil, opts); err == nil || !strings.Contains(err.Error(), doc.Name) {
				t.Fatalf("unsafe rollup must identify recorder: %v", err)
			}
			row, err := db.GetByID(ctx, doc.Name, id, doc)
			if err != nil || fmt.Sprint(row["posted"]) == "false" || fmt.Sprint(row["posted"]) == "0" {
				t.Fatalf("posted header lost on refusal: %#v, %v", row, err)
			}
			rows, err := db.GetMovements(ctx, stock.Name, stock, storage.RegFilter{})
			if err != nil || len(rows) != want {
				t.Fatalf("refusal changed the movement set: %#v, %v", rows, err)
			}
			if _, set := db.GetPostingLockDate(ctx); set {
				t.Fatal("refused rollup must not close the period")
			}
		}
		write(stock, early, late)
		for _, del := range []bool{false, true} {
			assertBlocked(storage.RollupOptions{Date: cutoff, Registers: []string{stock.Name}, DeleteDocuments: del}, 2)
		}
		write(stock, early)
		write(debts, early)
		assertBlocked(storage.RollupOptions{Date: cutoff, Registers: []string{stock.Name}}, 1)
		if _, err := db.Rollup(ctx, regs, docs, nil, nil, storage.RollupOptions{
			Date: cutoff, Registers: []string{stock.Name, debts.Name},
		}); err != nil {
			t.Fatalf("complete rollup should succeed: %v", err)
		}
	})
}
