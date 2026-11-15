"""Accept only fresh B20 XLSX exports; use installed streaming reader unchanged."""
import collections, hashlib, json, pathlib, sys
import openpyxl

assert sys.version_info[:2] == (3,14)
assert openpyxl.__version__ == '3.1.5'
root=pathlib.Path(sys.argv[1]).resolve()
assert root.name.startswith('b20-native-dimension-attempt')
files=sorted(root.glob('*-b20.xlsx'))
assert files, 'no fresh B20 exports'
result={'python':sys.version,'executable':sys.executable,'openpyxl':openpyxl.__version__,'module':openpyxl.__file__,'read_only_default':True,'reset_dimensions_called':False,'forced_iteration_bounds':False,'files':[]}
def collect(ws):
 rows=positions=filled=0;values=[];kinds=collections.Counter()
 for row in ws.iter_rows():
  rows+=1;positions+=len(row)
  for col,cell in enumerate(row,1):
   if cell.value is not None:
    filled+=1;kinds[type(cell.value).__name__]+=1;values.append((rows,col,cell.value))
 return {'rows':rows,'columns':ws.max_column,'max_row':ws.max_row,'positions':positions,'nonempty_values':filled,'value_types':dict(kinds)},values
for p in files:
 assert p.resolve().parent==root
 normal=openpyxl.load_workbook(p,read_only=False,data_only=False)
 streamed=openpyxl.load_workbook(p,read_only=True,data_only=False)
 try:
  a,av=collect(normal.worksheets[0]);b,bv=collect(streamed.worksheets[0])
  empty=p.name=='empty-b20.xlsx'
  equal=av==bv
  same_counts=all(a[k]==b[k]for k in ['rows','columns','max_row','positions','nonempty_values','value_types'])
  assert equal and (same_counts or empty),(p.name,a,b)
  if p.name=='large53893-b20.xlsx':assert b['rows']==830 and b['columns']==67 and b['positions']==55610 and b['nonempty_values']==141
  if p.name=='auto51-b20.xlsx':assert b['rows']==51 and b['columns']==67 and b['positions']==3417 and b['nonempty_values']==141
  if empty:assert a['nonempty_values']==b['nonempty_values']==0
  with p.open('rb')as f:sha=hashlib.file_digest(f,'sha256').hexdigest()
  result['files'].append({'file':p.name,'bytes':p.stat().st_size,'sha256':sha,'normal':a,'default_read_only':b,'filled_coordinates_and_typed_values_equal':equal,'counts_equal':same_counts,'empty_document_no_values_exception':empty and not same_counts})
 finally:
  normal.close();streamed.close()
out=root/'installed-default-reader.json';assert not out.exists()
out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'status':'PASS','files':len(files),'output':str(out),'sha256':hashlib.sha256(out.read_bytes()).hexdigest()}))
