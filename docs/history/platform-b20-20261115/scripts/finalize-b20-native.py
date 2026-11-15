import pathlib,json,hashlib,datetime,shutil
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=B/'b20-native-dimension-attempt1'
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
def sha(p):
 with pathlib.Path(p).open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
original=B/'manifest-1115b20-xlsx-dimension-DEV.json';assert sha(original)=='f98dd30ec8e75e71e6120f5b6551307d18c3aef473101a85f491b5bb7003c5af'
m=load(original);n=load(W/'receipt.json');reader=load(W/'installed-default-reader.json');end=load(B/'b20-native-cleanup-end.json')
assert n['status']=='PASS'and n['page_errors']==0 and len(n['checks'])==34 and all(n['checks'].values())and all(n['cleanup'].values())
assert n['sdk_platform']=='windows-amd64'and n['sdk_sha256']==m['artifacts']['windows']['sha256']
assert reader['openpyxl']=='3.1.5'and reader['read_only_default']and not reader['reset_dimensions_called']and not reader['forced_iteration_bounds']
assert len(reader['files'])==8 and all(x['counts_equal']and x['filled_coordinates_and_typed_values_equal']and not x['empty_document_no_values_exception']for x in reader['files'])
assert end['go_process_count']==end['chrome_process_count']==end['driver_process_count']==end['own_playwright_driver_count']==0 and end['own_port_closed']
for a in m['artifacts'].values():assert sha(a['path'])==a['sha256']
for case in n['cases'].values():assert sha(case['file'])==case['sha256']
pg=pathlib.Path(m['tests']['pg']['receipt']);assert sha(pg)=='a7d1bf16b6ed62913e20b05bde838b3b85c680778fcd40a29cd65ae43a2f4812'
oldexports=[(B/'b18-native-kpi-attempt3/large53893.xlsx','42abc18833d24177d2dddaceeedc6504193c2384dfc7421e571ba5231158a492'),(B/'b18-native-kpi-attempt3/auto51.xlsx','4c8a8d5a6bfac929fe367905456c02ac3e2cc1ee4cf94d61957451234e8bedc2')]
for p,h in oldexports:assert sha(p)==h
fixtures=[{'path':str(p.relative_to(W/'config')),'bytes':p.stat().st_size,'sha256':sha(p)}for p in sorted((W/'config').rglob('*'))if p.is_file()]
fixture=W/'source-fixture-manifest.json';assert not fixture.exists();fixture.write_text(json.dumps({'files':fixtures,'provided_layout_hash':'512146a7d06cb1f18da1091e4019261656a7e474fc01a28670153223d0bed1a1','scope':'source-only copy plus additive own synthetic buttons; no source DB copied'},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
desktop=pathlib.Path('C:/Users/a.sevrjukova/Desktop/ТабДок-B20-XLSX-dimension-20261115');desktop.mkdir(exist_ok=False)
screens=[]
for p in sorted(W.glob('*.png')):
 target=desktop/p.name;shutil.copy2(p,target);assert sha(p)==sha(target)
 screens.append({'path':str(p),'sha256':sha(p),'bytes':p.stat().st_size,'desktop':str(target)})
assert len(screens)==4
final={'status':'B20_COMBINED_LOCAL_PG_WINDOWS_NATIVE_DEFAULT_READER_GREEN_NOT_DEPLOYED','created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'runtime':m['runtime'],'source':m['source'],'base_b19':m['base_b19'],'original_build_manifest':{'path':str(original),'sha256':sha(original),'unchanged':True},'artifacts':m['artifacts'],'tests':m['tests'],'native':{'platform':'windows-amd64','sdk_sha256':n['sdk_sha256'],'browser_version':n['browser_version'],'checks_pass':34,'checks_fail':0,'page_errors':0,'cases':n['cases'],'receipt':str(W/'receipt.json'),'receipt_sha256':sha(W/'receipt.json'),'source_fixture_manifest':str(fixture),'source_fixture_manifest_sha256':sha(fixture),'runner':str(B/'b20-native-dimension.py'),'runner_sha256':sha(B/'b20-native-dimension.py'),'screens':screens,'no_linux_browser_proof_claimed':True,'no_old_proof_retag':True},'installed_reader':{'receipt':str(W/'installed-default-reader.json'),'receipt_sha256':sha(W/'installed-default-reader.json'),'script':str(B/'b20-installed-xlsx-reader.py'),'script_sha256':sha(B/'b20-installed-xlsx-reader.py'),'python':reader['python'],'executable':reader['executable'],'version':reader['openpyxl'],'files':reader['files'],'default_without_reset_or_forced_bounds':True,'normal_vs_default_all_counts_and_filled_typed_coordinates_equal':True,'empty_exception_used':False,'compatibility_scope':'Only checked openpyxl3.1.5 read_only/normal; Excel/LibreOffice/other readers UNKNOWN. Generated KPI geometry, not business numeric parity.'},'cleanup':{'explicit_browser_END':True,'processes':end,'receipt':str(B/'b20-native-cleanup-end.json'),'sha256':sha(B/'b20-native-cleanup-end.json'),'own_auth_db_removed':True},'old_B18_exports_verified_unchanged':[{'path':str(p),'sha256':h}for p,h in oldexports],'scope':{'db_copies':0,'shared_changed_by_platform':False,'release_execution':False,'live8444':'B18 remains; B20 preparation only until evening instruction','B19_intermediate_not_deployed':True,'P4_P7':'platform2 source separately; not included','source_runtime_pushed':True,'ready_for_final_candidate_integration':'Preserve exact tested B20 base; only subsequently reviewed changes in a new immutable candidate'}}
out=B/'manifest-1115b20-xlsx-dimension-DEV-final.json';assert not out.exists()
out.write_text(json.dumps(final,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
report=R/'20261115-platform-B20-native-XLSXdimension-PASS.md'
lines=[]
for f in reader['files']:
 d=f['default_read_only'];lines.append(f"| {f['file']} | {d['rows']} | {d['columns']} | {d['positions']} | {d['nonempty_values']} | PASS |")
report.write_text(f'''# B20 combined: XLSX dimension + result_proc input — native/reader PASS

Runtime `{m['runtime']}`, source `{m['source']}`, exactB19 parent `{m['base_b19']}`. Windows SDK `{n['sdk_sha256']}`, Chromium `{n['browser_version']}`. Linux SDK `{m['artifacts']['linux']['sha256']}` отдельно; данный native proof — Windows, не Linux.

**34 native checks PASS/JS0**, fresh ownSQLite/auth удалена,8НОВЫХexports. Actual XLSX ZIP dimension: largeA1:BO830/55610cells/1655merges;autoA1:BO51/3417cells/97merges;sparse/style-tailA1:BO830;offsetspanA1:H5;collapsedA1:D3;single/emptyA1. Model/style/merge хвосты не потеряны. Source layout512146a7 и исходный модуль такие же, как вB18; добавлены только собственные syntheticbuttons. Пустой/обычный/sparse/span/style-only/collapsed/auto иKPI проверены, не подменены одним dense тестом.

Installed Python3.14.3/openpyxl3.1.5: **default read_only=True, безreset/forcedbounds**, противnormal на этих новыхфайлах. ВоВСЕХ8 counts, заполненные координаты и типизированные значения равны; empty0rows/0positions/0values вОБОИХ, заранее разрешённое исключение не понадобилось.

| Новый файл | Строк | Колонок | Позиций | Непустых значений | Default vs normal |
|---|---:|---:|---:|---:|---|
{chr(10).join(lines)}

У исходной KPI-геометрии141значение — строки; это не доказательство числовой бизнес-полноты KPI/1С. Отдельные plain/sparse/styled controls имеют настоящий int42 и сохраняют тип. Проверена конкретная openpyxl3.1.5 совместимость; Excel/LibreOffice/прочие readers UNKNOWN, pandas/reset не использовались как acceptance.

Go/source:17PASS/1localPGSKIP, oldwriter7expectedRED; runtime212localleafPASS/10localPGrootsSKIP; build./.../i18nen PASS. Independent prebuilt66SQLite+66PG=132exactleafPASS/FAIL0/SKIP0/cleanup exact, SHAreceipt `{sha(pg)}`. Наследуемый B19 positive-limit422/ordinary-warning контроль повторён вPG132; unrelatedB18native/API6 не повторялись.

Native receipt `{W/'receipt.json'}` SHA `{sha(W/'receipt.json')}`. Installed reader `{W/'installed-default-reader.json'}` SHA `{sha(W/'installed-default-reader.json')}`. Fixture `{fixture}` SHA `{sha(fixture)}`.4PNG `{desktop}`, fullSHAs вsupplement. OriginalB20 buildmanifestf98dd30e и B18/B19 artifacts неизменны, старыеxlsx42abc/4c8a повторноSHAverified.

**Final supplement** `{out}` SHA `{sha(out)}`. **EXPLICIT BROWSER/APP/PORT/TUNNEL END**: CIMGo/compiler/vet/Chrome/driver/nodeplaywright0,18570closed, ownAuthDBremoved. :8444/shared/config/production не менялись. Её новое слово — подготовитьобщийвыпуск3базк17:30, НЕИСПОЛНЯТЬ до вечернего слова; B20 — tested base дляпоследующих reviewedchanges.17:45freshreboot checkpoint/cleanpushedWIP остаётся обязательным.
''',encoding='utf-8')
print(json.dumps({'final':str(out),'sha256':sha(out),'native_receipt_sha256':sha(W/'receipt.json'),'reader_sha256':sha(W/'installed-default-reader.json'),'cleanup_sha256':sha(B/'b20-native-cleanup-end.json'),'screens':len(screens),'report':str(report)},ensure_ascii=False))
