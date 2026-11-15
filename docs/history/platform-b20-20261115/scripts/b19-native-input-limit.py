import datetime, hashlib, json, os, pathlib, secrets, socket, subprocess, time, urllib.request
from playwright.sync_api import sync_playwright

B=pathlib.Path(__file__).parent
M=json.loads((B/'manifest-1115b19-tabular-input-limit-DEV.json').read_text('utf-8'))
SDK=pathlib.Path(M['artifacts']['windows']['path'])
assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b19-native-input-limit-attempt1';W.mkdir(exist_ok=False)
P=W/'config';DB=W/'own-native.sqlite';PORT=18568;ORIGIN=f'http://127.0.0.1:{PORT}'
LOGIN='own_b19_report';PASSWORD='B19!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for key in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(key,None)
FLAGS=subprocess.CREATE_NO_WINDOW
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'sdk_sha256':M['artifacts']['windows']['sha256'],'sdk_platform':'windows-amd64','scope':'own fresh synthetic SQLite; no DB copies/shared/config changes','native_browser':'Chromium109','started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'input_raw_rows':8,'input_distinct_documents':7,'cases':{},'checks':{},'cleanup':{},'phase':'fixture','go_started':False,'browser_os_network_isolated':False,'browser_request_boundary':'own context loopback allowlist','new_snapshot_observation':'publication via redirect/viewer requests only; private cache cardinality is covered by HTTP regression, not inspected in CLI'}
server=serverlog=driver=browser=context=page=None;errors=[];responses=[]

def require(name,condition):
 receipt['checks'][name]=bool(condition)
 if not condition:raise AssertionError(name)
def write(name,text):
 p=P/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text,encoding='utf-8')
def cli(args,stdin=None):
 p=subprocess.run([str(SDK),*args],input=stdin,env=ENV,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=FLAGS,timeout=60)
 (W/('cli-'+str(len(list(W.glob('cli-*.log'))))+'.log')).write_bytes(p.stdout)
 if p.returncode:raise RuntimeError('native CLI fixture failed')
def stop_app():
 global server,serverlog,context,page
 if context:context.close();context=None;page=None
 if server:
  server.terminate()
  try:server.wait(timeout=8)
  except subprocess.TimeoutExpired:server.kill();server.wait(timeout=3)
  server=None
 if serverlog:serverlog.close();serverlog=None
def on_response(r):
 path=r.url.split('?',1)[0]
 if path.startswith(ORIGIN+'/ui/report/') or path.startswith(ORIGIN+'/ui/tabular/'):
  responses.append({'method':r.request.method,'status':r.status,'kind':'viewer'if'/ui/tabular/'in path else'report','redirect_to_viewer':r.headers.get('location','').startswith('/ui/tabular/')})
def start_app(limit):
 global server,serverlog,context,page
 stop_app()
 write(pathlib.Path('config/app.yaml'),f'name: OwnB19InputLimit\nversion: "1.0"\nlimits:\n  report_max_rows: {limit}\n')
 serverlog=(W/f'server-cap-{limit}.log').open('xb')
 server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=serverlog,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(100):
  if server.poll()is not None:raise RuntimeError('native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('native readiness')
 context=browser.new_context(viewport={'width':1440,'height':900})
 context.route('**/*',lambda r:r.continue_()if r.request.url.startswith(ORIGIN+'/')else r.abort())
 page=context.new_page();page.on('pageerror',lambda e:errors.append(type(e).__name__));page.on('response',on_response)
 page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(LOGIN);page.locator('input[name="password"]').fill(PASSWORD)
 page.locator('button[type="submit"]').click();page.wait_for_timeout(200)
 require(f'cap{limit}_login_native',not page.url.endswith('/login'))
def report_request(report,method,query='',guard='off'):
 if method=='GET':return page.goto(ORIGIN+'/ui/report/'+report+'?__run=1'+query,wait_until='domcontentloaded')
 page.goto(ORIGIN+'/ui/report/'+report,wait_until='domcontentloaded')
 page.locator('input[name="Guard"]').fill(guard)
 with page.expect_response(lambda r:r.request.method=='POST'and'/ui/report/'+report in r.url)as pending:
  page.get_by_role('button',name='Сформировать',exact=True).click()
 page.wait_for_load_state('domcontentloaded')
 return pending.value
def documents():
 cells=page.locator('.doc-content table td').all_text_contents()
 return [x.strip()for x in cells if x.strip().startswith('DOC-')]
def assert_full(case,method):
 before=len(responses);r=report_request('NativeTabular',method)
 page.locator('.doc-content table tr').last.wait_for()
 docs=documents();require(case+'_seven_documents',docs==['DOC-'+c for c in'ABCDEFG'])
 require(case+'_native_viewer', '/ui/tabular/'in page.url)
 require(case+'_post303'if method=='POST'else case+'_get303',any(x['method']==method and x['status']==303 and x['redirect_to_viewer']for x in responses[before:]))
 receipt['cases'][case]={'method':method,'documents':len(docs),'last_document':'DOC-G','redirect':303,'viewer_http':200,'raw_input_rows':8}
def assert_refused(case,method,query='',guard='off'):
 before=len(responses);r=report_request('NativeTabular',method,query,guard)
 body=page.locator('body').inner_text()
 require(case+'_422_visible',r.status==422 and'report_max_rows'in body)
 require(case+'_no_module_error','MODULE_MUST_NOT_RUN'not in body)
 require(case+'_no_redirect',not r.headers.get('location'))
 require(case+'_no_viewer_published',not any(x['kind']=='viewer'or x['redirect_to_viewer']for x in responses[before:]))
 require(case+'_no_viewer_or_export_dom',page.locator('iframe[src*="/ui/tabular/"],a[href*="/ui/tabular/"]').count()==0 and'/ui/report/'in page.url)
 receipt['cases'][case]={'method':method,'http':r.status,'visible_limit_message':True,'new_viewer_redirects':0,'new_viewer_requests':0,'new_snapshot_publication_observed':False,'module_throw_guard':guard=='deny'}

try:
 with socket.socket()as probe:probe.bind(('127.0.0.1',PORT))
 query="SELECT 1 AS Ord, 'DOC-A' AS DocName UNION ALL SELECT 2, 'DOC-A' UNION ALL SELECT 3, 'DOC-B' UNION ALL SELECT 4, 'DOC-C' UNION ALL SELECT 5, 'DOC-D' UNION ALL SELECT 6, 'DOC-E' UNION ALL SELECT 7, 'DOC-F' UNION ALL SELECT 8, 'DOC-G' ORDER BY Ord"
 tiny="SELECT 1 AS Ord, 'DOC-A' AS DocName UNION ALL SELECT 2, 'DOC-A' ORDER BY Ord"
 for name in ['NativeTabular','OrdinaryQuery']:
  text=f'name: {name}\ntitle: B19 '+('Табличный документ'if name=='NativeTabular'else'Обычный отчёт')+'\nquery: '+json.dumps(query)+'\nparams:\n  - {name: Guard, type: string, label: Контроль модуля, default: off}\n'
  if name=='NativeTabular':text+='result_proc: Build\nvariants:\n  - name: Tiny\n    query: '+json.dumps(tiny)+'\n'
  write(pathlib.Path('reports')/(name+'.yaml'),text)
 write(pathlib.Path('src/NativeTabular.rep.os'),'''Функция Build()
 Если Параметры.Guard="deny" Тогда ВызватьИсключение "MODULE_MUST_NOT_RUN";КонецЕсли;
 Д=Новый ТабличныйДокумент;Н=1;Предыдущий="";
 Для Каждого Стр Из Результат Цикл
  Если Стр.DocName<>Предыдущий Тогда
   Д.Ячейка(Н,1).Текст=Стр.DocName;Н=Н+1;Предыдущий=Стр.DocName;
  КонецЕсли;
 КонецЦикла;Возврат Д;
КонецФункции
''')
 write(pathlib.Path('config/app.yaml'),'name: OwnB19InputLimit\nversion: "1.0"\n')
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',LOGIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PASSWORD+'\n').encode())
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True)
 receipt['browser_version']=browser.version
 for limit in [0,8]:
  receipt['phase']=f'cap{limit}-full';start_app(limit)
  for method in ['GET','POST']:assert_full(f'cap{limit}_{method}',method)
  if limit==8:
   page.screenshot(path=str(W/'cap8-seven-documents.png'),full_page=True)
   r=report_request('NativeTabular','GET','&Guard=deny')
   require('uncropped_module_guard_is_live',r.status==500 and'MODULE_MUST_NOT_RUN'in page.locator('body').inner_text())
   receipt['cases']['module_guard_control']={'http':r.status,'module_throw_observed':True,'limit':8}
 receipt['phase']='cap3-refusal';start_app(3)
 assert_refused('cap3_GET','GET');page.screenshot(path=str(W/'cap3-get-refused.png'),full_page=True)
 assert_refused('cap3_POST','POST');page.screenshot(path=str(W/'cap3-post-refused.png'),full_page=True)
 assert_refused('cap3_module_not_called','POST',guard='deny')
 receipt['phase']='same-server-no-stale-viewer'
 page.goto(ORIGIN+'/ui/report/NativeTabular?__run=1&__variant=Tiny',wait_until='domcontentloaded')
 old=page.url;require('prior_snapshot_native', '/ui/tabular/'in old and documents()==['DOC-A'])
 assert_refused('no_stale_viewer_POST','POST')
 old_response=context.request.get(old)
 require('prior_legitimate_snapshot_not_destroyed',old_response.status==200)
 receipt['cases']['no_stale_viewer_POST']['prior_valid_snapshot_existed']=True
 receipt['cases']['no_stale_viewer_POST']['prior_snapshot_http_after_refusal']=old_response.status
 page.screenshot(path=str(W/'no-stale-viewer.png'),full_page=True)
 receipt['phase']='ordinary-warning-policy'
 for method in ['GET','POST']:
  r=report_request('OrdinaryQuery',method)
  rows=page.locator('main table tbody tr');texts=rows.all_text_contents()
  warning=page.locator('main .card').filter(has_text='report_max_rows')
  visible_warning=warning.count()>0 and warning.first.is_visible()
  require('ordinary_'+method+'_200_warning',r.status==200 and visible_warning)
  require('ordinary_'+method+'_three_raw_rows',len(texts)==3 and all('DOC-G'not in x for x in texts))
  receipt['cases']['ordinary_'+method]={'http':r.status,'visible_warning':visible_warning,'raw_output_rows':len(texts),'distinct_output_documents':2}
 page.screenshot(path=str(W/'ordinary-query-warning.png'),full_page=True)
 require('JS0',not errors);receipt['status']='PASS'
except Exception as e:
 receipt['status']='FAIL';receipt['failure_type']=type(e).__name__;receipt['failed_check']=str(e)if isinstance(e,AssertionError)else None
finally:
 try:
  if context:context.close();context=None
  receipt['cleanup']['context']=True
 except Exception:receipt['cleanup']['context']=False
 try:
  if browser:browser.close()
  receipt['cleanup']['browser']=True
 except Exception:receipt['cleanup']['browser']=False
 try:
  if driver:driver.stop()
  receipt['cleanup']['driver']=True
 except Exception:receipt['cleanup']['driver']=False
 try:stop_app();receipt['cleanup']['app']=True
 except Exception:receipt['cleanup']['app']=False
 try:
  with socket.create_connection(('127.0.0.1',PORT),timeout=.2):receipt['cleanup']['port_closed']=False
 except OSError:receipt['cleanup']['port_closed']=True
 receipt['cleanup']['tunnel_not_used']=True
 if receipt['cleanup']['app']:
  for path in [DB,pathlib.Path(str(DB)+'-wal'),pathlib.Path(str(DB)+'-shm')]:
   if path.exists():path.unlink()
 receipt['cleanup']['own_synthetic_auth_db_removed']=not DB.exists()
 receipt['page_errors']=len(errors);receipt['safe_http_trace']=responses
 receipt['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat()
 (W/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
 print(json.dumps(receipt,ensure_ascii=False))
 if receipt['status']!='PASS'or not all(receipt['cleanup'].values()):raise SystemExit(1)
