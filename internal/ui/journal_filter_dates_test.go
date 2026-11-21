package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestJournalCalendarDateRangeHTTP(t *testing.T) {
	savedLocation := time.Local
	time.Local = time.FixedZone("MSK", 3*60*60)
	t.Cleanup(func() { time.Local = savedLocation })
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		doc := &metadata.Entity{Name: "JournalDayDocument", Kind: metadata.KindDocument, Fields: []metadata.Field{
			{Name: "Номер", Type: metadata.FieldTypeString},
			{Name: "Дата", Type: metadata.FieldTypeDate},
		}}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{doc})
		s.cfg.UIEnhancements = true
		s.reg.LoadJournals([]*metadata.Journal{{
			Name: "JournalDay", Documents: []string{doc.Name},
			Columns: []metadata.JournalColumn{{Field: "Дата"}, {Field: "Номер"}},
			Filters: []metadata.JournalFilter{{Field: "Дата", Type: "date_range"}},
		}})

		first := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.Local)
		late := first.Add(23*time.Hour + 30*time.Minute)
		for _, row := range []struct {
			number string
			moment time.Time
		}{
			{"BEFORE_SELECTED_DAY", first.Add(-time.Second)},
			{"MIDNIGHT_SELECTED_DAY", first},
			{"LATE_SELECTED_DAY", late},
			{"MIDNIGHT_FOLLOWING_DAY", first.AddDate(0, 0, 1)},
		} {
			require.NoError(t, db.Upsert(t.Context(), doc.Name, uuid.New(), map[string]any{
				"Номер": row.number, "Дата": row.moment,
			}, doc))
		}
		router := chi.NewRouter()
		s.Mount(router)
		filter := url.Values{"f.Дата.from": {"2026-09-30"}, "f.Дата.to": {"2026-09-30"}}
		request := func(path string) *httptest.ResponseRecorder {
			t.Helper()
			rec := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/ui/journal/JournalDay"+path+"?"+filter.Encode(), nil)
			router.ServeHTTP(rec, r)
			require.Equal(t, http.StatusOK, rec.Code, "journal response for %s", r.URL.String())
			return rec
		}

		body := request("").Body.String()
		require.Contains(t, body, "MIDNIGHT_SELECTED_DAY")
		require.Contains(t, body, "LATE_SELECTED_DAY")
		require.NotContains(t, body, "BEFORE_SELECTED_DAY")
		require.NotContains(t, body, "MIDNIGHT_FOLLOWING_DAY")

		filter.Set(listChunkParam, "1")
		filter.Set("offset", "1")
		body = request("").Body.String()
		require.Contains(t, body, "LATE_SELECTED_DAY")
		require.NotContains(t, body, "MIDNIGHT_FOLLOWING_DAY")
		require.NotContains(t, body, "MIDNIGHT_SELECTED_DAY")
		filter.Del("offset")
		filter.Del(listChunkParam)

		book, err := excelize.OpenReader(bytes.NewReader(request("/excel").Body.Bytes()))
		require.NoError(t, err)
		defer func() { require.NoError(t, book.Close()) }()
		rows, err := book.GetRows(book.GetSheetName(0))
		require.NoError(t, err)
		require.Len(t, rows, 3)
		require.ElementsMatch(t, []string{"MIDNIGHT_SELECTED_DAY", "LATE_SELECTED_DAY"}, []string{rows[1][2], rows[2][2]})

		// A date-time supplied explicitly retains its inclusive boundary.
		filter.Set("f.Дата.to", late.UTC().Format(time.RFC3339))
		body = request("").Body.String()
		require.Contains(t, body, "LATE_SELECTED_DAY")
		require.NotContains(t, body, "MIDNIGHT_FOLLOWING_DAY")
	})
}
