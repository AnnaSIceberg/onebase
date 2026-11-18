package ui

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/xuri/excelize/v2"
)

func TestOrdinaryReportXLSXUsedRangeMatrix(t *testing.T) {
	const detailCount = 22
	// ENG02: one header, two group rows containing their totals, all details,
	// and one grand total. Legacy totals.subtotals must not add two footers.
	const wantRows = 1 + 2 + detailCount + 1
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		e := &metadata.Entity{Name: "ListDimensionSource", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Team", Type: metadata.FieldTypeString}, {Name: "Amount", Type: metadata.FieldTypeNumber}}}
		if err := db.Migrate(ctx, []*metadata.Entity{e}); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= detailCount; i++ {
			team := "A"
			if i > 11 {
				team = "B"
			}
			if err := db.Upsert(ctx, e.Name, uuid.New(), map[string]any{"Team": team, "Amount": i}, e); err != nil {
				t.Fatal(err)
			}
		}
		rep := &reportpkg.Report{Name: "OrdinaryDimension", Query: `ВЫБРАТЬ Team, Amount ИЗ Справочник.ListDimensionSource`, Composition: &reportpkg.Composition{Groupings: []string{"team"}, Measures: []reportpkg.Measure{{Field: "amount", Agg: "sum"}}, Detail: true, Totals: reportpkg.Totals{Grand: true, Subtotals: true}}}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{e}, Reports: []*reportpkg.Report{rep}})
		s := &Server{store: db, reg: reg, interp: interpreter.New(), messages: NewMessageStore(), lockMgr: runtime.NewLockManager(), ops: newOperationLimiter()}
		t.Cleanup(s.Close)
		router := chi.NewRouter()
		s.Mount(router)
		check := func(t *testing.T, data []byte, mode string) {
			t.Helper()
			z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range z.File {
				if p.Name != "xl/worksheets/sheet1.xml" {
					continue
				}
				r, err := p.Open()
				if err != nil {
					t.Fatal(err)
				}
				raw, err := io.ReadAll(r)
				if closeErr := r.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
				if err != nil {
					t.Fatal(err)
				}
				var sh struct {
					Dimension struct {
						Ref string `xml:"ref,attr"`
					} `xml:"dimension"`
					Rows []struct {
						R int `xml:"r,attr"`
					} `xml:"sheetData>row"`
				}
				if err := xml.Unmarshal(raw, &sh); err != nil {
					t.Fatal(err)
				}
				wantDimension := fmt.Sprintf("A1:B%d", wantRows)
				if sh.Dimension.Ref != wantDimension || len(sh.Rows) != wantRows {
					t.Fatalf("ordinary XLSX dimension=%s rows%d, want%s/%d", sh.Dimension.Ref, len(sh.Rows), wantDimension, wantRows)
				}
				found = true
			}
			if !found {
				t.Fatal("worksheet missing")
			}
			f, err := excelize.OpenReader(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			rows, err := f.GetRows("Лист1")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != wantRows || len(rows[wantRows-1]) < 2 || rows[wantRows-1][0] != "Итого" || rows[wantRows-1][1] != "253" {
				t.Fatal("ordinary report rows/totals changed")
			}
			wantSubtotals := map[string]int{"A": 66, "B": 187}
			seenGroups := make(map[string]bool)
			seenDetails := make(map[int]bool)
			detailSubtotals := make(map[string]int)
			group := ""
			for index, row := range rows[1 : wantRows-1] {
				if len(row) != 2 {
					t.Fatalf("row%d lost cells: %v", index+2, row)
				}
				label := strings.TrimSpace(row[0])
				value, err := strconv.Atoi(row[1])
				if err != nil {
					t.Fatalf("row%d invalid amount %q: %v", index+2, row[1], err)
				}
				if label != "" {
					want, ok := wantSubtotals[label]
					if !ok || seenGroups[label] || value != want {
						t.Fatalf("row%d duplicate footer or wrong group subtotal: %v", index+2, row)
					}
					seenGroups[label] = true
					group = label
					continue
				}
				if group == "" || value < 1 || value > detailCount || seenDetails[value] || (group == "A" && value > 11) || (group == "B" && value <= 11) {
					t.Fatalf("row%d missing, duplicate or misplaced detail in group %q: %v", index+2, group, row)
				}
				seenDetails[value] = true
				detailSubtotals[group] += value
			}
			if len(seenGroups) != 2 || len(seenDetails) != detailCount || detailSubtotals["A"] != 66 || detailSubtotals["B"] != 187 {
				t.Fatalf("ordinary XLSX lost groups/details: groups%v details%d sums%v", seenGroups, len(seenDetails), detailSubtotals)
			}
			if dir := os.Getenv("ONEBASE_TEST_XLSX_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil { // #nosec G703 -- Opt-in test artifact directory supplied by the trusted test runner.
					t.Fatal(err)
				}
				name := "ordinary-" + db.Dialect().Name() + "-" + mode + ".xlsx"
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil { // #nosec G703 -- Trusted runner directory; the filename uses only fixed test fixture labels.
					t.Fatal(err)
				}
			}
		}
		t.Run("direct", func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/ui/report/OrdinaryDimension/excel", nil))
			if w.Code != 200 {
				t.Fatalf("export HTTP%d %s", w.Code, w.Body.String())
			}
			check(t, w.Body.Bytes(), "direct")
		})
		t.Run("background", func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/ui/report/OrdinaryDimension/export/excel", nil))
			if w.Code != 303 {
				t.Fatalf("queueHTTP%d", w.Code)
			}
			jobID := strings.TrimPrefix(w.Header().Get("Location"), "/ui/export-jobs/")
			deadline := time.Now().Add(10 * time.Second)
			for {
				j, ok := s.exportJobStore().get(jobID)
				if ok && j.Status == exportJobDone {
					check(t, j.Data, "background")
					return
				}
				if ok && j.Status == exportJobError {
					t.Fatal(j.Error)
				}
				if time.Now().After(deadline) {
					t.Fatal("export timeout")
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	})
}
