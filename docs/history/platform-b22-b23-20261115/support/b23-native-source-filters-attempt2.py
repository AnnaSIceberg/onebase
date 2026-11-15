import pathlib,json,os,hashlib,secrets,socket,sqlite3,subprocess,time,urllib.request,zipfile,xml.etree.ElementTree as ET
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent;M=json.loads((B/'manifest-1115b23-report-source-filters-DEV.json').read_text('utf-8'));SDK=pathlib.Path(M['artifacts']['windows']['path']);assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b23-native-source-filters-attempt2';W.mkdir(exist_ok=False);P=W/'config';DB=W/'own-native.sqlite';PORT=18576;ORIGIN=f'http://127.0.0.1:{PORT}';ADMIN='own_b23_admin';DENIED='own_b23_reader';PW1='B23!'+secrets.token_urlsafe(24);PW2='B23!'+secrets.token_urlsafe(24)
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
def settings(value):return json.dumps({'variant':'','filters':[{'field':'Метка','op':'eq','value':value}]},ensure_ascii=False)
def exports(path,tag,want):
 import openpyxl
 from pypdf import PdfReader
 x=context.request.get(ORIGIN+path+'/xlsx');require(tag+'_xlsx200',x.status==200);xp=W/(tag+'.xlsx');xp.write_bytes(x.body());f=openpyxl.load_workbook(xp,read_only=True,data_only=True);vals=[c for r in f.active.values for c in r if c is not None];f.close();text=' '.join(map(str,vals));require(tag+'_xlsx_filtered',('COUNT:'+str(want))in text and ('BETA'not in text if want==1 else 'BETA'in text))
 p=context.request.get(ORIGIN+path+'/pdf');require(tag+'_pdf200',p.status==200);pp=W/(tag+'.pdf');pp.write_bytes(p.body());pdf=PdfReader(pp);txt=' '.join(page.extract_text()for page in pdf.pages);pages=len(pdf.pages);require(tag+'_pdf_filtered',('COUNT:'+str(want))in txt and ('BETA'not in txt if want==1 else 'BETA'in txt));receipt.setdefault('exports',{})[tag]={'xlsx_sha256':hashlib.sha256(xp.read_bytes()).hexdigest(),'xlsx_values':vals,'pdf_sha256':hashlib.sha256(pp.read_bytes()).hexdigest(),'pdf_pages':pages,'same_snapshot_path':path}
try:
 with socket.socket()as s:s.bind(('127.0.0.1',PORT))
 write('config/app.yaml',"name: B23-native-source-filters\nversion: '1.0'\n")
 write('catalogs/B23Source.yaml','name: B23Source\nfields:\n - {name: Метка, type: string}\n - {name: Количество, type: number}\n')
 base='''params:
 - {name: Dummy, type: string, default: own}
query: ВЫБРАТЬ Метка, Количество ИЗ Справочник.B23Source
composition:
 groupings: [Метка]
 detail: true
 measures: [{field: Количество, agg: sum}]
variants:
 - name: alpha
   filters: [{field: Метка, op: eq, value: ALPHA}]
'''
 write('reports/B23Ordinary.yaml','name: B23Ordinary\n'+base)
 write('reports/B23Tabular.yaml','name: B23Tabular\nresult_proc: Build\n'+base)
 write('src/B23Tabular.rep.os','Функция Build() Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="COUNT:"+Строка(Результат.Количество());Н=2;Для Каждого Стр Из Результат Цикл Д.Ячейка(Н,1).Текст=Стр.Метка;Д.Ячейка(Н,2).Значение=Стр.Количество;Н=Н+1;КонецЦикла;Возврат Д;КонецФункции')
 write('roles/B23Reader.yaml','name: B23Reader\npermissions:\n reports:\n  B23Ordinary: [run]\n  B23Tabular: [run]\n catalogs:\n  B23Source: [read]\n')
 receipt['fixture_files']=[{'path':str(p.relative_to(P)),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()}for p in sorted(P.rglob('*'))if p.is_file()]
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',ADMIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW1+'\n').encode())
 cli(['user','add',DENIED,'--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW2+'\n').encode())
 cli(['user','role','assign',DENIED,'B23Reader','--project',str(P),'--sqlite',str(DB)])
 import uuid,urllib.parse
 con=sqlite3.connect(DB);con.executemany('INSERT INTO b23source(id,"метка","количество")VALUES(?,?,?)',[(str(uuid.uuid4()),'ALPHA',1),(str(uuid.uuid4()),'BETA',2)]);con.commit();con.close();con=None
 log=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(120):
  if server.poll()is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True);receipt['browser_version']=browser.version
 page=login_as(DENIED,PW2);receipt['phase']='ordinary-filter-control'
 q=urllib.parse.urlencode({'__run':'1','__settings':settings('ALPHA')});response=page.goto(ORIGIN+'/ui/report/B23Ordinary?'+q,wait_until='domcontentloaded');txt=page.locator('body').inner_text();require('ordinary_same_filter_ALPHA_only',response.status==200 and 'ALPHA'in txt and 'BETA'not in txt);page.screenshot(path=str(W/'01-ordinary-alpha.png'),full_page=True)
 receipt['phase']='native-generate-unfiltered'
 page.goto(ORIGIN+'/ui/report/B23Tabular',wait_until='domcontentloaded');page.get_by_role('button',name='Сформировать',exact=True).click();page.wait_for_url('**/ui/tabular/**');require('native_Generate_full2','COUNT:2'in page.locator('body').inner_text());exports(urllib.parse.urlsplit(page.url).path,'unfiltered',2)
 receipt['phase']='native-get-filtered'
 response=page.goto(ORIGIN+'/ui/report/B23Tabular?'+q,wait_until='domcontentloaded');txt=page.locator('body').inner_text();require('native_GET_filtered1',response.status==200 and '/ui/tabular/'in page.url and 'COUNT:1'in txt and 'ALPHA'in txt and 'BETA'not in txt);exports(urllib.parse.urlsplit(page.url).path,'filtered-get',1);page.screenshot(path=str(W/'02-tabular-alpha-snapshot.png'),full_page=True)
 receipt['phase']='public-post-and-variant'
 response=context.request.post(ORIGIN+'/ui/report/B23Tabular',form={'Dummy':'own','__settings':settings('ALPHA')},max_redirects=0);require('public_POST_filtered_snapshot303',response.status==303 and response.headers.get('location','').startswith('/ui/tabular/'));path=response.headers['location'];page.goto(ORIGIN+path,wait_until='domcontentloaded');require('public_POST_viewer_filtered1','COUNT:1'in page.locator('body').inner_text());exports(path,'filtered-post',1)
 qvariant=urllib.parse.urlencode({'__run':'1','__variant':'alpha','__settings':settings('ALPHA')});page.goto(ORIGIN+'/ui/report/B23Tabular?'+qvariant,wait_until='domcontentloaded');require('P2_variant_AND_user_filter','COUNT:1'in page.locator('body').inner_text()and'BETA'not in page.locator('body').inner_text())
 receipt['phase']='REST-filter-before-pagination'
 r=context.request.get(ORIGIN+'/api/v2/report/B23Tabular?'+urllib.parse.urlencode({'__settings':settings('ALPHA'),'limit':'1'}));require('REST_filtered200',r.status==200);v=r.json();require('REST_filtered_total1_rows1',v['meta']['total']==1 and len(v['data'])==1 and list(v['data'][0].values()).count('ALPHA')==1)
 receipt['phase']='invalid-filter-denial'
 invalid=json.dumps({'filters':[{'field':'Unknown','op':'eq','value':'ALPHA'}]});r=context.request.post(ORIGIN+'/ui/report/B23Tabular',form={'__settings':invalid},max_redirects=0);require('unsupported_filter_no_snapshot_HTTP400',r.status==400 and not r.headers.get('location'))
 page.screenshot(path=str(W/'03-p2-filtered-viewer.png'),full_page=True);require('JS0',not errors);receipt['status']='PASS'
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
