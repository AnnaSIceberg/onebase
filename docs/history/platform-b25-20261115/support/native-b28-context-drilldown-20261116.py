import pathlib,hashlib,json,os,sqlite3,socket,subprocess,time,secrets,uuid,urllib.request,urllib.parse,zipfile,io
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent;M=json.loads((B/'manifest-b28-native-input-20261116.json').read_text('utf-8'));SDK=pathlib.Path(M['artifacts']['windows']['path']);assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b28-native-context-drilldown-attempt1-20261116';W.mkdir(exist_ok=False);P=W/'config';DB=W/'own-native.sqlite';PORT=18572;ORIGIN=f'http://127.0.0.1:{PORT}';LOGIN='own_b21_context';PASSWORD='B21!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for k in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(k,None)
FLAGS=subprocess.CREATE_NO_WINDOW;server=log=driver=browser=context=con=None;errors=[];network=[]
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'upstream_inputs':M['upstream_inputs'],'sdk_platform':'windows-amd64','sdk_sha256':M['artifacts']['windows']['sha256'],'checks':{},'cleanup':{},'phase':'source-fixture','db_copies':0,'shared_changes':False,'go_started':False,'browser_os_network_isolated':False,'browser_request_boundary':'own context exact loopback origin allowlist'}
def require(name,ok):
 receipt['checks'][name]=bool(ok)
 if not ok:raise AssertionError(name)
def write(rel,text):
 p=P/rel;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text,encoding='utf-8')
def cli(args,stdin=None):
 p=subprocess.run([str(SDK),*args],env=ENV,input=stdin,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=FLAGS,timeout=90);(W/f'cli-{len(list(W.glob("cli-*")))}.log').write_bytes(p.stdout)
 if p.returncode:raise RuntimeError('own native CLI fixture failed')
try:
 with socket.socket() as sock:sock.bind(('127.0.0.1',PORT))
 write('config/app.yaml',"name: B21-native-context\nversion: '1.0'\n")
 write('catalogs/B21Department.yaml','name: B21Department\nfields:\n - {name: Name, type: string}\n')
 write('catalogs/B21Staff.yaml','name: B21Staff\nfields:\n - {name: Account, type: reference:_users}\n - {name: Department, type: reference:B21Department}\n')
 write('catalogs/B21Entry.yaml','name: B21Entry\nfields:\n - {name: Name, type: string}\n - {name: Owner, type: string}\n - {name: Department, type: reference:B21Department}\n')
 params="params:\n - {name: Actor, type: string, context: {user: login}}\n - name: Unit\n   type: reference:B21Department\n   context: {user_card: {catalog: B21Staff, account_field: Account, field: Department}}\n"
 query='ВЫБРАТЬ Name, Ссылка КАК RefID ИЗ Справочник.B21Entry ГДЕ Owner=&Actor И Department=&Unit'
 write('reports/B21Context.yaml','name: B21Context\ntitle: Trusted context\n'+params+'query: '+query+'\n')
 write('reports/B21Result.yaml','name: B21Result\ntitle: Trusted queryless result\n'+params+'result_proc: BuildContext\n')
 write('src/B21Result.rep.os','''Функция BuildContext()
 Д=Новый ТабличныйДокумент;
 Д.Ячейка(1,1).Текст="Trusted unit";
 Д.Ячейка(1,2).Текст=Строка(Параметры.Unit);
 Д.Ячейка(2,1).Текст="Trusted actor";
 Д.Ячейка(2,2).Текст=Параметры.Actor;
 Возврат Д;
 КонецФункции
''')
 detail='composition:\n groupings: []\n detail: true\n detail_fields:\n  - {field: name, title: Cell target}\n'
 projection='ВЫБРАТЬ Name, Ссылка КАК RefID'
 for i in range(11):projection+=f', {i} КАК F{i}';detail+=f'  - {{field: f{i}, title: Field{i}}}\n'
 projection+=' ИЗ Справочник.B21Entry ГДЕ Owner=&Actor И Department=&Unit'
 dd='drilldowns:\n - field: name\n   id_field: refid\n   target: {kind: entity, entity: B21Entry}\n - title: Open row\n   id_field: refid\n   target: {kind: entity, entity: B21Entry}\n'
 write('reports/B21Typed.yaml','name: B21Typed\ntitle: Typed flat12\n'+params+'query: '+projection+'\n'+detail+dd)
 receipt['source_fixture_files']=[{'path':str(p.relative_to(P)),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()} for p in sorted(P.rglob('*')) if p.is_file()]
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',LOGIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PASSWORD+'\n').encode())
 unit,other,ownrow,foreignrow=str(uuid.uuid4()),str(uuid.uuid4()),str(uuid.uuid4()),str(uuid.uuid4())
 con=sqlite3.connect(DB);con.execute('PRAGMA foreign_keys=ON');actor=con.execute('SELECT id FROM _users WHERE login=?',(LOGIN,)).fetchone()[0]
 con.executemany('INSERT INTO b21department(id,name) VALUES(?,?)',[(unit,'OWN_UNIT_LABEL'),(other,'OTHER_UNIT_LABEL')])
 con.execute('INSERT INTO b21staff(id,account_id,department_id) VALUES(?,?,?)',(str(uuid.uuid4()),actor,unit))
 con.executemany('INSERT INTO b21entry(id,name,owner,department_id) VALUES(?,?,?,?)',[(ownrow,'VISIBLE_NATIVE_CONTEXT',LOGIN,unit),(foreignrow,'FOREIGN_NATIVE_CONTEXT',LOGIN,other)])
 con.commit();con.close()
 receipt['phase']='start-native'
 log=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(120):
  if server.poll() is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5) as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('native readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True);receipt['browser_version']=browser.version
 context=browser.new_context(viewport={'width':1440,'height':900},accept_downloads=True)
 context.route('**/*',lambda route:route.continue_() if route.request.url.startswith(ORIGIN+'/') else route.abort())
 context.on('response',lambda r:network.append({'kind':'drilldown' if '/drilldown?'in r.url else 'entity' if '/ui/catalog/B21Entry/'in r.url else 'other','status':r.status}) if '/drilldown?'in r.url or '/ui/catalog/B21Entry/'in r.url else None)
 page=context.new_page();page.on('pageerror',lambda e:errors.append(type(e).__name__))
 page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(LOGIN);page.locator('input[name="password"]').fill(PASSWORD);page.locator('button[type="submit"]').click();page.wait_for_timeout(250)
 spoof=urllib.parse.urlencode({'Actor':'client-other','Unit':other,'actor':'client-other','__run':'1'})
 receipt['phase']='context-report'
 response=page.goto(ORIGIN+'/ui/report/B21Context?'+spoof,wait_until='domcontentloaded');require('context_native_http200',response.status==200)
 text=page.locator('body').inner_text();require('context_actual_own_rows', 'VISIBLE_NATIVE_CONTEXT'in text and 'FOREIGN_NATIVE_CONTEXT'not in text);require('context_inputs_absent',page.locator('[name="Actor"],[name="Unit"]').count()==0)
 page.screenshot(path=str(W/'01-trusted-context.png'),full_page=True)
 receipt['phase']='queryless-result'
 response=page.goto(ORIGIN+'/ui/report/B21Result?'+spoof,wait_until='domcontentloaded');require('queryless_native_viewer',response.status==200 and '/ui/tabular/'in page.url)
 text=page.locator('body').inner_text();require('queryless_server_uuid_and_actor',unit in text and LOGIN in text and other not in text and 'client-other'not in text)
 with page.expect_download(timeout=30000) as pending:page.locator('a[href$="/xlsx"]').click()
 xlsx=W/'trusted-context-b21.xlsx';pending.value.save_as(xlsx)
 import openpyxl
 f=openpyxl.load_workbook(xlsx,read_only=True,data_only=True);values=list(f.active.values);f.close();require('queryless_default_reader_exact_bound_values',values==[('Trusted unit',unit),('Trusted actor',LOGIN)])
 receipt['xlsx']={'sha256':hashlib.sha256(xlsx.read_bytes()).hexdigest(),'bytes':xlsx.stat().st_size,'rows':len(values),'reader':openpyxl.__version__,'read_only':True,'reset_dimensions_called':False}
 page.screenshot(path=str(W/'02-queryless-context.png'),full_page=True)
 receipt['phase']='typed-flat12'
 response=page.goto(ORIGIN+'/ui/report/B21Typed?'+spoof,wait_until='domcontentloaded');require('typed_native_http200',response.status==200)
 table=page.locator('table').filter(has=page.locator('tr.det')).first;require('flat12_no_service_column',table.locator('thead th').count()==13 and table.locator('tr.det').first.locator('td').count()==13);require('flat12_two_typed_actions',table.locator('a.report-drilldown').count()==2)
 text=table.inner_text();require('flat12_own_context', 'VISIBLE_NATIVE_CONTEXT'in text and 'FOREIGN_NATIVE_CONTEXT'not in text)
 page.screenshot(path=str(W/'03-flat12-actions.png'),full_page=True)
 links=table.locator('a.report-drilldown');hrefs=links.evaluate_all('(xs)=>xs.map(a=>a.getAttribute("href"))')
 require('typed_actions_fixed_server_route',all(x.startswith('/ui/report/B21Typed/drilldown?') for x in hrefs))
 for i,href in enumerate(hrefs):
  response=page.goto(ORIGIN+href,wait_until='domcontentloaded');require(f'typed_{i}_native_redirect_target',response.status==200 and '/ui/catalog/B21Entry/'+ownrow in page.url)
  require(f'typed_{i}_native_entity_loaded',page.locator('input[value="VISIBLE_NATIVE_CONTEXT"]').count()>0 or 'VISIBLE_NATIVE_CONTEXT'in page.locator('body').inner_text())
  if i==0:page.screenshot(path=str(W/'04-native-entity.png'),full_page=True)
 import re
 statuses=[]
 for line in (W/'server.log').read_text('utf-8',errors='replace').splitlines():
  if 'drilldown' in line:
   m=re.search(r'(?:status=|"status"\s*:\s*)(\d+)',line)
   if m:statuses.append(int(m.group(1)))
 receipt['redirect_evidence']={'server_drilldown_statuses':statuses,'browser_terminal_status_observer_not_used_for_intermediate_redirect':True}
 require('typed_route_authorized_303',statuses.count(303)>=2)
 require('typed_entity_http200',sum(x['kind']=='entity' and x['status']==200 for x in network)>=2)
 # Use a real UI anchor once in addition to ordinary native HTTP navigation.
 page.goto(ORIGIN+'/ui/report/B21Typed?'+spoof,wait_until='domcontentloaded');page.locator('a.report-drilldown').first.click();page.wait_for_timeout(500)
 require('actual_anchor_causes_entity_navigation',any('/ui/catalog/B21Entry/'+ownrow in f.url for p in context.pages for f in p.frames))
 require('JS0',not errors);receipt['status']='PASS'
except Exception as e:
 receipt['status']='FAIL';receipt['failure_type']=type(e).__name__;receipt['failed_check']=str(e) if isinstance(e,AssertionError) else None
 import traceback
 receipt['failure_line']=traceback.extract_tb(e.__traceback__)[-1].lineno
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
   if p.exists():
    try:p.unlink()
    except OSError:receipt['cleanup']['db_delete_failed']=False
 receipt['cleanup']['own_auth_db_removed']=not DB.exists();receipt['page_errors']=len(errors)
 (W/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');print(json.dumps(receipt,ensure_ascii=True))
 if receipt['status']!='PASS' or not all(receipt['cleanup'].values()):raise SystemExit(1)
