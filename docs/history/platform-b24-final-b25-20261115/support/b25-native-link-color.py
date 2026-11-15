import pathlib,json,os,hashlib,secrets,socket,sqlite3,subprocess,time,urllib.request,zipfile,xml.etree.ElementTree as ET
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent;M=json.loads((B/'manifest-1115b25-tabular-link-color-DEV.json').read_text('utf-8'));SDK=pathlib.Path(M['artifacts']['windows']['path']);assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b25-native-link-color-attempt1';W.mkdir(exist_ok=False);P=W/'config';DB=W/'own-native.sqlite';PORT=18582;ORIGIN=f'http://127.0.0.1:{PORT}';ADMIN='own_b25_admin';DENIED='own_b25_reader';PW1='B25!'+secrets.token_urlsafe(24);PW2='B25!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for k in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(k,None)
FLAGS=subprocess.CREATE_NO_WINDOW;server=log=driver=browser=context=con=None;errors=[]
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'base_b24':M['base_b24'],'sdk_platform':'windows-amd64','sdk_sha256':M['artifacts']['windows']['sha256'],'checks':{},'cleanup':{},'phase':'prepare','go_started':False,'db_copies':0,'shared_changes':False,'browser_os_network_isolated':False,'browser_request_boundary':'exact own loopback context allowlist'}
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
try:
 with socket.socket()as s:s.bind(('127.0.0.1',PORT))
 write('config/app.yaml','name: Own-T7-Color\n')
 write('catalogs/OwnColorMaster.yaml','name: OwnColorMaster\nfields:\n - {name: Name, type: string}\n')
 write('reports/OwnColor.yaml','name: OwnColor\nresult_proc: Build\nparams:\n - {name: Dummy, type: string, default: own}\n')
 code='Функция Build() Д=Новый ТабличныйДокумент;С=Справочники.OwnColorMaster.НайтиПоРеквизиту("Name","own-master");'
 colors=[('RED','#FF0000','rgb(255, 0, 0)'),('ORANGE','#FF9900','rgb(255, 153, 0)'),('YELLOW','#A3A300','rgb(163, 163, 0)'),('GREEN','#008000','rgb(0, 128, 0)'),('DEFAULT','','')]
 for row,(label,color,rgb)in enumerate(colors,1):
  c='Д.Ячейка('+str(row)+',1)';code+=c+'.Текст="'+label+'";'+c+'.Расшифровка=С;'
  if color:code+=c+'.ЦветТекста="'+color+'";'
 code+='Возврат Д;КонецФункции';write('src/OwnColor.rep.os',code)
 write('roles/OwnColorReader.yaml','name: OwnColorReader\npermissions:\n reports:\n  OwnColor: [run]\n catalogs:\n  OwnColorMaster: [read]\n')
 receipt['fixture_files']=[{'path':str(p.relative_to(P)),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()}for p in sorted(P.rglob('*'))if p.is_file()]
 cli(['migrate','--project',str(P),'--sqlite',str(DB)]);cli(['user','add',ADMIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW1+'\n').encode());cli(['user','add',DENIED,'--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW2+'\n').encode());cli(['user','role','assign',DENIED,'OwnColorReader','--project',str(P),'--sqlite',str(DB)])
 import uuid
 con=sqlite3.connect(DB);ownid=str(uuid.uuid4());con.execute('INSERT INTO owncolormaster(id,name)VALUES(?,?)',(ownid,'own-master'));con.commit();con.close();con=None
 log=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(120):
  if server.poll()is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True);receipt['browser_version']=browser.version;page=login_as(DENIED,PW2);receipt['phase']='native-generate'
 page.goto(ORIGIN+'/ui/report/OwnColor',wait_until='domcontentloaded');page.get_by_role('button',name='Сформировать',exact=True).click();page.wait_for_url('**/ui/tabular/**');require('native_readonly_snapshot',len(page.locator('a[data-sheet-drilldown]').all())==5)
 data=page.locator('a[data-sheet-drilldown]').evaluate_all('(as)=>as.map(a=>({text:a.textContent.trim(),color:getComputedStyle(a).color,style:a.getAttribute("style"),parentColor:getComputedStyle(a.parentElement).color,href:a.getAttribute("href"),target:a.target,rel:a.rel}))');receipt['computed_links']=data
 for item,(label,color,rgb)in zip(data,colors):
  require(label+'_text',item['text']==label);require(label+'_protected_attrs',item['target']=='_blank'and item['rel']=='noopener'and'/cell?row='in item['href'])
  if color:require(label+'_computed_color',item['color']==rgb and item['color']==item['parentColor']and item['style']=='color:inherit')
  else:require('DEFAULT_native_link_unchanged',not item['style']and item['color']!=item['parentColor'])
 page.screenshot(path=str(W/'01-native-T7-four-colors-and-default.png'),full_page=True)
 import urllib.parse,openpyxl
 path=urllib.parse.urlsplit(page.url).path;x=context.request.get(ORIGIN+path+'/xlsx');require('same_snapshot_XLSX200',x.status==200);xp=W/'T7-colors.xlsx';xp.write_bytes(x.body());f=openpyxl.load_workbook(xp,read_only=False,data_only=True)
 for row,(label,color,rgb)in enumerate(colors[:4],1):
  c=f.active.cell(row,1);require(label+'_XLSX_color_unchanged',c.value==label and c.font.color.type=='rgb'and c.font.color.rgb.upper().endswith(color[1:]))
 f.close();r=context.request.get(ORIGIN+data[0]['href'],max_redirects=0);require('native_protected_link_still303',r.status==303 and'/ui/catalog/OwnColorMaster/'in r.headers.get('location',''))
 p=context.request.get(ORIGIN+path+'/pdf');require('same_snapshot_PDF200',p.status==200 and p.body().startswith(b'%PDF-'));(W/'T7-colors.pdf').write_bytes(p.body());receipt['PDF_color_not_claimed']='same model/export available; no separate PDF color parser gate added';require('JS0',not errors);receipt['status']='PASS'
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
