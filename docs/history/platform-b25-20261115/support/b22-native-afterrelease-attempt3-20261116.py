import pathlib,json,os,hashlib,secrets,socket,sqlite3,subprocess,time,urllib.request,zipfile,xml.etree.ElementTree as ET
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent;M=json.loads((B/'manifest-1115b22-report-acl-list-xlsx-DEV.json').read_text('utf-8'));SDK=pathlib.Path(M['artifacts']['windows']['path']);assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b22-native-acl-xlsx-attempt3-20261116';W.mkdir(exist_ok=False);P=W/'config';DB=W/'own-native.sqlite';PORT=18574;ORIGIN=f'http://127.0.0.1:{PORT}';ADMIN='own_b22_admin';DENIED='own_b22_denied';PW1='B22!'+secrets.token_urlsafe(24);PW2='B22!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for k in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(k,None)
FLAGS=subprocess.CREATE_NO_WINDOW;server=log=driver=browser=context=con=None;errors=[]
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'base_b21':M['base_b21'],'sdk_platform':'windows-amd64','sdk_sha256':M['artifacts']['windows']['sha256'],'checks':{},'cleanup':{},'phase':'prepare','go_started':False,'db_copies':0,'shared_changes':False,'browser_os_network_isolated':False,'browser_request_boundary':'exact own loopback context allowlist'}
def require(name,value):
 receipt['checks'][name]=bool(value)
 if not value:raise AssertionError(name)
def write(rel,text):
 p=P/rel;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text,encoding='utf-8')
def cli(args,stdin=None):
 p=subprocess.run([str(SDK),*args],env=ENV,input=stdin,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=FLAGS,timeout=90);(W/f'cli-{len(list(W.glob("cli-*")))}.log').write_bytes(p.stdout)
 if p.returncode:raise RuntimeError('own native CLI setup failed')
def login_as(login,password):
 global context
 if context:context.close()
 context=browser.new_context(viewport={'width':1440,'height':900},accept_downloads=True);context.route('**/*',lambda r:r.continue_()if r.request.url.startswith(ORIGIN+'/')else r.abort())
 page=context.new_page();page.on('pageerror',lambda e:errors.append(type(e).__name__));page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(login);page.locator('input[name="password"]').fill(password);page.locator('button[type="submit"]').click();page.wait_for_timeout(200);return page
def check_xlsx(path,mode):
 import openpyxl
 with zipfile.ZipFile(path)as z:xml=ET.fromstring(z.read('xl/worksheets/sheet1.xml'))
 ns={'x':'http://schemas.openxmlformats.org/spreadsheetml/2006/main'};dim=xml.find('x:dimension',ns).attrib['ref'];rows=xml.findall('x:sheetData/x:row',ns)
 a=openpyxl.load_workbook(path,read_only=False,data_only=True);b=openpyxl.load_workbook(path,read_only=True,data_only=True)
 normal=list(a.active.values);default=list(b.active.values);coords_a=[(c.coordinate,type(c.value).__name__,c.value)for row in a.active.iter_rows()for c in row if c.value is not None];coords_b=[(c.coordinate,type(c.value).__name__,c.value)for row in b.active.iter_rows()for c in row if c.value is not None];a.close();b.close()
 require(mode+'_ZIP_dimension_full',dim=='A1:B28'and len(rows)==28);require(mode+'_default_counts_full',len(default)==28 and all(len(r)==2 for r in default));require(mode+'_default_equals_normal_typed',default==normal and coords_a==coords_b);require(mode+'_grand253',default[-1][0]=='ВСЕГО'and default[-1][1]==253)
 receipt.setdefault('exports',{})[mode]={'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'bytes':path.stat().st_size,'dimension':dim,'rows':len(default),'columns':2,'filled_cells':len(coords_b),'default_reader':openpyxl.__version__,'reset_dimensions_called':False,'forced_bounds':False}
try:
 with socket.socket()as s:s.bind(('127.0.0.1',PORT))
 write('config/app.yaml',"name: B22-native-after-release\nversion: '1.0'\n")
 write('catalogs/B22ACLSource.yaml','name: B22ACLSource\nfields:\n - {name: Name, type: string}\n')
 write('catalogs/B22ListSource.yaml','name: B22ListSource\nfields:\n - {name: Team, type: string}\n - {name: Amount, type: number}\n')
 query='ВЫБРАТЬ Name ИЗ Справочник.B22ACLSource'
 write('reports/B22Direct.yaml','name: B22Direct\nquery: '+query+'\n')
 names=['B22Direct','B22WithQuery','B22Queryless','B22Rethrow','B22Fake','B22BadCompile']
 for name in names[1:]:
  write('reports/'+name+'.yaml','name: '+name+'\nresult_proc: Build\n'+('query: ВЫБРАТЬ 1 КАК Marker\n'if name=='B22WithQuery'else''))
  text='ВЫБРАТЬ'if name=='B22BadCompile'else query
  call='Строки=З.Выполнить();'
  if name=='B22Rethrow':call='Строки=Неопределено;Попытка Строки=З.Выполнить();Исключение ВызватьИсключение;КонецПопытки;'
  code=f'Функция Build() З=Новый Запрос;З.Текст="{text}";{call} Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="READ_GRANTED";Д.Ячейка(1,2).Значение=Строки.Количество();Возврат Д;КонецФункции'
  if name=='B22Fake':code='Функция Build() ВызватьИсключение "нет доступа к объекту: B22ACLSource";КонецФункции'
  write('src/'+name+'.rep.os',code)
 write('reports/B22Ordinary.yaml','''name: B22Ordinary
title: Ordinary XLSX used range
params:
 - {name: Label, type: string, label: Filter, default: all}
query: ВЫБРАТЬ Team, Amount ИЗ Справочник.B22ListSource
composition:
 groupings: [team]
 detail: true
 measures: [{field: amount, agg: sum}]
 totals: {grand: true, subtotals: true}
''')
 role='name: B22ReportOnly\npermissions:\n reports:\n'+''.join('  '+n+': [run]\n'for n in names);write('roles/B22ReportOnly.yaml',role)
 receipt['fixture_files']=[{'path':str(p.relative_to(P)),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()}for p in sorted(P.rglob('*'))if p.is_file()]
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',ADMIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW1+'\n').encode())
 cli(['user','add',DENIED,'--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW2+'\n').encode())
 cli(['user','role','assign',DENIED,'B22ReportOnly','--project',str(P),'--sqlite',str(DB)])
 import uuid
 con=sqlite3.connect(DB);con.execute('INSERT INTO b22aclsource(id,name)VALUES(?,?)',(str(uuid.uuid4()),'SECRET_SYNTHETIC_SOURCE_MARKER'));con.executemany('INSERT INTO b22listsource(id,team,amount)VALUES(?,?,?)',[(str(uuid.uuid4()),'A'if i<=11 else'B',str(i))for i in range(1,23)]);con.commit();con.close();con=None
 log=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(120):
  if server.poll()is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True);receipt['browser_version']=browser.version
 page=login_as(DENIED,PW2);receipt['phase']='native-nonadmin-denied'
 for name in ['B22Direct','B22WithQuery','B22Queryless','B22Rethrow']:
  response=page.goto(ORIGIN+'/ui/report/'+name+'?__run=1',wait_until='domcontentloaded');require(name+'_HTTP403',response.status==403);body=page.locator('body').inner_text();require(name+'_no_data_or_viewer','SECRET_SYNTHETIC_SOURCE_MARKER'not in body and '/ui/tabular/'not in page.url)
  if name=='B22Queryless':page.screenshot(path=str(W/'01-native-denial403.png'),full_page=True)
 for name in ['B22Fake','B22BadCompile']:
  response=page.goto(ORIGIN+'/ui/report/'+name+'?__run=1',wait_until='domcontentloaded');require(name+'_HTTP500_not_ACL',response.status==500)
  if name=='B22Fake':page.screenshot(path=str(W/'02-user-error-stays500.png'),full_page=True)
 page=login_as(ADMIN,PW1);receipt['phase']='native-positive'
 response=page.goto(ORIGIN+'/ui/report/B22Direct?__run=1',wait_until='domcontentloaded');require('allowed_direct200_and_data',response.status==200 and 'SECRET_SYNTHETIC_SOURCE_MARKER'in page.locator('body').inner_text())
 for name in ['B22WithQuery','B22Queryless']:
  response=page.goto(ORIGIN+'/ui/report/'+name+'?__run=1',wait_until='domcontentloaded');require(name+'_allowed_native_viewer',response.status==200 and '/ui/tabular/'in page.url and 'READ_GRANTED'in page.locator('body').inner_text())
  if name=='B22Queryless':page.screenshot(path=str(W/'03-allowed-native-viewer.png'),full_page=True)
 receipt['phase']='ordinary-native-generate'
 page.goto(ORIGIN+'/ui/report/B22Ordinary',wait_until='domcontentloaded')
 with page.expect_response(lambda r:r.request.method=='POST'and '/ui/report/B22Ordinary'in r.url,timeout=30000)as pending:page.get_by_role('button',name='Сформировать',exact=True).click()
 require('ordinary_native_generate200',pending.value.status==200);page.locator('tr.grand').wait_for(state='visible');require('ordinary_visible_grand253','253'in page.locator('tr.grand').inner_text());page.screenshot(path=str(W/'04-ordinary-report-full.png'),full_page=True)
 direct_response=context.request.get(ORIGIN+'/ui/report/B22Ordinary/excel?Label=all')
 require('direct_public_route200',direct_response.status==200)
 direct=W/'ordinary-direct-b22.xlsx';direct.write_bytes(direct_response.body());check_xlsx(direct,'direct')
 receipt['phase']='ordinary-native-background'
 page.get_by_role('link',name='Excel',exact=True).first.click();require('background_native_job_page','/ui/export-jobs/'in page.url);page.locator('a[href$="/download"]').wait_for(state='visible',timeout=30000)
 with page.expect_download(timeout=30000)as pending:page.locator('a[href$="/download"]').click()
 background=W/'ordinary-background-b22.xlsx';pending.value.save_as(background);check_xlsx(background,'background');page.screenshot(path=str(W/'05-background-job.png'),full_page=True)
 require('JS0',not errors);receipt['status']='PASS'
except Exception as e:
 import traceback
 receipt['status']='FAIL';receipt['failure_type']=type(e).__name__;receipt['failure_line']=traceback.extract_tb(e.__traceback__)[-1].lineno;receipt['failed_check']=str(e)if isinstance(e,AssertionError)else None
finally:
 if con is not None:
  try:con.close()
  except Exception:pass
 for name,obj in [('context',context),('browser',browser)]:
  try:
   if obj:obj.close()
   receipt['cleanup'][name]=True
  except Exception:receipt['cleanup'][name]=False
 try:
  if driver:driver.stop()
  receipt['cleanup']['driver']=True
 except Exception:receipt['cleanup']['driver']=False
 if server:
  server.terminate()
  try:server.wait(timeout=8)
  except subprocess.TimeoutExpired:server.kill();server.wait(timeout=3)
 receipt['cleanup']['app']=server is None or server.poll()is not None
 if log:log.close()
 try:
  with socket.create_connection(('127.0.0.1',PORT),timeout=.2):receipt['cleanup']['port_closed']=False
 except OSError:receipt['cleanup']['port_closed']=True
 receipt['cleanup']['tunnel_not_used']=True
 if receipt['cleanup']['app']:
  for p in [DB,pathlib.Path(str(DB)+'-wal'),pathlib.Path(str(DB)+'-shm')]:
   assert p.resolve().is_relative_to(W.resolve())
   try:
    if p.exists():p.unlink()
   except OSError:receipt['cleanup']['db_delete_failed']=False
 receipt['cleanup']['own_auth_db_removed']=not DB.exists();receipt['page_errors']=len(errors)
 (W/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');print(json.dumps(receipt,ensure_ascii=True))
 if receipt['status']!='PASS'or not all(receipt['cleanup'].values()):raise SystemExit(1)
