package ui

import (
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/storage"
)

func (s *Server) accountsList(w http.ResponseWriter, r *http.Request) {
	planName := chi.URLParam(r, "plan")
	if dec, err := url.PathUnescape(planName); err == nil {
		planName = dec
	}
	chart := s.reg.GetChartOfAccounts(planName)
	if chart == nil {
		http.Error(w, "plan of accounts not found: "+planName, 404)
		return
	}
	size, offset := s.listBatch(r)
	var window []storage.AccountListWindow
	if size > 0 {
		window = []storage.AccountListWindow{{Limit: size + 1, Offset: offset}}
	}
	rows, err := s.store.GetAccounts(r.Context(), chart.Name, window...)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render(w, r, "page-accounts", listBatchData(r, size, offset, rows, map[string]any{"Chart": chart}))
}

func (s *Server) accountRegMovements(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if dec, err := url.PathUnescape(name); err == nil {
		name = dec
	}
	ar := s.reg.GetAccountRegister(name)
	if ar == nil {
		http.Error(w, "account register not found: "+name, 404)
		return
	}
	size, offset := s.listBatch(r)
	var window []storage.AccountListWindow
	if size > 0 {
		window = []storage.AccountListWindow{{Limit: size + 1, Offset: offset}}
	}
	rows, err := s.store.GetAccountMovements(r.Context(), ar.Name, ar, window...)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.resolveAccountRows(r.Context(), rows, ar)
	s.render(w, r, "page-accountreg-movements", listBatchData(r, size, offset, rows, map[string]any{"Register": ar}))
}

func (s *Server) accountRegBalances(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if dec, err := url.PathUnescape(name); err == nil {
		name = dec
	}
	ar := s.reg.GetAccountRegister(name)
	if ar == nil {
		http.Error(w, "account register not found: "+name, 404)
		return
	}

	// resolve associated chart of accounts
	chart := s.reg.GetChartOfAccounts(ar.Accounts)

	asOf := time.Now()
	if d := r.URL.Query().Get("date"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			asOf = t.Add(24*time.Hour - time.Second)
		}
	}

	planName := ""
	if chart != nil {
		planName = chart.Name
	}

	size, offset := s.listBatch(r)
	var window []storage.AccountListWindow
	if size > 0 {
		window = []storage.AccountListWindow{{Limit: size + 1, Offset: offset}}
	}
	rows, err := s.store.AccountBalances(r.Context(), ar.Name, planName, asOf, ar.Resources, ar.Subconto, window...)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	s.resolveAccountRows(r.Context(), rows, ar)

	// Sort rows by code
	sort.Slice(rows, func(i, j int) bool {
		ci, _ := rows[i]["code"].(string)
		cj, _ := rows[j]["code"].(string)
		return ci < cj
	})

	s.render(w, r, "page-accountreg-balances", listBatchData(r, size, offset, rows, map[string]any{
		"Register": ar,
		"Chart":    chart,
		"AsOf":     asOf.Format("2006-01-02"),
	}))
}
