package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/scheduler"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/widget"
)

func TestPlatformShellRealCopyProfile(t *testing.T) {
	stage := os.Getenv("PLATFORM_SHELL_STAGE")
	if stage == "" {
		t.Skip("own synthetic profile only")
	}
	ctx := context.Background()
	copyPath := os.Getenv("PLATFORM_REAL_SQLITE_COPY")
	if copyPath != "/home/onebase/cc2-release36-dev-profile-snapshot-20261113/dev-profile.sqlite" || os.Getenv("PLATFORM_REAL_READY") != "1" {
		t.Fatal("private ready copy required")
	}
	db, err := storage.ConnectSQLite(ctx, copyPath)
	if err != nil {
		t.Fatal("private copy open failed")
	}
	defer db.Close()
	proj, err := project.Load(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer proj.Close()
	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: proj.Entities, Programs: proj.Programs, ManagerPrograms: proj.ManagerPrograms, ServicePrograms: proj.ServicePrograms, PagePrograms: proj.PagePrograms, Registers: proj.Registers, InfoRegs: proj.InfoRegisters, Enums: proj.Enums, Constants: proj.Constants, Reports: proj.Reports, PrintForms: proj.PrintForms})
	reg.LoadModules(proj.Modules)
	reg.LoadProcessors(proj.Processors)
	reg.LoadPages(proj.Pages)
	reg.LoadSubsystems(proj.Subsystems)
	reg.LoadJournals(proj.Journals)
	reg.LoadAccountRegisters(proj.AccountRegisters, proj.ChartsOfAccounts)
	reg.LoadWidgets(proj.Widgets)
	if os.Getenv("PLATFORM_SHELL_RELEASE36") == "1" {
		reg.GetWidget("ЗадачиСотрудника").ExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
	}
	reg.LoadHomePage(proj.HomePage)
	interp := interpreter.New()
	interp.LookupProc = reg.GetModuleProc
	repo := auth.NewRepo(db)
	actorID := os.Getenv("PLATFORM_REAL_ACTOR_ID")
	if actorID == "" {
		t.Fatal("own test actor required")
	}
	if os.Getenv("PLATFORM_SHELL_STAFF_PROFILE") == "1" {
		baseRoles, err := repo.GetRolesForUser(ctx, actorID)
		if err != nil || len(baseRoles) != 2 {
			t.Fatal("copy pair role fixture differs")
		}
		for _, role := range baseRoles {
			if role.Name != "ОператорКЦ" && role.Name != "ОператорМессенджера" {
				t.Fatal("copy base role differs")
			}
		}
		u, err := repo.Create(ctx, "platform-copy-staff-"+uuid.NewString(), uuid.NewString()+"Aa9!", "Synthetic copy profile", false)
		if err != nil {
			t.Fatal("copy own account create failed")
		}
		defer func() {
			if err := repo.Delete(ctx, u.ID); err != nil {
				t.Error("copy own account cleanup failed")
			}
		}()
		for _, role := range baseRoles {
			if err := repo.AssignRole(ctx, u.ID, role.ID); err != nil {
				t.Fatal("copy role assignment failed")
			}
		}
		staffEntity, positionEntity := reg.GetEntity("Пользователи"), reg.GetEntity("ДолжностиОрганизаций")
		if staffEntity == nil || positionEntity == nil {
			t.Fatal("copy staff metadata missing")
		}
		var departmentID string
		if err := db.QueryRow(ctx, "SELECT id FROM "+metadata.TableName("Подразделения")+" WHERE код=$1", "000000003").Scan(&departmentID); err != nil {
			t.Fatal("copy KC department lookup failed")
		}
		positionID, staffID := uuid.New(), uuid.New()
		if err := db.Upsert(ctx, positionEntity.Name, positionID, map[string]any{"Наименование": "Сотрудник"}, positionEntity); err != nil {
			t.Fatal("copy own position create failed")
		}
		defer func() {
			if err := db.Delete(ctx, positionEntity.Name, positionID); err != nil {
				t.Error("copy own position cleanup failed")
			}
		}()
		if err := db.Upsert(ctx, staffEntity.Name, staffID, map[string]any{"Наименование": "Synthetic platform copy staff", "УчётнаяЗапись": uuid.MustParse(u.ID), "Подразделение": uuid.MustParse(departmentID), "Должность": positionID, "Архив": false}, staffEntity); err != nil {
			t.Fatal("copy own staff create failed")
		}
		defer func() {
			if err := db.Delete(ctx, staffEntity.Name, staffID); err != nil {
				t.Error("copy own staff cleanup failed")
			}
		}()
		var ir *metadata.InfoRegister
		for _, item := range reg.InfoRegisters() {
			if item.Name == "А_ДолжностиПодразделения" {
				ir = item
			}
		}
		if ir == nil {
			t.Fatal("copy job register missing")
		}
		period := time.Now().Add(-time.Hour)
		if err := db.InfoRegSet(ctx, ir, map[string]any{"Пользователь": staffID}, map[string]any{"Подразделение": uuid.MustParse(departmentID), "Должность": positionID}, &period); err != nil {
			t.Fatal("copy own current job create failed")
		}
		defer func() {
			if err := db.InfoRegDelete(ctx, ir, map[string]any{"Пользователь": staffID}, &period); err != nil {
				t.Error("copy own job cleanup failed")
			}
		}()
		var count int
		accountColumn, archiveColumn := "", ""
		for _, field := range staffEntity.Fields {
			if field.Name == "УчётнаяЗапись" {
				accountColumn = metadata.ColumnName(field)
			}
			if field.Name == "Архив" {
				archiveColumn = metadata.ColumnName(field)
			}
		}
		if accountColumn == "" || archiveColumn == "" {
			t.Fatal("copy staff fields missing")
		}
		if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM "+metadata.TableName(staffEntity.Name)+" WHERE "+accountColumn+"=$1 AND "+archiveColumn+"=$2", u.ID, false).Scan(&count); err != nil || count != 1 {
			t.Fatal("copy own active card invariant failed")
		}
		actorID = u.ID
		t.Log("OWN_STAFF_PROFILE active_card_count=1 pair=true nonadmin=true")
	}
	httpOldQuery, httpNewQuery := "", ""
	token, err := repo.CreateSession(ctx, actorID, auth.SessionMeta{Kind: auth.SessionKindEnterprise})
	if err != nil {
		t.Fatal("private copy session create failed")
	}
	defer repo.DeleteSession(context.Background(), token)
	if prototype := os.Getenv("PLATFORM_TASK_DIRECTIONS_EXISTS"); prototype != "" {
		w := reg.GetWidget("СписокЗадач")
		join := "ЛЕВОЕ СОЕДИНЕНИЕ Справочник.Пользователи.СписокНаправлений КАК СН ПО СН.Ссылка = Я.Ссылка И СН.Направление = З.Направление"
		test := "СН.Направление ЕСТЬ НЕ NULL"
		if w == nil || strings.Count(w.Query, join) != 1 || strings.Count(w.Query, test) != 1 {
			t.Fatal("prototype source shape differs")
		}
		optimized := strings.Replace(w.Query, join, "", 1)
		optimized = strings.Replace(optimized, test, `EXISTS (
     ВЫБРАТЬ 1
     ИЗ Справочник.Пользователи.СписокНаправлений КАК ТЧНаправлений
     ГДЕ ТЧНаправлений.Ссылка = Я.Ссылка
       И ТЧНаправлений.Направление = З.Направление
   )`, 1)

		switch prototype {
		case "manager-on":
			optimized = strings.Replace(w.Query, join, join+` И (ДД.Наименование = "Менеджер направления" ИЛИ ДК.Наименование = "Менеджер направления")`, 1)
		case "group-all", "group-all-stable":
			if strings.Count(w.Query, "ВЫБРАТЬ РАЗЛИЧНЫЕ") != 1 || strings.Count(w.Query, "УПОРЯДОЧИТЬ ПО") != 1 {
				t.Fatal("group prototype shape differs")
			}
			projectionEnd := strings.Index(w.Query, "ИЗ Документ.А_Задача")
			if projectionEnd < 0 || strings.Count(w.Query[:projectionEnd], " КАК ") != 12 {
				t.Fatal("group prototype projection count differs")
			}
			optimized = strings.Replace(w.Query, "ВЫБРАТЬ РАЗЛИЧНЫЕ", "ВЫБРАТЬ", 1)
			optimized = strings.Replace(optimized, "УПОРЯДОЧИТЬ ПО", "СГРУППИРОВАТЬ ПО 1,2,3,4,5,6,7,8,9,10,11,12\nУПОРЯДОЧИТЬ ПО", 1)
			if prototype == "group-all-stable" {
				if strings.Count(optimized, "УПОРЯДОЧИТЬ ПО З.Дата УБЫВ") != 1 {
					t.Fatal("stable order source differs")
				}
				optimized = strings.Replace(optimized, "УПОРЯДОЧИТЬ ПО З.Дата УБЫВ", "УПОРЯДОЧИТЬ ПО З.Дата УБЫВ, З.Номер УБЫВ, З.Ссылка УБЫВ", 1)
			}
		case "exists", "1":
		default:
			t.Fatal("unknown private prototype")
		}
		user, err := repo.GetByID(ctx, actorID)
		if err != nil {
			t.Fatal("private actor lookup failed")
		}
		user.Roles, err = repo.GetRolesForUser(ctx, actorID)
		if err != nil {
			t.Fatal("private role lookup failed")
		}
		runner := widget.New(reg, db)
		runner.User = user
		runner.CurrentUser = user.Login
		templateParams := map[string]any{}
		for k, v := range w.Params {
			templateParams[k] = v
		}
		params, err := scheduler.ResolveParamTemplates(templateParams, scheduler.NewConstantResolver(ctx, db, reg))
		if err != nil {
			t.Fatal("private parameter resolution failed")
		}
		params["ВсеДаты"] = false
		params["Филиал"] = nil
		before, after := *w, *w
		before.Limit = 0
		after.Limit = 0
		after.Query = optimized
		oldStart := time.Now()
		oldResult := runner.RunWithOptions(ctx, &before, widget.RunOptions{Params: params, Fresh: true})
		oldMS := float64(time.Since(oldStart).Microseconds()) / 1000
		newStart := time.Now()
		newResult := runner.RunWithOptions(ctx, &after, widget.RunOptions{Params: params, Fresh: true})
		newMS := float64(time.Since(newStart).Microseconds()) / 1000
		if oldResult.Error != "" || newResult.Error != "" || oldResult.AccessDenied || newResult.AccessDenied {
			t.Fatal("private prototype query failed: " + oldResult.Error + " / " + newResult.Error)
		}
		canonical := func(rows []map[string]any) []string {
			out := make([]string, len(rows))
			for i, row := range rows {
				raw, _ := json.Marshal(row)
				out[i] = string(raw)
			}
			sort.Strings(out)
			return out
		}
		oldSet, _ := json.Marshal(canonical(oldResult.Rows))
		newSet, _ := json.Marshal(canonical(newResult.Rows))
		oldCols, _ := json.Marshal(oldResult.Columns)
		newCols, _ := json.Marshal(newResult.Columns)
		take := func(rows []map[string]any) []map[string]any {
			if len(rows) > 20 {
				return rows[:20]
			}
			return rows
		}
		oldTop, _ := json.Marshal(take(oldResult.Rows))
		newTop, _ := json.Marshal(take(newResult.Rows))
		equality := map[string]any{"old_rows": len(oldResult.Rows), "new_rows": len(newResult.Rows), "all_rows_equal": bytes.Equal(oldSet, newSet), "columns_equal": bytes.Equal(oldCols, newCols), "ordered_top20_equal": bytes.Equal(oldTop, newTop)}

		urlCount := func(rows []map[string]any) int {
			n := 0
			for _, row := range rows {
				if value, ok := row["_row_url"].(string); ok && value != "" {
					n++
				}
			}
			return n
		}
		equality["old_navigation_rows"] = urlCount(oldResult.Rows)
		equality["new_navigation_rows"] = urlCount(newResult.Rows)
		equality["prototype"] = prototype
		equality["old_full_query_ms"] = oldMS
		equality["new_full_query_ms"] = newMS
		rowDate := func(row map[string]any) string {
			for k, v := range row {
				if strings.EqualFold(k, "Дата") {
					raw, _ := json.Marshal(v)
					return string(raw)
				}
			}
			return ""
		}
		dates := func(rows []map[string]any) []string {
			out := []string{}
			for _, row := range rows {
				out = append(out, rowDate(row))
			}
			return out
		}
		oldDates, _ := json.Marshal(dates(oldResult.Rows))
		newDates, _ := json.Marshal(dates(newResult.Rows))
		equality["ordered_date_keys_equal"] = bytes.Equal(oldDates, newDates)
		oldTopSet, _ := json.Marshal(canonical(take(oldResult.Rows)))
		newTopSet, _ := json.Marshal(canonical(take(newResult.Rows)))
		equality["top20_set_equal"] = bytes.Equal(oldTopSet, newTopSet)

		proof, _ := json.Marshal(equality)
		t.Logf("EQUIVALENCE %s", proof)
		if !bytes.Equal(oldSet, newSet) || !bytes.Equal(oldCols, newCols) || !bytes.Equal(oldDates, newDates) {
			t.Fatal("private prototype changed rows, columns or ordered dates; no benchmark")
		}
		if prototype == "group-all-stable" {
			// analysis 13.11: equal-date order was unspecified. Unique final
			// key is mandatory; preserve full values/URLs and ordered dates.
			repeat := runner.RunWithOptions(ctx, &after, widget.RunOptions{Params: params, Fresh: true})
			newAll, _ := json.Marshal(newResult.Rows)
			repeatAll, _ := json.Marshal(repeat.Rows)
			stable := repeat.Error == "" && !repeat.AccessDenied && bytes.Equal(newAll, repeatAll)
			t.Logf("STABLE_ORDER %t", stable)
			if !stable {
				t.Fatal("private prototype repeat order changed; no benchmark")
			}
		} else if !bytes.Equal(oldTop, newTop) {
			t.Fatal("private prototype changed strict top20; no benchmark")
		}
		httpOldQuery, httpNewQuery = w.Query, optimized
		w.Query = optimized
	}

	app, err := project.LoadConfig(stage)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Lang: "ru"}
	if app != nil {
		cfg.AppName = app.Name
		cfg.AppVersion = app.Version
		cfg.CompactLists = app.CompactLists()
	}
	s := New(reg, db, interp, repo, cfg, nil)
	defer s.Shutdown(context.Background())
	router := chi.NewRouter()
	s.Mount(router)
	handler := repo.Middleware(router)
	type metric struct {
		Phases  map[string]float64 `json:"phases_ms"`
		SQL     map[string]int64   `json:"sql_counts"`
		Rows    map[string]int     `json:"reference_row_counts"`
		Page    string             `json:"page"`
		Attempt int                `json:"attempt"`
		MS      float64            `json:"ms"`
		Queries int64              `json:"queries"`
		Bytes   int                `json:"bytes"`
		SHA     string             `json:"html_sha256"`
	}
	rows := []metric{}
	pages := []string{"", "Мессенджер"}
	if os.Getenv("PLATFORM_SHELL_HOME_ONLY") == "1" {
		pages = pages[:1]
	}
	variants := []struct{ name, query string }{{"", httpNewQuery}}
	if os.Getenv("PLATFORM_SHELL_COMPARE_HTTP") == "1" {
		if httpOldQuery == "" || httpNewQuery == "" {
			t.Fatal("HTTP comparison requires prototype")
		}
		variants = []struct{ name, query string }{{"before:", httpOldQuery}, {"after:", httpNewQuery}}
	}
	for _, variant := range variants {
		if variant.query != "" {
			reg.GetWidget("СписокЗадач").Query = variant.query
		}
		for _, sub := range pages {

			path := "/ui/app?home=1"
			if sub != "" {
				path += "&subsystem=" + url.QueryEscape(sub)
			}
			values := []float64{}
			for n := 0; n < 15; n++ {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.AddCookie(&http.Cookie{Name: "onebase_session", Value: token})
				req.Header.Set("Accept", "text/html")
				st := &serverTiming{start: time.Now(), spans: map[string]time.Duration{}}
				probe := &storage.ProfileProbeState{}
				req = req.WithContext(storage.WithProfileProbe(context.WithValue(req.Context(), serverTimingKey{}, st), probe))
				rec := httptest.NewRecorder()
				before := int64(0)
				started := time.Now()
				handler.ServeHTTP(rec, req)
				ms := float64(time.Since(started).Microseconds()) / 1000
				if rec.Code != 200 {
					t.Fatalf("synthetic GET status=%d", rec.Code)
				}
				if n == 0 {
					continue
				}
				body := rec.Body.Bytes()
				digest := sha256.Sum256(body)
				phases := map[string]float64{}
				st.mu.Lock()
				for name, duration := range st.spans {
					phases[name] = float64(duration.Microseconds()) / 1000
				}
				counts := st.counts
				st.mu.Unlock()
				rows = append(rows, metric{Phases: phases, SQL: probe.Snapshot(), Rows: counts, Page: variant.name + sub, Attempt: n, MS: ms, Queries: probe.Snapshot()["query"] + probe.Snapshot()["query_row"] + probe.Snapshot()["exec"] - before, Bytes: len(body), SHA: hex.EncodeToString(digest[:])})
				values = append(values, ms)
			}
			sort.Float64s(values)
			t.Logf("PROFILE page=%q p50_ms=%.3f p90_ms=%.3f", variant.name+sub, values[len(values)/2], values[int(float64(len(values))*.9)])
		}
	}
	raw, _ := json.Marshal(rows)
	t.Logf("METRICS %s", raw)
}
