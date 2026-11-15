import pathlib,json,os,hashlib,secrets,socket,sqlite3,subprocess,time,urllib.request,zipfile,xml.etree.ElementTree as ET
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent;M=json.loads((B/'manifest-1115b24-tabular-block-output-DEV.json').read_text('utf-8'));SDK=pathlib.Path(M['artifacts']['windows']['path']);assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b24-native-block-output-attempt2';W.mkdir(exist_ok=False);P=W/'config';DB=W/'own-native.sqlite';PORT=18578;ORIGIN=f'http://127.0.0.1:{PORT}';ADMIN='own_b24_admin';DENIED='own_b24_reader';PW1='B24!'+secrets.token_urlsafe(24);PW2='B24!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for k in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(k,None)
FLAGS=subprocess.CREATE_NO_WINDOW;server=log=driver=browser=context=con=None;errors=[]
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'base_b23':M['base_b23'],'sdk_platform':'windows-amd64','sdk_sha256':M['artifacts']['windows']['sha256'],'checks':{},'cleanup':{},'phase':'prepare','go_started':False,'db_copies':0,'shared_changes':False,'browser_os_network_isolated':False,'browser_request_boundary':'exact own loopback context allowlist'}
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
def get_exports(path,tag):
 import openpyxl
 from pypdf import PdfReader
 r=context.request.get(ORIGIN+path+'/xlsx');require(tag+'_XLSX200',r.status==200);p=W/(tag+'.xlsx');p.write_bytes(r.body());book=openpyxl.load_workbook(p,read_only=True,data_only=True);cells={c.coordinate:c.value for row in book.active for c in row if c.value is not None};book.close()
 r=context.request.get(ORIGIN+path+'/pdf');require(tag+'_PDF200',r.status==200);q=W/(tag+'.pdf');q.write_bytes(r.body());coords=[]
 def visitor(text,cm,tm,font,size):
  if text.strip():coords.append({'text':text.strip(),'x':round(tm[4],2),'y':round(tm[5],2)})
 pdf=PdfReader(q)
 for page in pdf.pages:page.extract_text(visitor_text=visitor)
 receipt.setdefault('exports',{})[tag]={'xlsx_sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'PDF_sha256':hashlib.sha256(q.read_bytes()).hexdigest(),'filled_cells':cells,'PDF_text_positions':coords,'PDF_pages':len(pdf.pages),'snapshot_path':path,'default_reader':'openpyxl3.1.5 read_only; no reset'}
 return cells,coords
try:
 with socket.socket()as s:s.bind(('127.0.0.1',PORT))
 receipt['phase']='original-KPI-repro-fixture'
 original=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/perenos-codex2-run-20261115T120509402724Z/repro-project.zip');require('original_repro_ZIP_pinned',hashlib.sha256(original.read_bytes()).hexdigest()=='a7bcd66c34ed33c7d95db3c1ad5417cdc3121b4ae6d8bfc839dc4f125facd3a6')
 P.mkdir(parents=True)
 with zipfile.ZipFile(original)as z:
  receipt['original_repro_files']=[]
  for entry in z.infolist():
   p=P/entry.filename;require('repro_path_within_own_config_'+str(len(receipt['original_repro_files'])),p.resolve().is_relative_to(P.resolve()));p.parent.mkdir(parents=True,exist_ok=True);raw=z.read(entry);p.write_bytes(raw);receipt['original_repro_files'].append({'path':entry.filename,'sha256':hashlib.sha256(raw).hexdigest()})
 code='Функция Build() Д=Новый ТабличныйДокумент;'
 def out(name,rows,cols,text,method):return name+'=Д.Область(1,1,'+str(rows)+','+str(cols)+');'+name+'.Текст="'+text+'";Д.'+method+'('+name+');'
 code+=out('З',3,12,'TITLE','Вывести')+out('Л',2,3,'HEAD','Вывести')+out('П',2,9,'REASON','Присоединить')+out('ЛС',1,1,'NAME','Вывести')+out('ПС',1,11,'JAN','Присоединить')+out('Футер',1,12,'FOOT','Вывести')+'Возврат Д;КонецФункции'
 write('reports/OwnA002Block.yaml','name: OwnA002Block\nresult_proc: Build\n');write('src/OwnA002Block.rep.os',code)
 cli(['migrate','--project',str(P),'--sqlite',str(DB)]);cli(['user','add',ADMIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW1+'\n').encode())
 log=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(120):
  if server.poll()is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True);receipt['browser_version']=browser.version;page=login_as(ADMIN,PW1)
 import urllib.parse
 receipt['phase']='managed-original-KPI-action'
 page.goto(ORIGIN+'/ui/processor/'+urllib.parse.quote('ТестПрисоединения'),wait_until='domcontentloaded')
 with page.expect_response(lambda r:r.request.method=='POST'and'/form-event'in r.url,timeout=30000)as pending:page.get_by_role('button',name='Проверка',exact=True).click()
 require('native_managed_action_HTTP200',pending.value.status==200);page.wait_for_function("()=>{const f=document.querySelector('iframe[data-ob-tabular-field]');return f&&!!f.getAttribute('src')&&!f.hidden;}");path=page.locator('iframe[data-ob-tabular-field]').get_attribute('src');require('native_managed_published_snapshot',path.startswith('/ui/tabular/'));cells,coords=get_exports(path,'KPI-T5-original')
 require('KPI_XLSX_exact_2x2',cells=={'A1':'LEFT-1','A2':'LEFT-2','B1':'RIGHT-1','B2':'RIGHT-2'})
 by={v['text']:v for v in coords};require('KPI_PDF_same_rows_right_of_left',all(abs(by['LEFT-'+str(i)]['y']-by['RIGHT-'+str(i)]['y'])<.1 and by['RIGHT-'+str(i)]['x']>by['LEFT-'+str(i)]['x']for i in [1,2]))
 html=context.request.get(ORIGIN+path).text();require('KPI_HTML_all_4_cells',all(x in html for x in ['LEFT-1','LEFT-2','RIGHT-1','RIGHT-2']));page.screenshot(path=str(W/'01-original-KPI-managed-2x2.png'),full_page=True)
 receipt['phase']='A002-minimal-header-month-footer'
 page.goto(ORIGIN+'/ui/report/OwnA002Block?__run=1',wait_until='domcontentloaded');path=urllib.parse.urlsplit(page.url).path;require('A002_native_viewer',path.startswith('/ui/tabular/'));cells,coords=get_exports(path,'A002-block-minimal');require('A002_header_rows4_5',cells['D4']=='REASON'and cells['L5']=='REASON');require('A002_month_column2',cells['B6']=='JAN');require('A002_footer_12columns',cells['A7']=='FOOT'and cells['L7']=='FOOT'and not any(k.startswith(('M','N','O','P','Q','R','S','T'))for k in cells));require('A002_geometry7x12',len(cells)==84)
 header=[x for x in coords if x['text']=='REASON'];month=[x for x in coords if x['text']=='JAN'];foot=[x for x in coords if x['text']=='FOOT'];name=[x for x in coords if x['text']=='NAME'];require('A002_PDF_month_right_same_row',bool(month and name)and abs(month[0]['y']-name[0]['y'])<.1 and month[0]['x']>name[0]['x']);require('A002_PDF_footer_below_body',bool(foot)and foot[0]['y']<name[0]['y']);page.screenshot(path=str(W/'02-A002-minimal-geometry.png'),full_page=True)
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
