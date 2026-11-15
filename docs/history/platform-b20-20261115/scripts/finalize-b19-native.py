import datetime, hashlib, json, pathlib, re, shutil
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=B/'b19-native-input-limit-attempt2'
def sha(p):
 with pathlib.Path(p).open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
original=B/'manifest-1115b19-tabular-input-limit-DEV.json'
assert sha(original)=='94054e97d5cf6c879e6c7b9fc302a43f88a623096450a72962a5468e59837662'
m=load(original);native=load(W/'receipt.json');cleanup=load(B/'b19-native-cleanup-end.json');cache=load(B/'20261115-platform-build-cache-cleanup-b19.json')
assert native['status']=='PASS'and native['page_errors']==0 and all(native['checks'].values()) and len(native['checks'])==43 and all(native['cleanup'].values())
assert native['sdk_platform']=='windows-amd64'and native['sdk_sha256']==m['artifacts']['windows']['sha256']
assert cleanup['go_process_count']==cleanup['chrome_process_count']==cleanup['driver_process_count']==0 and cleanup['own_port_closed']
assert cache['go_before']==cache['go_after']==0 and cache['removed_file_bytes']==4185073390
for a in m['artifacts'].values():assert sha(a['path'])==a['sha256']
pg=pathlib.Path(m['tests']['pg']['receipt']);assert sha(pg)=='64f9697c05825f1a6f558023d04811b193ba3775d896bd9cb8fd58a23a26fd55'
wire=[]
for cap in [0,8,3]:
 rows=[]
 for line in (W/f'server-cap-{cap}.log').read_text('utf-8').splitlines():
  match=re.search(r'component=http method=(GET|POST) .*?route=/ui/(report/\{name\}|tabular/\{token\}) status=(\d+)',line)
  if match:rows.append({'method':match[1],'route_class':'report'if match[2].startswith('report')else'viewer','status':int(match[3])})
 wire.append({'configured_report_max_rows':cap,'metadata_only_native_http':rows,'source_log_sha256':sha(W/f'server-cap-{cap}.log')})
wirepath=W/'native-wire-summary.json';assert not wirepath.exists();wirepath.write_text(json.dumps(wire,indent=2)+'\n',encoding='utf-8')
assert sum(x['status']==422 for x in wire[2]['metadata_only_native_http'])==4
desktop=pathlib.Path('C:/Users/a.sevrjukova/Desktop/ТабДок-B19-входной-лимит-20261115');desktop.mkdir(exist_ok=False)
screens=[]
for p in sorted(W.glob('*.png')):
 target=desktop/p.name;shutil.copy2(p,target)
 assert sha(p)==sha(target)
 screens.append({'path':str(p),'sha256':sha(p),'bytes':p.stat().st_size,'desktop':str(target)})
assert len(screens)==5
old=B/'b19-native-input-limit-attempt1/receipt.json'
assert load(old)['status']=='FAIL'and all(load(old)['cleanup'].values())
final={'status':'DEV_B19_LOCAL_PG_NATIVE_WINDOWS_DELTA_GREEN','created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'original_build_manifest':{'path':str(original),'sha256':sha(original),'immutable_unchanged':True},'runtime':m['runtime'],'source':m['source'],'base_b18':m['base_b18'],'artifacts':m['artifacts'],'pg':m['tests']['pg'],'native':{'sdk_platform':'windows-amd64','sdk_sha256':native['sdk_sha256'],'browser_version':native['browser_version'],'checks_pass':43,'checks_fail':0,'page_errors':0,'cases':native['cases'],'receipt':str(W/'receipt.json'),'receipt_sha256':sha(W/'receipt.json'),'native_wire_summary':str(wirepath),'native_wire_sha256':sha(wirepath),'cache_cardinality_not_observed_in_cli':True,'cache_cardinality_covered_by_exact_public_http_matrix':True,'no_linux_browser_proof_claimed':True,'no_B18_native_retag':True,'screens':screens,'screenshot_visual_inspection':['cap8-seven-documents.png','cap3-post-refused.png','ordinary-query-warning.png']},'retained_failed_attempt':{'path':str(old),'sha256':sha(old),'classification':'observer expected raw303 from Playwright; actual own native journal303 followed by viewerGET200 and seven documents. Corrected observation only, same product/SDK/config fixture; no rebuild'},'cleanup':{'explicit_browser_END':True,'processes':cleanup,'receipt':str(B/'b19-native-cleanup-end.json'),'sha256':sha(B/'b19-native-cleanup-end.json')},'disk_cleanup':{'receipt':str(B/'20261115-platform-build-cache-cleanup-b19.json'),'sha256':sha(B/'20261115-platform-build-cache-cleanup-b19.json'),'removed_bytes':cache['removed_file_bytes'],'free_C_after':cache['free_C_after'],'only_build_cache':True},'source_draft':'https://github.com/ivanarama/onebase/pull/2627','scope':{'db_copies':0,'shared_changes_by_platform':False,'configuration_changes':False,'production_package':False,'xlsx_dimension':'separate OPEN unchanged','native_Linux_acceptance':'cc2 owns next root gates/deployment; not claimed here'}}
out=B/'manifest-1115b19-tabular-input-limit-DEV-final.json';assert not out.exists()
out.write_text(json.dumps(final,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
report=R/'20261115-platform-B19-native-input-limit-PASS.md'
report.write_text(f'''# B19 native Windows: вход result_proc — PASS, cleanup END

Exact runtime `{m['runtime']}`, source `{m['source']}`, Windows SDK `{native['sdk_sha256']}`, Chromium `{native['browser_version']}`. Linux SDK `{m['artifacts']['linux']['sha256']}` собран отдельно; этот native proof — Windows, не Linux.

**43 проверки PASS, JS0.** Новая собственная синтетика:8 rawrows запроса/7 документов, собственная authSQLite удалена после закрытия. Никаких sourceDBcopies/shared/config изменений.

| Путь | Факт |
|---|---|
| cap3, GET и штатный Generate POST | видимый HTTP422/report_max_rows; без viewer/экспортных ссылок/redirect к новому или старому snapshot |
| cap3, module throw guard | HTTP422 до модуля; при cap8 тот же guard даёт ожидаемый500/MODULE_MUST_NOT_RUN, подтверждая действенность контроля |
| cap8 (точная граница), limit0 | GET и Generate POST: все7 документов DOC-A…DOC-G |
| no stale viewer | при том же server/cap3 сначала допустимый Tiny-query2rawrows/1doc создаёт snapshot, затем полный запрос получает422 и не показывает прежний viewer; прежний легитимный snapshot остаётся доступен200 |
| обычный query report, cap3 | GET/POST200, видимое предупреждение,3rawrows/2distinctdocs — прежняя policy |

Публикацию snapshot в CLI наблюдали через native server303/запрос viewer/DOM; приватную cardinality cache не читали. Её отсутствие при отказе отдельно проверено exact HTTP-матрицей в PG118. У старого Chrome/Playwright response observer Generate POST был виден как terminal200 при server303→viewerGET200; первый строгий observerFAIL сохранён отдельно. Второй observer сверяет server metadata status и фактический viewer; продукт/SDK не менялись.

Receipt `{W/'receipt.json'}` SHA `{sha(W/'receipt.json')}`. Metadata-only wire summary `{wirepath}` SHA `{sha(wirepath)}`. Пять PNG: `{desktop}`. Три основных снимка просмотрены: семь документов; понятный422; обычный отчёт с видимым warning и3строками.

PG59SQLite+59PG=118leaf PASS/FAIL0/SKIP0, cleanup exact; receipt SHA `{sha(pg)}`. Original buildmanifest94054e97 неизменён. **Final supplement** `{out}` SHA `{sha(out)}`.

**Browser/app/port18568/tunnel END**: CIMGo/compiler/Chrome/driver0, ownDB удалена. Затем разрешённый pinned `go clean -cache` при Go0 до/после освободил4185073390B (~3.90GiB), Cfree5753634816B (~5.36GiB); SDK/modcache/source/DB/квитанции не затронуты. XLSXdimension остаётся отдельнымOPEN. Rootcc2 выполняет Linux layout/result/deploy своей очередью; production не готовился.
''',encoding='utf-8')
print(json.dumps({'final':str(out),'sha256':sha(out),'receipt_sha256':sha(W/'receipt.json'),'cleanup_sha256':sha(B/'b19-native-cleanup-end.json'),'screens':len(screens),'screens_directory':str(desktop),'report':str(report)},ensure_ascii=False))
