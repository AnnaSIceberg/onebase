package ui

import (
	"net/http"
	"net/url"
	"strconv"
)

const listChunkParam = "_list_chunk"

func (s *Server) continuousLists(r *http.Request) bool {
	// The package presentation profile enables this for every non-operator,
	// including roles outside the compact controls and dashboard scope.
	return s.cfg.UIEnhancements && !s.legacyEntityUI(r)
}

func listChunkURL(r *http.Request, key string, next int) string {
	if next < 0 {
		return ""
	}
	q := cloneQuery(r.URL.Query())
	q.Set(listChunkParam, "1")
	q.Set(key, strconv.Itoa(next))
	return (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).String()
}

func listChunkNext(current, total int, backwards bool) int {
	if backwards {
		if current > 1 {
			return current - 1
		}
	} else if current < total {
		return current + 1
	}
	return -1
}

func (s *Server) listBatch(r *http.Request) (size, offset int) {
	if !s.continuousLists(r) {
		return 0, 0
	}
	size = s.store.GetListPageSizeDefault(r.Context(), 50)
	if r.URL.Query().Get(listChunkParam) == "1" {
		if n, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 31); err == nil && n > 0 {
			offset = int(n)
		}
	}
	return size, offset
}

func listBatchData(r *http.Request, size, offset int, rows []map[string]any, data map[string]any) map[string]any {
	data["Rows"] = rows
	if size == 0 {
		return data
	}
	data["StreamDir"] = "down"
	if len(rows) > size {
		data["Rows"] = rows[:size]
		data["StreamURL"] = listChunkURL(r, "offset", offset+size)
	}
	return data
}
