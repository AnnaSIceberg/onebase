import hashlib,json,os,pathlib,secrets,shutil,socket,subprocess,time,urllib.request,xml.etree.ElementTree as ET,zipfile
from playwright.sync_api import sync_playwright
B=pathlib.Path(__file__).parent
M=json.loads((B/'manifest-1115b20-xlsx-dimension-DEV.json').read_text('utf-8'))
SDK=pathlib.Path(M['artifacts']['windows']['path'])
assert hashlib.file_digest(SDK.open('rb'),'sha256').hexdigest()==M['artifacts']['windows']['sha256']
W=B/'b20-native-dimension-attempt1';W.mkdir(exist_ok=False)
SOURCE=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/perenos-task6-20261113/codex2/kpi-50k-repro')
assert not any(p.suffix in ['.sqlite','.db']for p in SOURCE.rglob('*')if p.is_file())
P=W/'config';shutil.copytree(SOURCE,P)
DB=W/'own-native.sqlite';PORT=18570;ORIGIN=f'http://127.0.0.1:{PORT}'
LOGIN='own_b20_xlsx';PASSWORD='B20!'+secrets.token_urlsafe(24)
ENV=dict(os.environ)
for key in ['DATABASE_URL','TEST_DATABASE_URL']:ENV.pop(key,None)
FLAGS=subprocess.CREATE_NO_WINDOW;errors=[];server=serverlog=browser=context=driver=None
receipt={'status':'RUNNING','runtime':M['runtime'],'source':M['source'],'sdk_platform':'windows-amd64','sdk_sha256':M['artifacts']['windows']['sha256'],'browser':'Chromium109','checks':{},'cases':{},'cleanup':{},'phase':'prepare','db_copies':0,'go_started':False,'shared_changes':False,'browser_os_network_isolated':False,'browser_request_boundary':'context own loopback allowlist','old_B18_exports_rewritten':False}
def require(name,condition):
 receipt['checks'][name]=bool(condition)
 if not condition:raise AssertionError(name)
def sha(p):
 with p.open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
def cli(args,stdin=None):
 p=subprocess.run([str(SDK),*args],input=stdin,env=ENV,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=FLAGS,timeout=60)
 (W/('cli-'+str(len(list(W.glob('cli-*'))))+'.log')).write_bytes(p.stdout)
 if p.returncode:raise RuntimeError('native CLI fixture failed')
try:
 with socket.socket()as probe:probe.bind(('127.0.0.1',PORT))
 layout=P/'printforms/КПИРазмер/ТаблицаРИ.layout.yaml'
 require('provided_layout_exact',sha(layout)=='512146a7d06cb1f18da1091e4019261656a7e474fc01a28670153223d0bed1a1')
 module=P/'forms/кпиразмер/объекта.form.os';original=module.read_text('utf-8')
 old_receipt=json.loads((B/'b18-native-kpi-attempt3/receipt.json').read_text('utf-8'))
 require('provided_module_unchanged_since_B18',hashlib.sha256(original.encode()).hexdigest()==old_receipt['provided_module_sha256'])
 custom={
 'Plain':('Plain','Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Значение=42;Объект.Результат=Д;','A1'),
 'Empty':('Empty','Д=Новый ТабличныйДокумент;Объект.Результат=Д;','A1'),
 'Auto51':('Авто51','Д=ДокументРазмера(31);Объект.Результат=Д;','A1:BO51'),
 'Sparse':('Sparse','Д=Новый ТабличныйДокумент;Д.Ячейка(830,67).Значение=42;Объект.Результат=Д;','A1:BO830'),
 'Span':('Span','Д=Новый ТабличныйДокумент;Д.Ячейка(3,4).Текст="Span";Д.Объединить(3,4,5,8);Объект.Результат=Д;','A1:H5'),
 'Styled':('Styled','Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Значение=42;Д.Ячейка(830,67).ЦветФона="#fffaf0";Объект.Результат=Д;','A1:BO830'),
 'Collapsed':('Collapsed','Д=Новый ТабличныйДокумент;С=Д.Область(1,1,1,4);С.Текст="42";Д.НачатьАвтогруппировкуСтрок();Д.Вывести(С,1,"parent",Ложь);Д.Вывести(С,2,"child",Истина);Д.Вывести(С,3,"detail",Истина);Д.ЗакончитьАвтогруппировкуСтрок();Объект.Результат=Д;','A1:D3')}
 form=P/'forms/кпиразмер/объекта.form.yaml';extra='';source_extra=''
 for name,(label,body,want)in custom.items():
  source_extra+='\nПроцедура '+name+'(Элемент)\n'+body+'\nКонецПроцедуры\n'
  extra+='\n- name: '+name+'\n  kind: Кнопка\n  title: {ru: '+label+'}\n  events: {Нажатие: '+name+'}\n'
 module.write_text(original+source_extra,encoding='utf-8');form.write_text(form.read_text('utf-8')+extra,encoding='utf-8')
 cli(['migrate','--project',str(P),'--sqlite',str(DB)])
 cli(['user','add',LOGIN,'--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(PASSWORD+'\n').encode())
 serverlog=(W/'server.log').open('wb');server=subprocess.Popen([str(SDK),'run','--project',str(P),'--sqlite',str(DB),'--host','127.0.0.1','--port',str(PORT)],env=ENV,stdout=serverlog,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,creationflags=FLAGS)
 for _ in range(100):
  if server.poll()is not None:raise RuntimeError('native server exited')
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
 page.goto(ORIGIN+'/login',wait_until='domcontentloaded');page.locator('input[name="login"]').fill(LOGIN);page.locator('input[name="password"]').fill(PASSWORD);page.locator('button[type="submit"]').click();page.wait_for_timeout(250)
 cases=[('large53893','СМ — 830×67','A1:BO830')]+[(name.lower(),label,want)for name,(label,body,want)in custom.items()]
 for name,label,want in cases:
  receipt['phase']=name
  page.goto(ORIGIN+'/ui/processor/КПИРазмер',wait_until='domcontentloaded')
  with page.expect_response(lambda r:r.request.method=='POST'and r.url.endswith('/form-event'),timeout=30000)as pending:page.get_by_role('button',name=label,exact=True).click()
  body=pending.value.json();require(name+'_native_ok',body.get('ok')is True and bool(body.get('tabularDocuments')))
  frame=page.frame_locator('iframe[data-ob-tabular-field]');frame.locator('.doc-content').wait_for(state='visible')
  with page.expect_download(timeout=30000)as pending:frame.locator('a[href$="/xlsx"]').click()
  path=W/(name+'-b20.xlsx');pending.value.save_as(path)
  with zipfile.ZipFile(path)as archive:xml=ET.fromstring(archive.read('xl/worksheets/sheet1.xml'))
  ns={'x':'http://schemas.openxmlformats.org/spreadsheetml/2006/main'};dim=xml.find('x:dimension',ns).attrib['ref']
  rows=xml.findall('x:sheetData/x:row',ns);cells=xml.findall('x:sheetData/x:row/x:c',ns)
  require(name+'_actual_dimension',dim==want)
  require(name+'_clean',page.evaluate('window._obFormDirty===false'))
  if name=='large53893':require('large_full_styled_rectangle',len(rows)==830 and len(cells)==55610)
  if name=='auto51':require('auto51_full_styled_rectangle',len(rows)==51 and len(cells)==3417)
  if name=='collapsed':require('collapsed_marker_preserved',any(r.attrib.get('collapsed')=='1'for r in rows))
  receipt['cases'][name]={'file':str(path),'sha256':sha(path),'bytes':path.stat().st_size,'dimension':dim,'xml_rows':len(rows),'xml_cells':len(cells),'merges':len(xml.findall('x:mergeCells/x:mergeCell',ns))}
  if name in ['large53893','auto51','span','collapsed']:page.screenshot(path=str(W/(name+'.png')),full_page=True)
 receipt['phase']='installed-default-reader'
 read=subprocess.run(['C:/Python314/python.exe',str(B/'b20-installed-xlsx-reader.py'),str(W)],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,creationflags=FLAGS,timeout=90)
 (W/'installed-reader.log').write_bytes(read.stdout);require('installed_default_reader_exit0',read.returncode==0)
 reader=W/'installed-default-reader.json';data=json.loads(reader.read_text('utf-8'));require('eight_new_exports_default_reader',len(data['files'])==8 and not data['reset_dimensions_called']and not data['forced_iteration_bounds'])
 require('all_filled_coordinates_and_typed_values_equal_normal',all(x['filled_coordinates_and_typed_values_equal']for x in data['files']))
 require('all_nonempty_default_counts_equal_normal',all(x['counts_equal']or x['empty_document_no_values_exception']for x in data['files']))
 receipt['installed_reader_receipt']={'path':str(reader),'sha256':sha(reader),'version':data['openpyxl'],'default_without_reset':True}
 require('JS0',not errors);receipt['status']='PASS'
except Exception as e:
 receipt['status']='FAIL';receipt['failure_type']=type(e).__name__;receipt['failed_check']=str(e)if isinstance(e,AssertionError)else None
finally:
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
 if serverlog:serverlog.close()
 try:
  with socket.create_connection(('127.0.0.1',PORT),timeout=.2):receipt['cleanup']['port_closed']=False
 except OSError:receipt['cleanup']['port_closed']=True
 receipt['cleanup']['tunnel_not_used']=True
 if receipt['cleanup']['app']:
  for path in [DB,pathlib.Path(str(DB)+'-wal'),pathlib.Path(str(DB)+'-shm')]:
   if path.exists():path.unlink()
 receipt['cleanup']['own_auth_db_removed']=not DB.exists();receipt['page_errors']=len(errors)
 (W/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');print(json.dumps(receipt,ensure_ascii=False))
 if receipt['status']!='PASS'or not all(receipt['cleanup'].values()):raise SystemExit(1)
