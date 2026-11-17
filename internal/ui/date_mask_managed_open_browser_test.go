package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	orun "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// An autofocused empty date must not abort OnOpen before operator tabs are
// hidden. Otherwise their intrinsic width moves the fixed sidebar below them.
// Exercise the real renderer, date asset, managed FormData and server handler.
func TestDateMaskBrowserManagedOpeningKeepsSidebarBesideTabs(t *testing.T) {
	db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "mask-open.db"))
	if err != nil {
		t.Fatal(err)
	}
	form := &metadata.FormModule{
		Name: "ФормаОбъекта", Kind: "object", EntityName: "MaskOpenLayout",
		LayoutKind: metadata.FormLayoutManaged, Density: "compact", Autofocus: "first_empty",
		Handlers: map[metadata.FormEventType]string{metadata.FormEventOnOpen: "ПриОткрытии"},
		ProgramAST: mustParse(t, `Процедура ПриОткрытии()
		Объект.Hide = Истина;
		КонецПроцедуры`),
	}
	noFrame := false
	pages := &metadata.FormElement{Kind: metadata.FormElementPages, Name: "Pages", FitWindow: true, TabTitlesBold: true}
	for i, title := range []string{"Заказчик", "Изделие", "Дополнение", "Действия", "Деньги", "Отзывы, обзвоны, доп. инф.", "Работы и детали", "Зар. плата мастеров", "Файлы", "Комментарии", "МРМ", "Звонки"} {
		page := &metadata.FormElement{Kind: metadata.FormElementPage, Name: "Page" + strconv.Itoa(i), TitleMap: map[string]string{"ru": title}}
		if i >= 3 {
			page.HiddenWhen = "Hide"
		}
		if i == 0 {
			page.Children = []*metadata.FormElement{{Kind: metadata.FormElementGroupBox, Name: "Content", Width: 1023, Frame: &noFrame,
				Children: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "Name", DataPath: "Объект.Наименование"}}}}
		}
		pages.Children = append(pages.Children, page)
	}
	form.Elements = []*metadata.FormElement{
		{Kind: metadata.FormElementDatePicker, Name: "Day", DataPath: "Объект.Day", Width: 125},
		{Kind: metadata.FormElementGroupBox, Name: "Main", Orientation: "horizontal", Frame: &noFrame, Children: []*metadata.FormElement{
			pages,
			{Kind: metadata.FormElementGroupBox, Name: "Sidebar", Width: 320, FitWindow: true, Children: []*metadata.FormElement{
				{Kind: metadata.FormElementButton, Name: "Accept", TitleMap: map[string]string{"ru": "Принять заявку"}},
			}},
		}},
	}
	entity := &metadata.Entity{Name: "MaskOpenLayout", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "Day", Type: metadata.FieldTypeDate}, {Name: "Hide", Type: metadata.FieldTypeBool}}, Forms: []*metadata.FormModule{form}}
	if err := db.Migrate(t.Context(), []*metadata.Entity{entity}); err != nil {
		t.Fatal(err)
	}
	reg := orun.NewRegistry()
	reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{entity}})
	interp := interpreter.New()
	interp.LookupProc = reg.GetModuleProc
	s := New(reg, db, interp, auth.NewRepo(db), Config{ShortDateInput: true}, nil)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithOpenAccess(r.Context())))
		})
	})
	s.MountStatic(router)
	s.Mount(router)
	var openEvents atomic.Int32
	var nonemptyDate atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/layout-probe" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><body style="margin:0;overflow:hidden"><iframe src="/ui/catalog/maskopenlayout/new" style="border:0;width:1440px;height:648px"></iframe><script>
const started=performance.now();
function sample(){
 const doc=document.querySelector('iframe').contentDocument;
 const tabs=doc&&doc.querySelector('[data-tabs="Pages"]'), side=doc&&doc.querySelector('[data-ob-el="Sidebar"]');
 if(!tabs||!side){setTimeout(sample,20);return;}
 const visible=Array.from(tabs.querySelectorAll(':scope>.managed-tab-headers>button')).filter(e=>e.getClientRects().length);
 if(visible.length!==3 && performance.now()-started<3000){setTimeout(sample,20);return;}
 requestAnimationFrame(()=>requestAnimationFrame(()=>{
  const a=tabs.getBoundingClientRect(),b=side.getBoundingClientRect();
  const out=document.createElement('pre');out.id='list-mobile-measure';out.style.display='none';
  out.textContent=JSON.stringify({visibleTabs:visible.length,topDelta:Math.abs(a.top-b.top),sidebarLeft:b.left,tabsRight:a.right,stacked:tabs.getAttribute('data-ob-fit-stacked')==='1'});
  document.body.appendChild(out);
 }));
}
sample();</script>`))
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/form-event") {
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("_event") == "ПриОткрытии" {
				openEvents.Add(1)
				if r.Form.Get("Day") != "" {
					nonemptyDate.Store(true)
				}
			}
		}
		router.ServeHTTP(w, r)
	}))
	t.Cleanup(func() { s.CloseEventStreams(); server.Close(); s.Close(); _ = db.Close() })
	browser := newMobileLayoutBrowserSize(t, managedLayoutTestBrowser(t), 1440, 750)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	raw, err := browser.measure(ctx, server.URL+"/layout-probe")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		VisibleTabs            int
		TopDelta               float64
		SidebarLeft, TabsRight float64
		Stacked                bool
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if openEvents.Load() != 1 || nonemptyDate.Load() {
		t.Errorf("opening snapshot must contain the native empty date: events=%d nonempty=%t", openEvents.Load(), nonemptyDate.Load())
	}
	if got.VisibleTabs != 3 || got.Stacked || got.TopDelta > 1 || got.SidebarLeft < got.TabsRight {
		t.Errorf("OnOpen did not keep operator sidebar beside tabs: %s", raw)
	}
}
