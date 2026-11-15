import pathlib
W=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/reports-p4-p7-b21-runtime')
for pkg,cn,en,dn in [('ui','ContextStaff','ContextEntry','ContextDepartment'),('api','APIContextStaff','APIContextEntry','APIContextDepartment')]:
 p=W/f'internal/{pkg}/report_context_test.go';s=p.read_text(encoding='utf-8')
 s=s.replace('card := &metadata.Entity{Name: "'+cn+'"','department := &metadata.Entity{Name: "'+dn+'", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}\n\t\tunitID, otherUnitID := uuid.New(), uuid.New()\n\t\tcard := &metadata.Entity{Name: "'+cn+'"',1)
 s=s.replace('{Name: "Department", Type: metadata.FieldTypeString}', '{Name: "Department", Type: "reference:'+dn+'", RefEntity: "'+dn+'"}')
 name='row' if pkg=='ui' else 'entity'
 s=s.replace('[]*metadata.Entity{card, '+name+'}', '[]*metadata.Entity{department, card, '+name+'}')
 marker='\t\tcardID := uuid.New()' if pkg=='ui' else '\t\tif err := db.Upsert(ctx, card.Name, uuid.New(),'
 pos=s.index(marker);s=s[:pos]+'''\t\tfor _, id := range []uuid.UUID{unitID, otherUnitID} {
            if err := db.Upsert(ctx, department.Name, id, map[string]any{"Name": "synthetic unit"}, department); err != nil { t.Fatal(err) }
        }
'''+s[pos:]
 s=s.replace('"Department": "OWN_UNIT"','"Department": unitID.String()').replace('"Department": "OTHER_UNIT"','"Department": otherUnitID.String()')
 s=s.replace('{Name: "Unit", Type: "string", Context:', '{Name: "Unit", Type: "reference:'+dn+'", Context:')
 # Client spoof deliberately differs from the valid server UUID.
 s=s.replace('"Unit": {"OTHER_UNIT"}', '"Unit": {otherUnitID.String()}')
 marker='\t\tt.Run("post-spoof"' if pkg=='ui' else '\t\th := &handler'
 pos=s.index(marker)
 binding='''        bound, err := reportpkg.ResolveContextParams(auth.ContextWithUser(ctx, user), rep, map[string]any{"Unit": otherUnitID.String()}, reg, db)
        if err != nil { t.Fatal(err) }
        if actual, ok := bound["Unit"].(uuid.UUID); !ok || actual != unitID { t.Fatalf("native card reference was reparsed/lost: %T", bound["Unit"]) }
'''
 s=s[:pos]+binding+s[pos:]
 if pkg=='ui':
  s=s.replace('"github.com/ivantit66/onebase/internal/dsl/interpreter"','"github.com/ivantit66/onebase/internal/dsl/interpreter"\n"github.com/ivantit66/onebase/internal/dsl/ast"')
  marker='\t\tt.Run("ambiguous-card-refused"'
  pos=s.index(marker)
  block='''        for _, mode := range []string{"queryless-result-proc", "query-result-proc"} {
            t.Run(mode, func(t *testing.T) {
                oldQuery := rep.Query
                rep.ResultProc = "BuildContext"
                if mode == "queryless-result-proc" { rep.Query = "" }
                defer func(){ rep.Query=oldQuery;rep.ResultProc="" }()
                s.reg.Load(runtime.LoadOptions{Reports: []*reportpkg.Report{rep}, Programs: map[string]*ast.Program{rep.Name: mustParse(t, `Функция BuildContext()
 Д=Новый ТабличныйДокумент;
 Д.Ячейка(1,1).Текст=Строка(Параметры.Unit);
 Д.Ячейка(2,1).Текст=Параметры.Actor;
 Д.Ячейка(3,1).Значение=Результат.Количество();
 Возврат Д;
 КонецФункции`)}})
                s.interp.LookupProc=s.reg.GetModuleProc
                for _, method := range []string{http.MethodGet,http.MethodPost} {
                    target:="/ui/report/"+rep.Name
                    form:=spoof
                    if method==http.MethodGet { target+="?__run=1&"+spoof.Encode();form=nil }
                    response:=tabularHTTP(s,user,"context-result-session",method,target,form)
                    if response.Code!=http.StatusSeeOther { t.Fatalf("result binding:%d %s",response.Code,response.Body.String()) }
                    result:=s.tabularResults.items[strings.TrimPrefix(response.Header().Get("Location"),"/ui/tabular/")]
                    if result==nil || result.doc.Cell(0,0).Text!=unitID.String() || result.doc.Cell(1,0).Text!=user.Login { t.Fatal("result_proc used client or missing context") }
                    wantRows:="1";if mode=="queryless-result-proc" {wantRows="0"}
                    if fmt.Sprint(result.doc.Cell(2,0).Value)!=wantRows { t.Fatalf("result rows %v",result.doc.Cell(2,0).Value) }
                    if view:=tabularHTTP(s,user,"context-result-session",http.MethodGet,response.Header().Get("Location"),nil);view.Code!=http.StatusOK { t.Fatal("bound viewer inaccessible") }
                    if xlsx:=tabularHTTP(s,user,"context-result-session",http.MethodGet,response.Header().Get("Location")+"/xlsx",nil);xlsx.Code!=http.StatusOK { t.Fatal("bound snapshot export inaccessible") }
                }
            })
        }
'''
  s=s[:pos]+block+s[pos:];s=s.replace('"context"','"context"\n"fmt"',1)
 p.write_text(s,encoding='utf-8')
p=W/'internal/ui/report_drilldown_test.go';s=p.read_text(encoding='utf-8');s=s.replace('"context"','"context"\n"fmt"',1)
s=s.replace('"flat", "composed"','"flat", "composed", "flat12-detail-fields"')
marker='\t\t\t\tw := run("POST", "/ui/report/TypedActions")';pos=s.index(marker)
s=s[:pos]+'''                if mode=="flat12-detail-fields" {
                    rep.Composition=&reportpkg.Composition{Detail:true}
                    for _, field:=range []string{"doccell","photo","recording","call"} { rep.Composition.DetailFields=append(rep.Composition.DetailFields,reportpkg.DetailField{Field:field}) }
                    for i:=0;i<8;i++ {field:=fmt.Sprintf("f%d",i);rep.Query+=fmt.Sprintf(", %d AS %s",i,field);rep.Composition.DetailFields=append(rep.Composition.DetailFields,reportpkg.DetailField{Field:field})}
                }
'''+s[pos:]
# Extra projections belong before FROM, not after it.
s=s.replace('rep.Query+=fmt.Sprintf(", %d AS %s",i,field)', 'rep.Query=strings.Replace(rep.Query," ИЗ ",fmt.Sprintf(", %d AS %s ИЗ ",i,field),1)')
s=s.replace('\t\t\t\tgot := links(w.Body.String())','\t\t\t\tif mode=="flat12-detail-fields" { assertFlatReportColumns(t,w.Body.String(),13) }\n\t\t\t\tgot := links(w.Body.String())',1)
p.write_text(s,encoding='utf-8')
print('Runtime fixtures upgraded: native DB UUID context UI/API + queryless/query result_proc + P4 flat12')
