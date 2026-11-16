import hashlib,json,os,pathlib,secrets,socket,sqlite3,subprocess,time,urllib.request,uuid
import openpyxl
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent
M=json.loads((B/'b29-enum-build-20261116/manifest.json').read_text('utf-8'))
assert M['status']=='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
SDK=pathlib.Path(M['artifacts']['windows']['path'])
with SDK.open('rb') as f:assert hashlib.file_digest(f,'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b29-native-enum-attempt1-20261116';W.mkdir(exist_ok=False)
P=W/'config';DB=W/'own-native.sqlite';PORT=18586;ORIGIN=f'http://127.0.0.1:{PORT}'
LOGIN='own_b29_enum_user';PW='B29!'+secrets.token_urlsafe(24)
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
 write('config/app.yaml',"name: B29-native-enum\nversion: '1.0'\n")
 write('enums/ServiceChoice.yaml','''name: ServiceChoice
values:
 - name: Visit
   titles: {ru: Выезд, en: On site}
 - name: Office
   titles: {ru: В офисе, en: At office}
''')
 write('catalogs/B29EnumRow.yaml','name: B29EnumRow\nfields:\n - {name: Name, type: string}\n - {name: Service, type: enum:ServiceChoice}\n')
 params='params:\n - {name: Pick, type: enum:ServiceChoice, default: Visit}\n - {name: Ref, type: reference:B29EnumRow}\n'
 write('reports/B29Enum.yaml','name: B29Enum\ntitle: Native enumeration report\n'+params+'query: ВЫБРАТЬ Name ИЗ Справочник.B29EnumRow ГДЕ Service=&Pick\n')
 module='''Функция Build()
 Д=Новый ТабличныйДокумент;
 Д.Ячейка(1,1).Текст="NATIVE_ENUM_PARAMETER";
 Если Параметры.Pick=Неопределено Тогда
  Д.Ячейка(1,2).Текст="EMPTY_SELECTION";
 Иначе
  Д.Ячейка(1,2).Текст=Строка(Параметры.Pick);
 КонецЕсли;
 Возврат Д;
КонецФункции
'''
 for name in ['B29EnumTabular','B29EnumQueryTabular']:
  write('reports/'+name+'.yaml','name: '+name+'\n'+params+'result_proc: Build\n'+('query: ВЫБРАТЬ 1 КАК Marker\n' if name=='B29EnumQueryTabular' else ''))
  write('src/'+name+'.rep.os',module)
 write('roles/B29EnumRunner.yaml','name: B29EnumRunner\npermissions:\n catalogs:\n  B29EnumRow: [read]\n reports:\n  B29Enum: [run]\n  B29EnumTabular: [run]\n  B29EnumQueryTabular: [run]\n')
 cli(['check','--project',str(P)]);require('public_configuration_check',True)
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',LOGIN,'--password-stdin','--project',str(P),'--sqlite',str(DB)],(PW+'\n').encode())
 cli(['user','role','assign',LOGIN,'B29EnumRunner','--project',str(P),'--sqlite',str(DB)])
 con=sqlite3.connect(DB)
 require('fixture_user_nonadmin',con.execute('SELECT is_admin FROM _users WHERE login=?',(LOGIN,)).fetchone()[0]==0)
 for value,marker in [('Visit','NATIVE_ROW_VISIT'),('Office','NATIVE_ROW_OFFICE')]:
  con.execute('INSERT INTO b29enumrow(id,name,service) VALUES(?,?,?)',(str(uuid.uuid4()),marker,value))
 con.commit();con.close();con=None
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
 context=browser.new_context(viewport={'width':1440,'height':900},accept_downloads=True)
 context.route('**/*',lambda r:r.continue_()if r.request.url.startswith(ORIGIN+'/')else r.abort())
 page=context.new_page();page.on('pageerror',lambda e:errors.append(type(e).__name__))
 page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(LOGIN);page.locator('input[name="password"]').fill(PW);page.locator('button[type="submit"]').click()
 page.goto(ORIGIN+'/ui/report/B29Enum',wait_until='domcontentloaded')
 select=page.locator('select[name="Pick"]')
 require('native_enum_select',select.count()==1 and page.locator('input[type="text"][name="Pick"]').count()==0)
 require('native_metadata_order_and_empty',select.locator('option').evaluate_all('(xs)=>xs.map(x=>x.value)')==['','Visit','Office'])
 require('native_russian_labels',select.locator('option').all_text_contents()[1:]==['Выезд','В офисе'])
 require('native_default_visit_selected',select.input_value()=='Visit')
 require('reference_native_picker_preserved',page.locator('select[name="Ref"][data-ref-entity="B29EnumRow"]').count()==1 and page.locator('[data-ob-ref-picker="rp-Ref"]').count()==1)
 page.screenshot(path=str(W/'01-enum-default-ru.png'),full_page=True)
 select.select_option('Office');page.get_by_role('button',name='Сформировать',exact=True).click();page.wait_for_load_state('domcontentloaded')
 require('actual_ui_selected_office_result','NATIVE_ROW_OFFICE'in page.locator('td').all_text_contents()and'NATIVE_ROW_VISIT'not in page.locator('td').all_text_contents())
 require('actual_ui_current_value_kept',page.locator('select[name="Pick"]').input_value()=='Office')
 page.screenshot(path=str(W/'02-enum-selected-result.png'),full_page=True)
 direct=context.request.get(ORIGIN+'/ui/report/B29Enum/excel?Pick=Office');require('direct_xlsx200',direct.status==200)
 artifact=W/'ordinary-direct.xlsx';artifact.write_bytes(direct.body());check_book(artifact,'direct','NATIVE_ROW_OFFICE')
 pdf=context.request.get(ORIGIN+'/ui/report/B29Enum/pdf?Pick=Office');require('direct_pdf200',pdf.status==200 and pdf.body().startswith(b'%PDF-'))
 page.get_by_role('link',name='Excel',exact=True).first.click();require('actual_toolbar_background_job','/ui/export-jobs/'in page.url)
 page.locator('a[href$="/download"]').wait_for(state='visible',timeout=30000)
 with page.expect_download(timeout=30000)as pending:page.locator('a[href$="/download"]').click()
 artifact=W/'ordinary-background.xlsx';pending.value.save_as(artifact);check_book(artifact,'background','NATIVE_ROW_OFFICE')
 page.goto(ORIGIN+'/ui/report/B29Enum?__run=1&Pick=Office',wait_until='domcontentloaded')
 page.locator('select[name="Pick"]').select_option('');page.get_by_role('button',name='Сформировать',exact=True).click();page.wait_for_load_state('domcontentloaded')
 require('empty_selection_not_defaulted',page.locator('select[name="Pick"]').input_value()==''and not any('NATIVE_ROW_'in text for text in page.locator('td').all_text_contents()))
 for name in ['B29Enum','B29EnumTabular','B29EnumQueryTabular']:
  bad=context.request.get(ORIGIN+'/ui/report/'+name+'?__run=1&Pick=Invented');require(name+'_invalid_HTTP400',bad.status==400)
 for suffix in ['excel','pdf']:
  bad=context.request.get(ORIGIN+'/ui/report/B29Enum/'+suffix+'?Pick=Invented');require('invalid_'+suffix+'_HTTP400',bad.status==400)
 good=context.request.get(ORIGIN+'/api/v2/report/B29Enum?Pick=Office');require('native_REST_canonical200',good.status==200 and 'NATIVE_ROW_OFFICE'in good.text()and'NATIVE_ROW_VISIT'not in good.text())
 bad=context.request.get(ORIGIN+'/api/v2/report/B29Enum?Pick=Invented');require('native_REST_invalid400',bad.status==400)
 for name in ['B29EnumTabular','B29EnumQueryTabular']:
  response=page.goto(ORIGIN+'/ui/report/'+name+'?__run=1&Pick=Office',wait_until='domcontentloaded')
  require(name+'_native_snapshot_canonical',response.status==200 and '/ui/tabular/'in page.url and 'Office'in page.locator('body').inner_text())
  with page.expect_download(timeout=30000)as pending:page.locator('a[href$="/xlsx"]').click()
  artifact=W/(name+'.xlsx');pending.value.save_as(artifact);check_book(artifact,name,'Office')
  response=page.goto(ORIGIN+'/ui/report/'+name+'?__run=1&Pick=',wait_until='domcontentloaded')
  require(name+'_native_empty_snapshot',response.status==200 and 'EMPTY_SELECTION'in page.locator('body').inner_text())
 page.screenshot(path=str(W/'03-tabular-empty-selection.png'),full_page=True)
 page.goto(ORIGIN+'/ui/report/B29Enum/export/excel?Pick=Invented',wait_until='domcontentloaded')
 require('invalid_background_has_job_page','/ui/export-jobs/'in page.url)
 for _ in range(100):
  status=context.request.get(page.url)
  if 'canonical enumeration value'in status.text():break
  time.sleep(.1)
 else:raise AssertionError('invalid_background_explicit_error')
 require('invalid_background_no_file','/download'not in status.text())
 page.goto(ORIGIN+'/ui/report/B29Enum',wait_until='domcontentloaded')
 status=page.evaluate("""async()=>{const r=await fetch('/ui/profile/lang',{method:'POST',body:new URLSearchParams({lang:'en'})});return r.status;}""")
 require('public_own_language_change',status==200)
 page.reload(wait_until='domcontentloaded')
 require('native_english_labels',page.locator('select[name="Pick"] option').all_text_contents()[1:]==['On site','At office'])
 require('native_english_values_canonical',page.locator('select[name="Pick"] option').evaluate_all('(xs)=>xs.map(x=>x.value)')==['','Visit','Office'])
 page.screenshot(path=str(W/'04-enum-default-en.png'),full_page=True)
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
