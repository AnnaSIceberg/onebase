import pathlib,json,hashlib,shutil,subprocess,datetime
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');R=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis');W=B/'reports-p4-p7-b21-runtime';N=B/'b21-native-context-drilldown-attempt3';H='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
def sha(p):
 with pathlib.Path(p).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
original=B/'manifest-1115b21-report-context-drilldown-DEV.json';assert sha(original)=='2de91957641e98920f0b19a1da7007b3521e3dc97858e6634c2988e326b2a5bb';m=load(original)
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()==H
assert not subprocess.check_output(['git','status','--porcelain'],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).strip()
pg=[]
for kind,prefix,want,expectedhash in [('api_pg','api-8f3025dc',6,'f6025235812e954cb1ffae93f648f0e332e6ed062f8895d48f5144761782c0ba'),('ui_pg','ui-e07da9bd',83,'6a553cc5990e6a2061120c3626ca22873e1c6c357ad483068b11529343c5fe14')]:
 p=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/cc2-task-rest-blocker-20261113')/('pg-'+prefix)/'receipt.json';r=load(p);assert sha(p)==expectedhash
 assert r['status']=='PASS' and r['runtime_head']==r['source_head']==H and r['binary_sha256']==m['artifacts'][kind]['sha256'];assert r['leaf_counts']=={'sqlite':want,'postgres':want} and r['expected_subtests_passed'] and not r['skip'] and not r['failed_tests'] and r['ephemeral_cleanup_exact'];assert r['schema_before_sha256']==r['schema_after_sha256']=='95a93e4bc016ac4a5762f6f53e38e8988d5ae40cbf0155370204c0ad932e7dc4'
 pg.append({'kind':kind,'receipt':str(p),'sha256':sha(p),'sqlite':want,'postgres':want,'fail':0,'skip':0,'expectednames_exact':True,'namespace_relations_routines_cleanup_exact':True})
np=N/'receipt.json';n=load(np);assert sha(np)=='2646775e3a73ae14c2148c2685562ac5384723d5d06f2e71c114d9f0efbc4daf';assert n['status']=='PASS' and n['sdk_sha256']==m['artifacts']['windows']['sha256'] and n['runtime']==H and len(n['checks'])==19 and all(n['checks'].values()) and all(n['cleanup'].values()) and n['page_errors']==0
end=B/'b21-native-cleanup-end.json';assert load(end)['go_compiler_driver_chrome109_count']==0
D=pathlib.Path(r'C:/Users/a.sevrjukova/Desktop/ТабДок-B21-П4-П7-native-20261115-platform');D.mkdir(exist_ok=False);screens=[]
for p in sorted(N.glob('*.png')):
 q=D/p.name;shutil.copy2(p,q);screens.append({'path':str(q),'sha256':sha(q),'bytes':q.stat().st_size})
assert len(screens)==4
frozen_fail=[]
for rel,kind in [('b21-native-context-drilldown-attempt1/receipt-recovered.json','setup fixture before server/browser; reference SQL physical _id columns; incomplete fixture connection cleanup recovered'),('b21-native-context-drilldown-attempt2/receipt.json','observer expected intermediate303 in Chromium event; functional15 controls passed; actual retained ownserver log2x303->200'),('b21-native-context-drilldown-attempt2/redirect-server-observation.json','RO server metadata status confirmation; no values/URI')]:
 p=B/rel;frozen_fail.append({'path':str(p),'sha256':sha(p),'classification':kind})
m.update({'status':'B21_OWNER_DEV_LOCAL_PG_NATIVE_GREEN_ROOT_ACCEPTANCE_PENDING','original_build_manifest':{'path':str(original),'sha256':sha(original),'unchanged':True},'independent_pg':{'unique_leaf_pass':178,'sqlite':89,'postgres':89,'fail':0,'skip':0,'receipts':pg},'native':{'status':'PASS','receipt':str(np),'receipt_sha256':sha(np),'checks_pass':19,'page_errors':0,'platform':'Windows-amd64 immutable e9bbbce6 SDK + Chromium109.0.5414.46','sdk_sha256':m['artifacts']['windows']['sha256'],'linux_native_proof':False,'own_fresh_synthetic':True,'DBcopies':0,'actor':'own synthetic admin for native positive; nonadmin/RBAC/RLS/revocation and binary rights verified in runtime HTTP PG matrices','checks':n['checks'],'fixture_files':n['source_fixture_files'],'snapshot_xlsx':n['xlsx'],'redirect_observer':'own server HTTP access status303 authoritative; browser follows terminal200; retained prior observer FAIL','screenshots':screens,'visual_inspection':['03-flat12-actions.png','02-queryless-context.png']},'browser_end':{'explicit':True,'path':str(end),'sha256':sha(end),'cleanup_alltrue':True,'go_chrome109_driver_count':0},'retained_native_attempts':frozen_fail,'known_boundaries':['Native positive uses own synthetic admin; no claim of real OTK/config business data acceptance','P4 raw register row compound-key drilldown unsupported and documented','Ordinary YAML report XLSXdimension A1 and nested result_proc ACL500 classification remain OPEN after-release owned by platform2; not fixed by B21','B20 full830/default-reader proof retained with byte-identical sheet/printform; B21 native adds small queryless snapshot default-reader, does not retag B20native proof','No shared/dev/production rollout performed; release PREPARE ONLY until evening user instruction; root acceptance still separate']})
out=B/'manifest-1115b21-report-context-drilldown-DEV-final.json';assert not out.exists();out.write_text(json.dumps(m,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
f=R/'20261115-platform-B21-П4-П7-PG-native-PASS.md';assert not f.exists();f.write_text(f'''# B21 П4/П7 — owner PG178 + WINDOWS native19 PASS

Exact combined runtime/source tree {H}, parent B20e6cb9a375b0bae15a430ae93b0913c02938bb45e. Sourceinputs П4cf5cf100 / П702c62ecd / B20 source d2329378. Source Draft2630/2629 остаются Draft. Runtime fork branch clean/pushed; build./.../i18nen PASS, local383leafPASS/14PGrootSKIP отдельно.

Независимый root exactprebuilt: API12(6SQLite+6PG), UI166(83+83), всего178uniqueleafPASS/FAIL0/SKIP0, expectednames matched; namespace/relations/routines before=after95a93e4b...7dc4 иcleanup exact. Receipt SHAs {pg[0]['sha256']} / {pg[1]['sha256']} независимо проверены. Исторические sourcePG18/14 не ретегированы.

Native exact Windows SDK e9bbbce60ce5b67e1867c0087eda77897b8e6ec540638bd8164814127bf6eb06 + Chrome109.0.5414.46:19checksPASS/JS0. FreshownSQLite/noDBcopy: spoofed Actor/Unit заменены серверным context, реальные строки только своего подразделения; native queryless result_proc получил UUID ссылки и actor; viewer/XLSX default openpyxl3.1.5 безreset дают точные2строки. П4:12detail_fields + отдельная actions(13cols), без пустой служебной колонки; cell+row routes303->nativeentity200; реальный UI anchor click открывает штатную форму. П4 nonadmin/RLS/revocation/blob/attachment guards проверены runtime HTTP SQLite+PG, native positive здесь synthetic admin — не прикладной ОТК acceptance.

Первый native setup FAIL доChrome (мойhelper physical reference columns) сохранён, SQLitecleanup восстановлен. Затем observer-only FAIL ожидания промежуточного303 в событии Chrome:15functionalchecksPASS, реальный server log2x303->200. Corrected observer+actualclick PASS без SDK/product изменений; оба FAIL immutable retained. Не объявляется fullCI/Ready/production readiness.

SDK Linux {m['artifacts']['linux']['sha256']} ({m['artifacts']['linux']['path']}); Windows {m['artifacts']['windows']['sha256']} ({m['artifacts']['windows']['path']}). Linux browser proof этим прогоном НЕ заявляется.

Final supplement {out} SHA {sha(out)}; original build manifest2de91957 неизменён. Native receipt2646775e, explicit cleanup {sha(end)}:browser/context/driver/app/port18572 closed, ownauthDB removed, tunnelnotused; freshCIMGo/Chrome109/driver0. Queue released cc2.

Desktop 4PNG: {D}. Flat12/queryless screenshots визуально прочитаны. XLSX/receipt hashes/полныеnames/regex/command bindings вmanifest. Обычный reportXLSXdimensionA1 и ACL403/500 остаются post-release уplatform2. Shared :8444 B18 не менялся; только подготовка вечернего состава, запуск по её слову/root очереди.

Generated-with: Codex
''',encoding='utf-8')
print(json.dumps({'final_manifest':str(out),'sha256':sha(out),'report_sha256':sha(f),'native_checks':19,'pg_leaf':178,'screenshots':len(screens),'cleanup_sha256':sha(end)}))
