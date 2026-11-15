import pathlib
W=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/report-source-filters-b23-runtime')
(W/'internal/report/compose/source_filters_test.go').write_text('''package compose
import("testing";"reflect";"strings";"github.com/google/uuid";"github.com/shopspring/decimal";"github.com/ivantit66/onebase/internal/report")
func TestApplySourceFilters(t *testing.T){
 id:=uuid.New();other:=uuid.New();rows:=[]Row{{"Label":"ALPHA","Amount":decimal.NewFromInt(1),"Ref":id},{"Label":"BETA","Amount":decimal.NewFromInt(2),"Ref":other},{"Label":"EMPTY","Amount":nil,"Ref":nil}}
 before:=make([]Row,len(rows));for i,r:=range rows{before[i]=make(Row);for k,v:=range r{before[i][k]=v}}
 for _,tc:=range []struct{name string;f report.Filter;count int;deny bool}{
 {"alpha",report.Filter{Field:"label",Op:"eq",Value:"ALPHA"},1,false},
 {"numeric",report.Filter{Field:"Amount",Op:"ge",Value:"2"},1,false},
 {"reference",report.Filter{Field:"Ref",Op:"eq",Value:strings.ToUpper(id.String())},1,false},
 {"reference-ne",report.Filter{Field:"Ref",Op:"ne",Value:id.String()},2,false},
 {"empty-reference",report.Filter{Field:"Ref",Op:"eq",Value:""},1,false},
 {"contains",report.Filter{Field:"Label",Op:"contains",Value:"alp"},1,false},
 {"empty-result",report.Filter{Field:"Label",Op:"eq",Value:"NONE"},0,false},
 {"unknown-field",report.Filter{Field:"Unknown",Op:"eq",Value:"A"},0,true},
 {"unknown-op",report.Filter{Field:"Label",Op:"bogus",Value:"A"},0,true},
 {"ref-display-label",report.Filter{Field:"Ref",Op:"eq",Value:"Display A"},0,true},
 {"ref-contains",report.Filter{Field:"Ref",Op:"contains",Value:"A"},0,true},
 }{t.Run(tc.name,func(t *testing.T){got,err:=ApplySourceFilters(rows,[]report.Filter{tc.f},[]string{"Label","Amount","Ref"});if(err!=nil)!=tc.deny{t.Fatalf("deny=%v err=%v",tc.deny,err)};if !tc.deny&&len(got)!=tc.count{t.Fatalf("rows%d want%d",len(got),tc.count)};if !reflect.DeepEqual(rows,before){t.Fatal("source rows or reference types mutated")}})}
}
''',encoding='utf-8')
(W/'internal/ui/report_source_filters_test.go').write_text('''package ui
import("bytes";"context";"fmt";"net/http";"net/url";"strings";"testing";"github.com/google/uuid";"github.com/ivantit66/onebase/internal/auth";"github.com/ivantit66/onebase/internal/dbtest";"github.com/ivantit66/onebase/internal/dsl/ast";"github.com/ivantit66/onebase/internal/dsl/interpreter";"github.com/ivantit66/onebase/internal/metadata";reportpkg "github.com/ivantit66/onebase/internal/report";"github.com/ivantit66/onebase/internal/runtime";"github.com/ivantit66/onebase/internal/storage";"github.com/xuri/excelize/v2")
func TestReportResultSourceFiltersHTTPMatrix(t *testing.T){dbtest.ForEachDialect(t,func(t *testing.T,db *storage.DB){
 ctx:=context.Background();ref:=&metadata.Entity{Name:"SourceFilterRef",Kind:metadata.KindCatalog,Fields:[]metadata.Field{{Name:"Name",Type:metadata.FieldTypeString}}};entity:=&metadata.Entity{Name:"SourceFilterRow",Kind:metadata.KindCatalog,Fields:[]metadata.Field{{Name:"Метка",Type:metadata.FieldTypeString},{Name:"Amount",Type:metadata.FieldTypeNumber},{Name:"RefValue",Type:"reference:SourceFilterRef",RefEntity:ref.Name},{Name:"Owner",Type:metadata.FieldTypeString},{Name:"Secret",Type:metadata.FieldTypeString}}}
 if err:=db.Migrate(ctx,[]*metadata.Entity{ref,entity});err!=nil{t.Fatal(err)};if err:=db.EnsureSettingsSchema(ctx);err!=nil{t.Fatal(err)}
 a,b:=uuid.New(),uuid.New();for _,id:=range []uuid.UUID{a,b}{if err:=db.Upsert(ctx,ref.Name,id,map[string]any{"Name":"Display label"},ref);err!=nil{t.Fatal(err)}}
 for i,label:=range []string{"ALPHA","BETA","FOREIGN"}{id:=a;if i==1{id=b};owner:="source-runner";if i==2{owner="other"};if err:=db.Upsert(ctx,entity.Name,uuid.New(),map[string]any{"Метка":label,"Amount":i+1,"RefValue":id.String(),"Owner":owner,"Secret":"RAW_SECRET"},entity);err!=nil{t.Fatal(err)}}
 cases:=[]string{"unfiltered","get-alpha","post-alpha","case-field","numeric","reference","masked","empty","variant-filter","variant-conflict","preset-alpha","default-preset-alpha","unknown-field","unsupported-op","reference-label","queryless-filter","queryless-no-filter","malformed-settings","query-budget","variant-budget","no-composition","settings-too-large","source-read-denied"}
 for index,name:=range cases{t.Run(name,func(t *testing.T){
 rep:=&reportpkg.Report{Name:fmt.Sprintf("SourceFilters%d",index),Query:`ВЫБРАТЬ Метка, Amount, RefValue, Secret ИЗ Справочник.SourceFilterRow`,ResultProc:"Build",Composition:&reportpkg.Composition{Detail:true,Measures:[]reportpkg.Measure{{Field:"Метка",Agg:"count"}}},Variants:[]reportpkg.ReportVariant{{Name:"beta",Filters:[]reportpkg.VariantFilter{{Field:"Метка",Op:"eq",Value:"BETA"}}}}}
 prog:=mustParse(t,`Функция Build() Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="COUNT:"+Строка(Результат.Количество());Н=2;Для Каждого Стр Из Результат Цикл Д.Ячейка(Н,1).Текст=Стр.Метка;Д.Ячейка(Н,2).Текст=Строка(Стр.RefValue);Н=Н+1;КонецЦикла;Возврат Д;КонецФункции`)
 reg:=runtime.NewRegistry();reg.Load(runtime.LoadOptions{Entities:[]*metadata.Entity{ref,entity},Reports:[]*reportpkg.Report{rep},Programs:map[string]*ast.Program{rep.Name:prog}});interp:=interpreter.New();interp.LookupProc=reg.GetModuleProc;s:=&Server{reg:reg,store:db,interp:interp,messages:NewMessageStore(),lockMgr:runtime.NewLockManager(),ops:newOperationLimiter()};s.entitySvc=s.newEntityService(nil);t.Cleanup(s.Close)
 role:=&auth.Role{Name:"SourceFilterRunner",Permissions:auth.Permission{Reports:map[string][]string{rep.Name:{"run"}},Catalogs:map[string][]string{entity.Name:{"read"},ref.Name:{"read"}},RowAccess:auth.RowAccess{Catalogs:map[string]auth.RowPolicies{entity.Name:{"read":{Field:"Owner",Op:"eq",Value:auth.RowValue{User:"login"}}}}},FieldAccess:auth.FieldAccess{Catalogs:map[string]auth.FieldPolicies{entity.Name:{"Secret":{Read:"mask_all"}}}}}}
 actor:=&auth.User{ID:uuid.NewString(),Login:"source-runner",Roles:[]*auth.Role{role}};settings:=&reportpkg.UserReportSettings{Filters:[]reportpkg.Filter{{Field:"Метка",Op:"eq",Value:"ALPHA"}}};want,status:=1,http.StatusSeeOther;method:=http.MethodPost
 switch name{case"unfiltered":settings=nil;want=2;case"get-alpha":method=http.MethodGet;case"case-field":settings.Filters[0].Field="мЕТКА";case"numeric":settings.Filters=[]reportpkg.Filter{{Field:"Amount",Op:"ge",Value:"2"}};case"reference":settings.Filters=[]reportpkg.Filter{{Field:"RefValue",Op:"eq",Value:strings.ToUpper(a.String())}};case"masked":settings.Filters=[]reportpkg.Filter{{Field:"Secret",Op:"eq",Value:"RAW_SECRET"}};want=0;case"empty":settings.Filters[0].Value="NONE";want=0;case"variant-filter","variant-budget":settings.Variant="beta";settings.Filters[0].Value="BETA";case"variant-conflict":settings.Variant="beta";want=0;case"unknown-field":settings.Filters[0].Field="Unknown";status=400;case"unsupported-op":settings.Filters[0].Op="bogus";status=400;case"reference-label":settings.Filters=[]reportpkg.Filter{{Field:"RefValue",Op:"eq",Value:"Display label"}};status=400;case"queryless-filter":rep.Query="";status=422;case"queryless-no-filter":rep.Query="";settings=nil;want=0;case"malformed-settings","settings-too-large":status=400;case"query-budget":s.cfg.Limits.ReportMaxRows=1;status=422;case"no-composition":rep.Composition=nil;case"source-read-denied":delete(role.Permissions.Catalogs,entity.Name);status=403}
 if name=="variant-budget"{s.cfg.Limits.ReportMaxRows=1}
 form:=url.Values{};if settings!=nil{raw,err:=settings.JSON();if err!=nil{t.Fatal(err)};form.Set("__settings",raw)}
 if name=="malformed-settings"{form.Set("__settings","{")};if name=="settings-too-large"{form.Set("__settings",strings.Repeat(" ",reportpkg.MaxSourceSettingsBytes+1))}
 if name=="preset-alpha"||name=="default-preset-alpha"{raw,_:=settings.JSON();id,err:=db.SaveReportPreset(ctx,storage.ReportPreset{Report:rep.Name,User:actor.Login,Name:"Own filter",SettingsJSON:raw,IsDefault:name=="default-preset-alpha"});if err!=nil{t.Fatal(err)};form.Del("__settings");if name=="preset-alpha"{form.Set("__preset",id)}}
 target:="/ui/report/"+rep.Name;if method==http.MethodGet{target+="?__run=1&"+form.Encode();form=nil}
 before:=len(s.tabularResults.items);w:=tabularHTTP(s,actor,"own-source-filter",method,target,form);if w.Code!=status{t.Fatalf("HTTP%d want%d: %s",w.Code,status,w.Body.String())}
 if status!=303{if len(s.tabularResults.items)!=before||w.Header().Get("Location")!=""{t.Fatal("unsupported/denied/truncated input published a snapshot")};return}
 path:=w.Header().Get("Location");result:=s.tabularResults.items[strings.TrimPrefix(path,"/ui/tabular/")];if result==nil||result.doc.GetCell(0,0).Text!=fmt.Sprintf("COUNT:%d",want){t.Fatal("result_proc got unfiltered rows")}
 for _,c:=range result.doc.Cells{if strings.Contains(c.Text,"FOREIGN")||strings.Contains(c.Text,"RAW_SECRET"){t.Fatal("RLS/mask leak into module result")}}
 if name=="reference"&&result.doc.GetCell(1,1).Text!=a.String(){t.Fatal("raw reference identity replaced by a display label")}
 view:=tabularHTTP(s,actor,"own-source-filter",http.MethodGet,path,nil);if view.Code!=200||!strings.Contains(view.Body.String(),fmt.Sprintf("COUNT:%d",want)){t.Fatal("HTML does not show filtered snapshot")}
 xlsx:=tabularHTTP(s,actor,"own-source-filter",http.MethodGet,path+"/xlsx",nil);f,err:=excelize.OpenReader(bytes.NewReader(xlsx.Body.Bytes()));if err!=nil{t.Fatal(err)};value,err:=f.GetCellValue(f.GetSheetList()[0],"A1");f.Close();if err!=nil||value!=fmt.Sprintf("COUNT:%d",want){t.Fatal("XLSX is not the filtered snapshot")}
 pdf:=tabularHTTP(s,actor,"own-source-filter",http.MethodGet,path+"/pdf",nil);if pdf.Code!=200||!bytes.HasPrefix(pdf.Body.Bytes(),[]byte("%PDF-")){t.Fatal("filtered snapshot PDF unavailable")};if s.tabularResults.items[strings.TrimPrefix(path,"/ui/tabular/")]!=result||len(s.tabularResults.items)!=before+1{t.Fatal("exports reran report or replaced source snapshot")}
 })}
})}
''',encoding='utf-8')
(W/'internal/api/report_source_filters_test.go').write_text('''package api
import("context";"encoding/json";"net/http";"net/http/httptest";"net/url";"strings";"testing";"github.com/go-chi/chi/v5";"github.com/google/uuid";"github.com/ivantit66/onebase/internal/auth";"github.com/ivantit66/onebase/internal/dbtest";"github.com/ivantit66/onebase/internal/dsl/interpreter";"github.com/ivantit66/onebase/internal/metadata";reportpkg "github.com/ivantit66/onebase/internal/report";"github.com/ivantit66/onebase/internal/runtime";"github.com/ivantit66/onebase/internal/storage")
func TestAPIReportSourceFiltersMatrix(t *testing.T){dbtest.ForEachDialect(t,func(t *testing.T,db *storage.DB){
 ctx:=context.Background();e:=&metadata.Entity{Name:"APISourceFilter",Kind:metadata.KindCatalog,Fields:[]metadata.Field{{Name:"Метка",Type:metadata.FieldTypeString},{Name:"Amount",Type:metadata.FieldTypeNumber},{Name:"Owner",Type:metadata.FieldTypeString},{Name:"Secret",Type:metadata.FieldTypeString}}};if err:=db.Migrate(ctx,[]*metadata.Entity{e});err!=nil{t.Fatal(err)}
 for i,label:=range []string{"ALPHA","BETA","FOREIGN"}{owner:="runner";if i==2{owner="other"};if err:=db.Upsert(ctx,e.Name,uuid.New(),map[string]any{"Метка":label,"Amount":i+1,"Owner":owner,"Secret":"RAW_SECRET"},e);err!=nil{t.Fatal(err)}}
 rep:=&reportpkg.Report{Name:"APISourceFilters",Query:`ВЫБРАТЬ Метка, Amount, Secret ИЗ Справочник.APISourceFilter`,Composition:&reportpkg.Composition{Groupings:[]string{"Метка"},Detail:true,Measures:[]reportpkg.Measure{{Field:"Amount",Agg:"sum"}}},Variants:[]reportpkg.ReportVariant{{Name:"beta",Filters:[]reportpkg.VariantFilter{{Field:"Метка",Op:"eq",Value:"BETA"}}}}}
 reg:=runtime.NewRegistry();reg.Load(runtime.LoadOptions{Entities:[]*metadata.Entity{e},Reports:[]*reportpkg.Report{rep}});h:=&handler{reg:reg,store:db,interp:interpreter.New()};router:=chi.NewRouter();h.mountV2(router)
 perm:=auth.Permission{Reports:map[string][]string{rep.Name:{"run"}},Catalogs:map[string][]string{e.Name:{"read"}},RowAccess:auth.RowAccess{Catalogs:map[string]auth.RowPolicies{e.Name:{"read":{Field:"Owner",Op:"eq",Value:auth.RowValue{User:"login"}}}}},FieldAccess:auth.FieldAccess{Catalogs:map[string]auth.FieldPolicies{e.Name:{"Secret":{Read:"mask_all"}}}}}
 for _,name:=range []string{"alpha","page-after-filter","numeric","case-field","empty","masked","variant","variant-conflict","composed","unknown-field","invalid-op","invalid-json","oversize","source-denied"}{t.Run(name,func(t *testing.T){
 settings:=&reportpkg.UserReportSettings{Filters:[]reportpkg.Filter{{Field:"Метка",Op:"eq",Value:"ALPHA"}}};q:=url.Values{};want,status:=1,200;p:=perm
 switch name{case"page-after-filter":q.Set("limit","1");settings.Filters[0].Value="BETA";case"numeric":settings.Filters=[]reportpkg.Filter{{Field:"Amount",Op:"ge",Value:"2"}};case"case-field":settings.Filters[0].Field="мЕТКА";case"empty":settings.Filters[0].Value="NONE";want=0;case"masked":settings.Filters=[]reportpkg.Filter{{Field:"Secret",Op:"eq",Value:"RAW_SECRET"}};want=0;case"variant":settings.Variant="beta";settings.Filters[0].Value="BETA";case"variant-conflict":settings.Variant="beta";want=0;case"composed":q.Set("compose","1");case"unknown-field":settings.Filters[0].Field="Unknown";status=400;case"invalid-op":settings.Filters[0].Op="bogus";status=400;case"invalid-json","oversize":status=400;case"source-denied":p.Catalogs=map[string][]string{};status=403}
 raw,_:=settings.JSON();q.Set("__settings",raw);if name=="invalid-json"{q.Set("__settings","{")};if name=="oversize"{q.Set("__settings",strings.Repeat(" ",reportpkg.MaxSourceSettingsBytes+1))}
 r:=withUser(httptest.NewRequest("GET","/api/v2/report/"+rep.Name+"?"+q.Encode(),nil),apiUser("runner",p));w:=httptest.NewRecorder();router.ServeHTTP(w,r);if w.Code!=status{t.Fatalf("HTTP%d want%d:%s",w.Code,status,w.Body.String())};if status!=200{return}
 var body struct{Data json.RawMessage `json:"data"`;Meta restV2Meta `json:"meta"`};if err:=json.Unmarshal(w.Body.Bytes(),&body);err!=nil{t.Fatal(err)};if body.Meta.Total!=want{t.Fatalf("filtered total%d want%d",body.Meta.Total,want)};if strings.Contains(string(body.Data),"FOREIGN")||strings.Contains(string(body.Data),"RAW_SECRET"){t.Fatal("RLS/mask leak")};if name!="composed"{var rows []map[string]any;if err:=json.Unmarshal(body.Data,&rows);err!=nil{t.Fatal(err)};if len(rows)!=want{t.Fatal("filter applied after pagination or ignored")}}
 })}
 t.Run("source-budget-refusal",func(t *testing.T){
  for i:=0;i<=restMaxLimit;i++{if err:=db.Upsert(ctx,e.Name,uuid.New(),map[string]any{"Метка":"BETA","Amount":1,"Owner":"runner"},e);err!=nil{t.Fatal(err)}}
  raw,_:=(&reportpkg.UserReportSettings{Filters:[]reportpkg.Filter{{Field:"Метка",Op:"eq",Value:"ALPHA"}}}).JSON();w:=httptest.NewRecorder();router.ServeHTTP(w,withUser(httptest.NewRequest(http.MethodGet,"/api/v2/report/"+rep.Name+"?__settings="+url.QueryEscape(raw),nil),apiUser("runner",perm)));if w.Code!=422{t.Fatalf("cropped filter source HTTP%d",w.Code)}
 })
})}
''',encoding='utf-8')
print('B23 unit/UI/API matrices prepared, no test execution yet')
