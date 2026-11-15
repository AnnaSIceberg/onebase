import pathlib
W=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/report-source-filters-b23-runtime')
(W/'internal/report/source_settings.go').write_text('''package report

import "fmt"

// Match the existing saved-settings budget; execution must not silently fall
// back to an unfiltered report when the explicitly supplied JSON is invalid.
const MaxSourceSettingsBytes = 64 * 1024

func ParseSourceSettings(raw string) (*UserReportSettings, error) {
 if len(raw)>MaxSourceSettingsBytes {return nil,fmt.Errorf("report source settings exceed 64 KiB")}
 settings,err:=ParseUserSettings(raw)
 if err!=nil{return nil,fmt.Errorf("invalid report source settings: %w",err)}
 return settings,nil
}
''',encoding='utf-8')
(W/'internal/report/compose/source_filters.go').write_text('''package compose

import (
 "fmt"
 "strings"
 "github.com/google/uuid"
 "github.com/ivantit66/onebase/internal/report"
)

// ApplySourceFilters filters masked query rows, retaining the ORIGINAL maps and
// typed values for result_proc. It does not resolve references by reading other
// catalogs or expand permissions. Unsupported display-label reference filters
// fail explicitly, rather than selecting an unfiltered input.
func ApplySourceFilters(rows []Row, filters []report.Filter, columns []string) ([]Row,error) {
 if len(filters)==0{return rows,nil}
 if len(filters)>128{return nil,fmt.Errorf("too many report source filters (maximum 128)")}
 fields:=make(map[string]bool,len(columns));for _,c:=range columns {fields[strings.ToLower(c)]=true}
 normalized:=append([]report.Filter(nil),filters...)
 for i:=range normalized {
  f:=&normalized[i];f.Op=strings.ToLower(strings.TrimSpace(f.Op))
  if f.Field==""||!fields[strings.ToLower(f.Field)] {return nil,fmt.Errorf("unknown report source filter field %q",f.Field)}
  switch f.Op {case "eq","ne","gt","ge","lt","le","contains":default:return nil,fmt.Errorf("unsupported report source filter operator %q",f.Op)}
  if len(f.Value)>report.MaxSourceSettingsBytes{return nil,fmt.Errorf("report source filter value is too large")}
  if id,err:=uuid.Parse(f.Value);err==nil {if id==uuid.Nil {f.Value=""}else{f.Value=id.String()}}
 }
 out:=make([]Row,0,len(rows))
 for _,row:=range rows {
  view:=row;copied:=false
  for _,f:=range normalized {
   raw,found:=lookupRowField(row,f.Field);if !found {continue}
   id,ok:=sourceFilterUUID(raw);if !ok {continue}
   if f.Op!="eq"&&f.Op!="ne" {return nil,fmt.Errorf("reference source filter %q supports eq/ne only",f.Field)}
   if f.Value!="" {if _,err:=uuid.Parse(f.Value);err!=nil{return nil,fmt.Errorf("reference source filter %q requires UUID, not a display label",f.Field)}}
   if !copied {view=make(Row,len(row));for k,v:=range row {view[k]=v};copied=true}
   value:="";if id!=uuid.Nil {value=id.String()}
   for key:=range view {if strings.EqualFold(key,f.Field){view[key]=value;break}}
  }
  keep:=true;for _,f:=range normalized {if !matchFilter(view,f){keep=false;break}}
  if keep {out=append(out,row)}
 }
 return out,nil
}

func sourceFilterUUID(v any) (uuid.UUID,bool) {
 switch x:=v.(type) {
 case uuid.UUID:return x,true
 case string:id,e:=uuid.Parse(x);return id,e==nil
 case []byte:id,e:=uuid.Parse(string(x));return id,e==nil
 default:return uuid.Nil,false
 }
}
''',encoding='utf-8')
p=W/'internal/ui/handlers_reports.go';s=p.read_text('utf-8');needle='\tsettings := rs.Settings\n';assert s.count(needle)>=1
# Only runReport obtains explicit source settings; reportForm does not execute modules.
start=s.index('func (s *Server) runReport(');pos=s.index(needle,start)+len(needle)
s=s[:pos]+''' if rep.ResultProc!="" {
  if raw,present:=requestFormValue(r,"__settings");present {
   explicit,err:=reportpkg.ParseSourceSettings(raw)
   if err!=nil {opStatus="error";http.Error(w,s.errText(r,err),http.StatusBadRequest);return}
   if explicit!=nil {settings=reportSettingsWithRequestVariant(r,explicit)}
  }
 }
'''+s[pos:]
needle='\t\tif variantErr == nil && strings.TrimSpace(resultQuery) == "" {\n';assert s.count(needle)==1;s=s.replace(needle,needle+'''   if settings!=nil&&len(settings.Filters)>0 {opStatus="error";http.Error(w,"source filters cannot be applied to a queryless result_proc",http.StatusUnprocessableEntity);return}
''',1)
needle='\t\tif err := s.runTabularReport(w, r.WithContext(opCtx), rep, queryValues, rows); err != nil {';assert s.count(needle)==1;s=s.replace(needle,'''  if settings!=nil&&len(settings.Filters)>0 {
   var filterErr error
   rows,filterErr=compose.ApplySourceFilters(rows,settings.Filters,cols)
   if filterErr!=nil {opStatus="error";http.Error(w,s.errText(r,filterErr),http.StatusBadRequest);return}
  }
'''+needle,1);p.write_text(s,encoding='utf-8')
p=W/'internal/api/v2.go';s=p.read_text('utf-8');start=s.index('func (h *handler) runReportV2()');needle='\t\tvariant := r.URL.Query().Get("variant")\n';pos=s.index(needle,start)
s=s[:pos]+'''  sourceSettings,err:=reportpkg.ParseSourceSettings(r.URL.Query().Get("__settings"))
  if err!=nil {writeError(w,http.StatusBadRequest,err.Error(),"",0);return}
'''+s[pos:]
needle='\t\ttext, values, err := rep.QueryForVariant(variant, params)';pos=s.index(needle,start);s=s[:pos]+'''  if variant==""&&sourceSettings!=nil {variant=sourceSettings.Variant}
'''+s[pos:]
needle='\t\tcompiled, err := h.compileQueryWithRowAccess(r.Context(), text, values)';pos=s.index(needle,start);s=s[:pos]+'''  if sourceSettings!=nil&&len(sourceSettings.Filters)>0&&strings.TrimSpace(text)=="" {writeError(w,http.StatusUnprocessableEntity,"report source filters require a query source","",0);return}
'''+s[pos:]
needle='\t\tif wantsReportComposition(r.URL.Query()) {';pos=s.index(needle,start)
s=s[:pos]+'''  if sourceSettings!=nil&&len(sourceSettings.Filters)>0 {
   // Client filters require a COMPLETE input before pagination. Stay within
   // the existing REST maximum rather than returning silently cropped input.
   rows,cols,truncated,runErr:=query.RunLimit(r.Context(),h.store,&compiled,restMaxLimit)
   if runErr!=nil {writeError(w,http.StatusInternalServerError,runErr.Error(),"",0);return}
   if truncated {writeError(w,http.StatusUnprocessableEntity,"filtered report source exceeds REST row budget; narrow the query or variant","",0);return}
   if err:=maskPlan.Apply(rows);err!=nil {writeError(w,http.StatusForbidden,"masked field: "+err.Error(),"",0);return}
   rows,err=compose.ApplySourceFilters(rows,sourceSettings.Filters,cols)
   if err!=nil {writeError(w,http.StatusBadRequest,err.Error(),"",0);return}
   total:=len(rows)
   if wantsReportComposition(r.URL.Query()) {
    comp:=rep.ActiveComposition(variant)
    if comp==nil {writeError(w,http.StatusBadRequest,"report has no composition","",0);return}
    kind,data,err:=h.composeReportV2(rows,comp)
    if err!=nil {writeError(w,http.StatusBadRequest,"report composition error: "+err.Error(),"",0);return}
    writeJSONV2(w,http.StatusOK,restV2Envelope{Data:data,Meta:&restV2Meta{Total:total,Page:1,Limit:total,TotalPages:totalPages(total,total),Columns:cols,Composed:true,Variant:variant,Kind:kind}});return
   }
   offset,ok:=reportPageOffset(page,limit)
   if !ok {writeError(w,http.StatusBadRequest,"invalid page","",0);return}
   if offset>total {offset=total}
   end:=min(total,offset+limit)
   writeJSONV2(w,http.StatusOK,restV2Envelope{Data:rows[offset:end],Meta:&restV2Meta{Total:total,Page:page,Limit:limit,TotalPages:totalPages(total,limit),Columns:cols,Variant:variant,Truncated:end<total}});return
  }
'''+s[pos:];p.write_text(s,encoding='utf-8')
print('B23 source filters implemented: strict source settings + masked raw input + bounded REST before paging; B22 excluded')
