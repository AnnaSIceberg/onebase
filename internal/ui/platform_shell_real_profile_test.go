package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"gopkg.in/yaml.v3"
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

	oldWidgets := []*metadata.Widget{}
	for _, w := range proj.Widgets {
		if w.Name == "СписокЗадач" || w.Name == "БыстрыеДействия" || w.Name == "ОрдераНеобработанные" || w.Name == "ОрдераОбработанные" || w.Name == "ОрдераПеремещения" {
			oldWidgets = append(oldWidgets, w)
		}
	}
	reg.LoadWidgets(oldWidgets)

	proj.HomePage.Rows = []metadata.HomePageRow{{Widgets: []string{"БыстрыеДействия", "СписокЗадач"}}}
	reg.LoadHomePage(proj.HomePage)
	interp := interpreter.New()
	interp.LookupProc = reg.GetModuleProc
	repo := auth.NewRepo(db)
	actorID := os.Getenv("PLATFORM_REAL_ACTOR_ID")
	if actorID == "" {
		t.Fatal("own test actor required")
	}
	token, err := repo.CreateSession(ctx, actorID, auth.SessionMeta{Kind: auth.SessionKindEnterprise})
	if err != nil {
		t.Fatal("private copy session create failed")
	}
	defer repo.DeleteSession(context.Background(), token)
	appRaw, err := os.ReadFile(stage + "/config/app.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Lang: "ru"}
	var app struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(appRaw, &app); err != nil {
		t.Fatal("display config parse failed")
	}
	cfg.AppName = app.Name
	cfg.AppVersion = app.Version
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
			req = req.WithContext(context.WithValue(req.Context(), serverTimingKey{}, st))
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
			counts := map[string]int{}
			st.mu.Unlock()
			rows = append(rows, metric{Phases: phases, SQL: map[string]int64{}, Rows: counts, Page: sub, Attempt: n, MS: ms, Queries: -before, Bytes: len(body), SHA: hex.EncodeToString(digest[:])})
			values = append(values, ms)
		}
		sort.Float64s(values)
		t.Logf("PROFILE page=%q p50_ms=%.3f p90_ms=%.3f", sub, values[len(values)/2], values[int(float64(len(values))*.9)])
	}
	raw, _ := json.Marshal(rows)
	t.Logf("METRICS %s", raw)
}
