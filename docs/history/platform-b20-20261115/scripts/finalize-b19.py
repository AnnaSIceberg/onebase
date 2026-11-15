import pathlib, json, hashlib, subprocess, datetime, collections
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=B/'report-tabular-result-runtime'
HEAD='cd9910c81fd7812874b9ca2cafa1870857abee5e'
SOURCE='815e1505f504659fedea159fd186eeff3ab36d33'
BASE='64886e644da31991abd93d48390aaff7b883e9bc'
def sha(p):
 with pathlib.Path(p).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
def git(*a):return subprocess.check_output(['git',*a],cwd=W,text=True,encoding='utf-8',creationflags=subprocess.CREATE_NO_WINDOW).strip()
assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
end=load(B/'b19-go-end-processes.json')
assert end['go_process_count']==0 and end['runtime_head']==HEAD and end['worktree_clean']
artifacts={k:load(B/('b19-'+k+'-artifact.json'))for k in ['ui_pg','linux','windows']}
for a in artifacts.values():
 assert a['runtime']==HEAD and a['source']==SOURCE and sha(a['path'])==a['sha256']
 info=pathlib.Path(a['buildinfo']).read_text('utf-8')
 assert 'go1.26.11' in info and 'CGO_ENABLED=0' in info
 a['embedded_vcs_revision_present']='vcs.revision' in info
 a['revision_binding']='Exact clean worktree before/after, command receipts and SHA; no invented embedded VCS revision'
expected=load(B/'b19-ui-pg-expected.json')
assert expected['expected_total']==len(set(expected['expected_subtests']))==118
pgdir=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/cc2-task-rest-blocker-20261113')
pgpath=pgdir/('pg-ui-'+artifacts['ui_pg']['sha256'][:8])/'receipt.json'
pg=load(pgpath)
assert pg['binary_sha256']==artifacts['ui_pg']['sha256'] and pg['runtime_head']==HEAD and pg['source_head']==SOURCE
assert pg['leaf_counts']=={'sqlite':59,'postgres':59} and pg['expected_subtests_passed'] and pg['status']=='PASS'
assert not pg['failed_tests'] and not pg['skip'] and pg['ephemeral_cleanup_exact']
assert pg['schema_before_sha256']==pg['schema_after_sha256']=='95a93e4bc016ac4a5762f6f53e38e8988d5ae40cbf0155370204c0ad932e7dc4'
summary=load(B/'b19-local-targeted-summary.json')
counts=collections.Counter()
for run in summary['runs']:counts.update(run['leaf_counts'])
assert counts=={'pass':59,'skip':9},counts
red=load(B/'b19-red-frozen621/receipt.json')
assert red['expected_red'] and len(red['failed_leaves'])==6
changed=git('diff','--name-only',BASE,HEAD).splitlines()
assert set(changed)=={'CHANGELOG.md','internal/i18n/locales/en.json','internal/ui/handlers_reports.go','internal/ui/tabular_query_limit_test.go'}
files=[]
for p in changed:
 blob=subprocess.check_output(['git','show',HEAD+':'+p],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW)
 files.append({'path':p,'git_blob_content_sha256':hashlib.sha256(blob).hexdigest(),'bytes':len(blob)})
frozen=[('manifest-1115b18-kpi-size-scale-DEV-final.json','3b84fec1a3c39c076a3e89e36f32b0a6baaecb37a74aa44da2732f1cb6ef5a05'),('onebase-1115b18-kpi-size-scale-linux-amd64','24897fe0c06347704440982007e8ccac9a8550165fd2fb2b6b6cc2cd153e10d4'),('onebase-1115b18-kpi-size-scale-windows-amd64.exe','a03f0edec3e125be33179fadec260a3e7a990b6ca5216e0af9a8847786bcfa35')]
for p,h in frozen:assert sha(B/p)==h
manifest={'status':'DEV_BUILD_LOCAL_AND_PG_GREEN_NATIVE_PENDING','created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'runtime':HEAD,'source':SOURCE,'base_b18':BASE,'runtime_branch':'fix/tabular-query-input-runtime-20261115','source_draft':'https://github.com/ivanarama/onebase/pull/2627','artifacts':artifacts,'contract':{'query_result_proc_truncated':'HTTP422 before module, no new snapshot/viewer/export or redirect to an old viewer','full_or_exact_boundary':'native tabular result unchanged','ordinary_query':'existing warning and row budget policy unchanged','report_max_rows':'positive budgets retained; zero still disables this query guard','cell_and_payload_budgets':'unchanged B18','no_query_result_proc':'unchanged','xlsx_dimension':'separate OPEN; not changed or claimed fixed'},'tests':{'red':{'fixture_source':SOURCE,'exact_handler_source':red['overlaid_handler_source'],'expected_failures':6,'receipt':str(B/'b19-red-frozen621/receipt.json'),'sha256':sha(B/'b19-red-frozen621/receipt.json'),'not_frozen_sdk_execution':True},'source_green':{'sqlite_leaf_pass':16,'local_postgres_root_skip':1,'log':str(B/'b19-source-input-limit-v1.jsonl'),'sha256':sha(B/'b19-source-input-limit-v1.jsonl')},'runtime_local':{'leaf_pass':59,'fail':0,'local_pg_root_skip':9,'summary':str(B/'b19-local-targeted-summary.json'),'sha256':sha(B/'b19-local-targeted-summary.json')},'pg':{'sqlite':59,'postgres':59,'total':118,'fail':0,'skip':0,'expectednames_all_matched':True,'receipt':str(pgpath),'sha256':sha(pgpath),'cleanup_exact':True},'build_all':{'exit':0,'log':str(B/'b19-build-all.log'),'sha256':sha(B/'b19-build-all.log')},'i18n_en':{'exit':0,'log':str(B/'b19-i18n-en.log'),'sha256':sha(B/'b19-i18n-en.log')},'api6_repeated':False},'expected':{'path':str(B/'b19-ui-pg-expected.json'),'sha256':sha(B/'b19-ui-pg-expected.json'),'regex':expected['regex'],'names':expected['expected_subtests']},'changed_files':files,'go_end':{'explicit':True,'processes':0,'receipt':str(B/'b19-go-end-processes.json'),'sha256':sha(B/'b19-go-end-processes.json')},'native_browser':{'status':'NOT_STARTED_SEPARATE_SLOT_REQUIRED','B18_proof_retagged':False},'scope':{'db_copies':0,'shared_changes':False,'config_changes':False,'production_package':False,'dev8444':'B18 remains until cc2 acceptance'},'frozen_b18_verified':[{'path':str(B/p),'sha256':h}for p,h in frozen]}
out=B/'manifest-1115b19-tabular-input-limit-DEV.json'
assert not out.exists()
out.write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
report=R/'20261115-platform-B19-result-proc-input-limit-DEV.md'
report.write_text(f'''# B19: result_proc — явный отказ при усечённом входе

Source `{SOURCE}`, runtime `{HEAD}`, точная база B18 `{BASE}`. Отдельный DEV-кандидат, Draft2627 остаётся Draft.

При положительном report_max_rows усечённый вход запроса теперь получает HTTP422 с понятным сообщением ДО исполнения result_proc. Новый viewer/snapshot/экспорт не создаётся, перенаправления на старый viewer нет. Полный вход и точная граница проходят. Обычный query-report сохраняет прежнее предупреждение/лимит; zero/default0 не меняется. Ячеечные/payload бюджеты B18 и отдельный открытый XLSX dimension не затронуты.

Публичная GET/POST матрица использует 8 raw query rows/7 документов: disabled, exact/below boundary, hit, module-must-not-run, no-stale-viewer, ordinary warning/disabled. Это созданная регрессия; число rawrows исходного perenos repro ею не подменяется. На exact621 handler через Go overlay 6 отрицательных проверок ожидаемо RED; с815 source16SQLite PASS. Runtime59SQLite PASS, local9PGrootsSKIP отдельно. Независимый Linux prebuilt59SQLite+59PG=118 exactleaf PASS, FAIL0/SKIP0, cleanup exact. Build ./... и i18n en PASS; unrelated API6 не повторялся.

| Artifact | SHA256 | Path |
|---|---|---|
| Linux UI prebuilt | `{artifacts['ui_pg']['sha256']}` | `{artifacts['ui_pg']['path']}` |
| Linux SDK | `{artifacts['linux']['sha256']}` | `{artifacts['linux']['path']}` |
| Windows SDK | `{artifacts['windows']['sha256']}` | `{artifacts['windows']['path']}` |

PG receipt `{pgpath}`, SHA `{sha(pgpath)}`. Полные имена/regex `{B/'b19-ui-pg-expected.json'}`, SHA `{sha(B/'b19-ui-pg-expected.json')}`. Manifest `{out}`, SHA `{sha(out)}`. Чистый WT/argv/env/SHA связывают source/runtime с артефактами; embeddedVCS не выдуман.

GO END: Go/compile/link/asm/cgo0; Chrome не запускался, следующий короткий слот отдельно. FrozenB18 SDK/finalmanifest повторно проверены и неизменны; :8444/config/shared/production не менялись. Native B19 ещё PENDING, прежняя B18 native приёмка ему не приписывается.
''',encoding='utf-8')
print(json.dumps({'manifest':str(out),'sha256':sha(out),'report':str(report)},ensure_ascii=False))
