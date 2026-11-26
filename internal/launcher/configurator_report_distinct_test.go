package launcher

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/configdb"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/ui"
	"github.com/xuri/excelize/v2"
)

func TestConfiguratorSaveReport_KeepsDistinctMeasureKeys(t *testing.T) {
	const path = "reports/distinct.yaml"
	const original = `name: Distinct
title: Original
query: SELECT 'region' AS Region, 1 AS Requests, 'request' AS CalcKey UNION ALL SELECT 'region', 1, 'request'
composition:
  groupings: [Region]
  distinct_measure_keys: {Requests: CalcKey}
  internal_fields: [CalcKey]
  measures: [{field: Requests, agg: sum}]
  totals: {grand: true, subtotals: true}
`
	for _, source := range []string{"file", "database"} {
		t.Run(source, func(t *testing.T) {
			ctx := context.Background()
			h, dir := newFileBaseHandler(t)
			h.runner = NewRunner()
			var readReport func() ([]byte, error)
			if source == "file" {
				filename := writeCfgFile(t, dir, "reports", "distinct.yaml", original)
				readReport = func() ([]byte, error) { return os.ReadFile(filename) }
			} else {
				base, err := h.store.Get("test")
				if err != nil {
					t.Fatal(err)
				}
				base.ConfigSource, base.DBType = "database", "sqlite"
				base.DBPath = filepath.Join(t.TempDir(), "config.db")
				if err := h.store.Update(base); err != nil {
					t.Fatal(err)
				}
				db, err := OpenDB(ctx, base)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { db.Close() })
				repository := configdb.New(db)
				if err := repository.EnsureSchema(ctx); err != nil {
					t.Fatal(err)
				}
				if err := repository.SaveFiles(ctx, []configdb.ConfigFile{
					{Path: "config/app.yaml", Content: []byte("name: Test\n")},
					{Path: path, Content: []byte(original)},
				}, configdb.VersionOptions{Message: "seed distinct report"}); err != nil {
					t.Fatal(err)
				}
				readReport = func() ([]byte, error) {
					content, found, err := repository.ReadFile(ctx, path)
					if !found && err == nil {
						t.Fatal("saved report is absent from database")
					}
					return content, err
				}
			}

			router := chi.NewRouter()
			router.Get("/bases/{id}/configurator", h.configuratorPage)
			router.Post("/bases/{id}/configurator/report", h.configuratorSaveReport)
			page := httptest.NewRecorder()
			router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/bases/test/configurator?tab=tree", nil))
			if page.Code != http.StatusOK {
				t.Fatalf("configurator returned %d", page.Code)
			}
			form := browserSubmit(t, page.Body.String(), "/configurator/report")
			if form.Get("comp.present") == "" {
				t.Fatal("native composition form missing")
			}
			form.Set("title", "Edited")
			form.Set("comp.distinct_measure_keys", "Requests: Region")
			form.Set("comp.internal_fields", "Region")
			req := httptest.NewRequest(http.MethodPost, "/bases/test/configurator/report", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Onebase-Ajax", "1")
			saved := httptest.NewRecorder()
			router.ServeHTTP(saved, req)
			if ok, reason := cfgResponse(t, saved); !ok {
				t.Fatal(reason)
			}
			data, err := readReport()
			if err != nil {
				t.Fatal(err)
			}
			rep, err := report.ParseBytes(data)
			if err != nil {
				t.Fatal(err)
			}
			if rep.Title != "Edited" || rep.Composition == nil ||
				!reflect.DeepEqual(rep.Composition.DistinctMeasureKeys, map[string]string{"Requests": "CalcKey"}) ||
				!reflect.DeepEqual(rep.Composition.InternalFields, []string{"CalcKey"}) {
				t.Fatalf("saved report lost trusted composition: %s", data)
			}
			checkDistinctReportOutputs(t, ctx, rep)
		})
	}
}

func checkDistinctReportOutputs(t *testing.T, ctx context.Context, rep *report.Report) {
	t.Helper()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "result.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureSettingsSchema(ctx); err != nil {
		t.Fatal(err)
	}
	registry := runtime.NewRegistry()
	registry.Load(runtime.LoadOptions{Reports: []*report.Report{rep}})
	server := ui.New(registry, db, interpreter.New(), nil, ui.Config{}, nil)
	t.Cleanup(server.Close)
	router := chi.NewRouter()
	server.Mount(router)
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, resp.Code)
		}
		return resp
	}
	screen := get("/ui/report/Distinct?__run=1").Body.String()
	if !strings.Contains(screen, ">1</td>") || strings.Contains(strings.ToLower(screen), "calckey") {
		t.Fatal("saved report returned a wrong count or offered the internal settings field")
	}
	book, err := excelize.OpenReader(bytes.NewReader(get("/ui/report/Distinct/excel").Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { book.Close() })
	rows, err := book.GetRows(book.GetSheetName(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 3 || len(rows[len(rows)-1]) != 2 || rows[len(rows)-1][1] != "1" {
		t.Fatalf("saved report has wrong XLSX total: %v", rows)
	}
}
