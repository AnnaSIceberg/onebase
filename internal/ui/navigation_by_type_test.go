package ui

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/auth"
)

func TestTypedMenuKeepsReportFoldersAndSortsEntries(t *testing.T) {
	s := &Server{cfg: Config{Lang: "ru"}}
	r := httptest.NewRequest("GET", "/ui/", nil)
	input := []navGroup{{Kind: "Операции", Nodes: []navTreeNode{
		{TargetKind: "document", Label: "Яблоки", URL: "/document/1"},
		{TargetKind: "document", TargetAction: "create", Label: "Создать: накладная", URL: "/document/new"},
		{TargetKind: "catalog", Label: "Мастера", URL: "/catalog/master"},
		{TargetKind: "processor", Label: "Распределение", URL: "/processor/example"},
		{TargetKind: "journal", Label: "Журнал", URL: "/journal/example"},
		{TargetKind: "inforeg", Label: "Остатки", URL: "/inforeg/example"},
		{Label: "KPI", Nodes: []navTreeNode{
			{TargetKind: "report", Label: "KPI краткий", URL: "/report/kpi"},
			{TargetKind: "processor", Label: "KPI сводный", URL: "/processor/kpi"},
			{TargetKind: "document", Label: "Аптеки", URL: "/document/2"},
		}},
	}}, {Kind: "Отчеты", Nodes: []navTreeNode{
		{TargetKind: "page", Label: "Отчёт 400", URL: "/page/400"},
		{TargetKind: "processor", Label: "Финансовый отчёт", URL: "/processor/finance"},
		{TargetKind: "document", Label: "Реклама", URL: "/document/3"},
	}}}
	out := s.navigationByType(r, "", input)
	var titles []string
	for _, g := range out {
		if !g.Gradient || g.Icon == "" {
			t.Fatalf("у раздела %q нет значка или фона", g.Kind)
		}
		titles = append(titles, g.Kind)
	}
	want := []string{"Документы", "Справочники", "Отчёты", "Обработки", "Журналы", "Регистры"}
	if !reflect.DeepEqual(titles, want) {
		t.Fatalf("разделы %v, ожидалось %v", titles, want)
	}
	var labels []string
	for _, n := range out[0].Nodes {
		labels = append(labels, n.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Создать: накладная", "Аптеки", "Реклама", "Яблоки"}) {
		t.Fatalf("документы: %v", labels)
	}
	report := out[2].Nodes
	if len(report) != 3 || report[0].Label != "KPI" || len(report[0].Nodes) != 2 || report[0].Nodes[0].URL != "/report/kpi" || report[0].Nodes[1].URL != "/processor/kpi" {
		t.Fatalf("исходная папка KPI: %#v", report)
	}
	if report[1].URL != "/page/400" || report[2].URL != "/processor/finance" {
		t.Fatalf("прикладные команды не остались в Отчётах: %#v", report)
	}
}

func TestTypedMenuOperatorKeepsExistingMenu(t *testing.T) {
	f := newNavigationHTTPFixture(t)
	f.server.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
	f.server.cfg.NavigationByType = true
	rec := f.request(t, "GET", "/ui/", "admin", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `data-nav-id="cfg:menu-type:`) {
		t.Fatal("новая группировка отсутствует в рабочем меню")
	}
	for _, role := range f.server.cfg.UIExceptRoles {
		r := httptest.NewRequest("GET", "/ui/", nil)
		r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{IsAdmin: true, Roles: []*auth.Role{{Name: role}}}))
		groups := f.server.buildNav(r, "")
		for _, g := range groups {
			if strings.HasPrefix(g.ID, "cfg:menu-type:") {
				t.Fatalf("новая иерархия применена к %s", role)
			}
			for _, item := range g.Items {
				if item.CreateURL != "" {
					t.Fatalf("у %s появилась новая кнопка +", role)
				}
			}
		}
	}
}

func TestHideHomeNavigationKeepsDepartmentAndOperatorMenus(t *testing.T) {
	f := newNavigationHTTPFixture(t)
	f.server.cfg.NavigationByType = true
	f.server.cfg.HideHomeNavigation = true
	f.server.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
	rec := f.request(t, "GET", "/ui/app?home=1", "admin", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `<aside id="ob-nav" hidden>`) ||
		strings.Contains(body, `data-nav-id="cfg:menu-type:`) || !strings.Contains(body, `id="ob-tabhome"`) {
		t.Fatal("главная должна оставить рабочую область без меню объектов")
	}
	rec = f.request(t, "GET", "/ui/app?home=1&subsystem=School", "admin", nil)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), `<aside id="ob-nav" hidden`) {
		t.Fatal("раздел выдал скрытое меню")
	}
	for _, role := range f.server.cfg.UIExceptRoles {
		r := httptest.NewRequest("GET", "/ui/app?home=1", nil)
		r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{IsAdmin: true, Roles: []*auth.Role{{Name: role}}}))
		page := httptest.NewRecorder()
		f.plain.ServeHTTP(page, r)
		if page.Code != 200 || strings.Contains(page.Body.String(), `<aside id="ob-nav" hidden`) {
			t.Fatalf("главная у роли %s изменилась", role)
		}
	}
}
