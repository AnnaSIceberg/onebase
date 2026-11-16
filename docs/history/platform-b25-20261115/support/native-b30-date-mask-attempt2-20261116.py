import datetime
import hashlib,json,os,pathlib,secrets,socket,sqlite3,subprocess,time,urllib.request,uuid
import openpyxl
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent
M=json.loads((B/'b30-date-mask-build-20261116/manifest.json').read_text('utf-8'))
assert M['status']=='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
SDK=pathlib.Path(M['artifacts']['windows']['path'])
with SDK.open('rb') as f:assert hashlib.file_digest(f,'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b30-native-date-mask-attempt2-20261116';W.mkdir(exist_ok=False)
P=W/'config';DB=W/'own-native.sqlite';PORT=18587;ORIGIN=f'http://127.0.0.1:{PORT}'
LOGIN='own_b30_date_user';PW='B29!'+secrets.token_urlsafe(24)
ENV=dict(os.environ,GOMAXPROCS='2')
for k in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(k,None)
server=log=driver=browser=context=con=None;errors=[]
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'sdk_sha256':M['artifacts']['windows']['sha256'],'checks':{},'cleanup':{},'shared_changes':False,'db_copies':0,'go_started':False,'network_boundary':'exact own loopback browser-context allowlist; no OS isolation claim'}
def require(name,ok):
 receipt['checks'][name]=bool(ok)
 if not ok:raise AssertionError(name)
def write(name,text):
 p=P/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text,encoding='utf-8')
def cli(args,stdin=None):
 p=subprocess.run([str(SDK),*args],env=ENV,input=stdin,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=90)
 (W/f'cli-{len(list(W.glob("cli-*")))}.log').write_bytes(p.stdout)
 if p.returncode:raise RuntimeError('own native CLI setup failed')
def check_book(path,key,marker):
 a=openpyxl.load_workbook(path,read_only=False,data_only=True);b=openpyxl.load_workbook(path,read_only=True,data_only=True)
 normal=list(a.active.values);stream=list(b.active.values);a.close();b.close()
 require(key+'_default_reader_equals_normal',normal==stream)
 text=json.dumps(stream,ensure_ascii=False)
 require(key+'_canonical_selected_result',marker in text and ('NATIVE_ROW_VISIT'not in text if marker=='NATIVE_ROW_OFFICE'else True))
 receipt.setdefault('xlsx',{})[key]={'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'rows':len(stream),'reader':openpyxl.__version__,'reset_dimensions_called':False,'forced_bounds':False}
try:
 with socket.socket() as s:s.bind(('127.0.0.1',PORT))
 write('config/app.yaml',"name: B30-native-date\nversion: '1.0'\nui:\n short_date_input: true\n")
 write('catalogs/B30DateRow.yaml','name: B30DateRow\nfields:\n - {name: Name, type: string}\n - {name: Day, type: date}\n - {name: Moment, type: date}\n')
 write('forms/b30daterow/B30DateRow.form.yaml','''schema: onebase.form/v1
form: {name: ФормаОбъекта, kind: object, entity: B30DateRow, enter_submits_form: true}
attributes:
 - {name: Объект, type: CatalogRef.B30DateRow, save: true, main: true}
elements:
 - {kind: ПолеВвода, name: NameField, data_path: Объект.Name}
 - {kind: ПолеДаты, name: DayField, data_path: Объект.Day}
 - {kind: ПолеВвода, name: MomentField, data_path: Объект.Moment}
''')
 write('reports/B30Date.yaml','''name: B30Date
title: Date mask native report
params:
 - {name: Day, type: date}
 - {name: Moment, type: datetime}
query: ВЫБРАТЬ 1 КАК Marker
''')
 for role in ['ОТК','ОператорКЦ']:
  write('roles/'+role+'.yaml','name: '+role+'\npermissions:\n catalogs:\n  B30DateRow: [read, write]\n reports:\n  B30Date: [run]\n')
 cli(['check','--project',str(P)]);require('public_configuration_check',True)
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add','own_b30_bootstrap','--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(secrets.token_urlsafe(32)+'\n').encode())
 for login,role in [('own_b30_otk','ОТК'),('own_b30_operator','ОператорКЦ')]:
  cli(['user','add',login,'--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW+'\n').encode())
  cli(['user','role','assign',login,role,'--project',str(P),'--sqlite',str(DB)])
 con=sqlite3.connect(DB)
 require('fixture_users_nonadmin',con.execute('SELECT count(*) FROM _users WHERE login IN (?,?) AND is_admin=0',('own_b30_otk','own_b30_operator')).fetchone()[0]==2)
 con.close();con=None
 log=(W/'server.log').open('wb')
 server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=subprocess.CREATE_NO_WINDOW)
 for _ in range(120):
  if server.poll()is not None:raise RuntimeError('own native server exited')
  try:
   with urllib.request.urlopen(ORIGIN+'/login',timeout=.5)as r:
    if r.status==200:break
  except Exception:time.sleep(.1)
 else:raise TimeoutError('native readiness')
 driver=sync_playwright().start();browser=driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe',headless=True)
 receipt['browser_version']=browser.version
 for login in ['own_b30_otk','own_b30_operator']:
  context=browser.new_context(viewport={'width':1440,'height':900},accept_downloads=True,permissions=['clipboard-read','clipboard-write'])
  context.route('**/*',lambda r:r.continue_()if r.request.url.startswith(ORIGIN+'/')else r.abort())
  page=context.new_page();page.on('pageerror',lambda e:errors.append(type(e).__name__))
  page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(login);page.locator('input[name="password"]').fill(PW);page.locator('button[type="submit"]').click()
  requests=[];page.on('request',lambda r:requests.append((r.method,r.url)))
  page.goto(ORIGIN+'/ui/report/B30Date',wait_until='domcontentloaded')
  day=page.locator('input[name="Day"]');day.focus();require(login+'_report_date_mask',day.input_value()=='__.__.____')
  day.press_sequentially('16');require(login+'_report_caret_skips_dot',day.input_value()=='16.__.____'and day.evaluate('(e)=>e.selectionStart')==3)
  page.screenshot(path=str(W/(login+'-live-mask.png')),full_page=True)
  day.press_sequentially('112026');day.press('Enter');require(login+'_report_date_ISO',day.input_value()=='2026-11-16')
  moment=page.locator('input[name="Moment"]');moment.focus();moment.press_sequentially('16112026235958');moment.press('Enter');require(login+'_report_datetime_seconds',moment.input_value()=='2026-11-16T23:59:58')
  day.focus();day.press('Control+a');day.press_sequentially('1611');day.press('Enter');require(login+'_partial_error',day.evaluate('(el)=>el.validationMessage').find('полностью')>=0)
  day.press('Escape')
  # Save and restore clipboard solely inside the isolated page, never log it.
  page.evaluate('async()=>{window.savedNativeMaskClipboard=await navigator.clipboard.readText()}')
  try:
   for value in ['16.11.2026','16112026']:
    page.evaluate('(s)=>navigator.clipboard.writeText(s)',value)
    day.focus();day.press('Control+a');day.press('Delete');day.press('Control+v')
    require(login+'_paste_mask_'+value,day.input_value()=='16.11.2026');day.press('Enter')
    require(login+'_paste_ISO_'+value,day.input_value()=='2026-11-16')
  finally:page.evaluate('()=>navigator.clipboard.writeText(window.savedNativeMaskClipboard)')
  day.focus();day.press('Control+a');day.press_sequentially('16112026');day.press('Alt+ArrowDown')
  require(login+'_native_calendar_kept',day.get_attribute('type')=='date')
  page.keyboard.press('Escape');page.locator('h2').first.click()
  require(login+'_Enter_never_runs_report',not any(method=='POST'and '/ui/report/B30Date'in url for method,url in requests))
  page.screenshot(path=str(W/(login+'-report.png')),full_page=True)
  page.goto(ORIGIN+'/ui/catalog/b30daterow/new',wait_until='domcontentloaded')
  require(login+'_managed_form',page.locator('input[name="Day"]').count()==1 and page.locator('input[name="Moment"]').count()==1)
  name=page.locator('input[name="Name"]');name.fill('NATIVE_DATE_MASK_'+login)
  day=page.locator('input[name="Day"]');day.focus();day.press_sequentially('16112026');day.press('Enter');require(login+'_object_date_ISO',day.input_value()=='2026-11-16')
  moment=page.locator('input[name="Moment"]');moment.focus();moment.press_sequentially('17112026123456');moment.press('Tab')
  require(login+'_object_datetime_blur',moment.input_value()=='2026-11-17T12:34:56')
  page.screenshot(path=str(W/(login+'-object.png')),full_page=True)
  require(login+'_date_did_not_save_object',not any(method=='POST'and '/ui/catalog/b30daterow/new'in url for method,url in requests))
  # Save explicitly while the last date editor is active: form integration must
  # commit the mask before serialization, independently of Enter navigation.
  moment.focus();moment.press('Control+a');moment.press_sequentially('18112026112233')
  save=page.get_by_role('button',name='Записать и закрыть',exact=True)
  if save.count()==0:save=page.get_by_role('button',name='Записать',exact=True)
  save.click();page.wait_for_timeout(300)
  require(login+'_explicit_save_no_error','__.__.____'not in page.locator('body').inner_text())
  con=sqlite3.connect(DB);row=con.execute('SELECT day,moment FROM b30daterow WHERE name=?',('NATIVE_DATE_MASK_'+login,)).fetchone();con.close();con=None
  receipt.setdefault('synthetic_saved_dates',{})[login]=list(row)if row else None
  require(login+'_saved_dates_roundtrip',row is not None and datetime.datetime.fromisoformat(str(row[0]).replace('Z','+00:00')).astimezone().strftime('%Y-%m-%d')=='2026-11-16'and datetime.datetime.fromisoformat(str(row[1]).replace('Z','+00:00')).astimezone().strftime('%Y-%m-%dT%H:%M:%S')=='2026-11-18T11:22:33')
  require(login+'_reopened_form_dates',page.locator('input[name="Day"]').input_value()=='2026-11-16'and page.locator('input[name="Moment"]').input_value()=='2026-11-18T11:22:33')
  page.goto(ORIGIN+'/ui/catalog/b30daterow',wait_until='domcontentloaded')
  panel=page.locator('details').filter(has=page.locator('input[name="f.Day.from"]'))
  if panel.get_attribute('open') is None:panel.locator('summary').click()
  flt=page.locator('input[name="f.Day.from"]');flt.focus();require(login+'_list_filter_mask',flt.input_value()=='__.__.____')
  flt.press_sequentially('16112026');flt.press('Enter');require(login+'_list_filter_ISO',flt.input_value()=='2026-11-16')
  require(login+'_list_Enter_no_filter_run','f.Day.from'not in page.url)
  page.screenshot(path=str(W/(login+'-filter.png')),full_page=True)
  context.close();context=None
 require('JS0',not errors);receipt['status']='PASS'
except Exception as e:
 import traceback
 receipt.update(status='FAIL',failure_type=type(e).__name__,failure_line=traceback.extract_tb(e.__traceback__)[-1].lineno,failed_check=str(e)if isinstance(e,AssertionError)else None)
finally:
 if con is not None:
  try:con.close()
  except Exception:pass
 for key,obj in [('context',context),('browser',browser)]:
  try:
   if obj:obj.close()
   receipt['cleanup'][key]=True
  except Exception:receipt['cleanup'][key]=False
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
 if receipt['cleanup']['app']:
  for p in [DB,pathlib.Path(str(DB)+'-wal'),pathlib.Path(str(DB)+'-shm')]:
   assert p.resolve().is_relative_to(W.resolve());p.unlink(missing_ok=True)
 receipt['cleanup']['own_auth_db_removed']=not DB.exists();receipt['page_errors']=len(errors)
 (W/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
 print(json.dumps({'status':receipt['status'],'checks':len(receipt['checks']),'cleanup':receipt['cleanup'],'failed_check':receipt.get('failed_check'),'failure_type':receipt.get('failure_type')}))
 if receipt['status']!='PASS'or not all(receipt['cleanup'].values()):raise SystemExit(1)
