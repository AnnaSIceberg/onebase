import pathlib,json,hashlib,subprocess,datetime,collections
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=B/'report-tabular-result-runtime';S=B/'report-tabular-result-source'
HEAD='e6cb9a375b0bae15a430ae93b0913c02938bb45e';SOURCE='d232937812637564c730fbc17a9376d269dc5247';BASE='cd9910c81fd7812874b9ca2cafa1870857abee5e'
def sha(p):
 with pathlib.Path(p).open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
def git(w,*a):return subprocess.check_output(['git',*a],cwd=w,text=True,encoding='utf-8',creationflags=subprocess.CREATE_NO_WINDOW).strip()
assert git(W,'rev-parse','HEAD')==HEAD and not git(W,'status','--porcelain')
assert git(S,'rev-parse','HEAD')==SOURCE and not git(S,'status','--porcelain')
end=load(B/'b20-go-end-processes.json');assert end['go_process_count']==0 and end['runtime_head']==HEAD and end['worktree_clean']
artifacts={k:load(B/('b20-'+k+'-artifact.json'))for k in ['ui_pg','linux','windows']}
for a in artifacts.values():
 assert a['runtime']==HEAD and a['source']==SOURCE and sha(a['path'])==a['sha256']
 info=pathlib.Path(a['buildinfo']).read_text('utf-8');assert'go1.26.11'in info and'CGO_ENABLED=0'in info
 a['embedded_vcs_revision_present']='vcs.revision'in info
 a['revision_binding']='Exact clean source/runtime, argv/env command receipts and artifact SHA; embedded VCS not invented'
expected=load(B/'b20-ui-pg-expected.json');assert expected['expected_total']==len(set(expected['expected_subtests']))==132
pgpath=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/cc2-task-rest-blocker-20261113')/('pg-ui-'+artifacts['ui_pg']['sha256'][:8])/'receipt.json'
pg=load(pgpath)
assert pg['binary_sha256']==artifacts['ui_pg']['sha256']and pg['runtime_head']==HEAD and pg['source_head']==SOURCE
assert pg['leaf_counts']=={'sqlite':66,'postgres':66}and pg['expected_subtests_passed']and pg['status']=='PASS'
assert not pg['failed_tests']and not pg['skip']and pg['ephemeral_cleanup_exact']
assert pg['schema_before_sha256']==pg['schema_after_sha256']=='95a93e4bc016ac4a5762f6f53e38e8988d5ae40cbf0155370204c0ad932e7dc4'
summary=load(B/'b20-local-targeted-summary.json');counts=collections.Counter()
for run in summary['runs']:counts.update(run['leaf_counts'])
assert counts.get('fail',0)==0 and counts['skip']==10
ui=next(x for x in summary['runs']if x['log'].endswith('b20-ui-targeted.jsonl'));assert ui['leaf_counts']=={'pass':66,'skip':10}
red=load(B/'b20-red-old-xlsx/receipt.json');assert red['exit']==1 and len(red['failed_leaves'])==7
raw=(B/'b20-source-dimension-v1.jsonl').read_bytes();text=raw.decode('utf-16'if raw.startswith(b'\xff\xfe')else'utf-8-sig')
events=[json.loads(x)for x in text.splitlines()if x.startswith('{')]
named={(r['Package'],r['Test']):r['Action']for r in events if r.get('Test')and r['Action']in('pass','fail','skip')}
leaves={k:v for k,v in named.items()if not any(q[0]==k[0]and q[1].startswith(k[1]+'/')for q in named)}
assert dict(collections.Counter(leaves.values()))=={'pass':17,'skip':1}
changed=git(W,'diff','--name-only',BASE,HEAD).splitlines();assert set(changed)=={'CHANGELOG.md','internal/sheet/xlsx.go','internal/sheet/xlsx_dimension_test.go','internal/ui/tabular_xlsx_dimension_test.go'}
files=[]
for p in changed:
 blob=subprocess.check_output(['git','show',HEAD+':'+p],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW)
 files.append({'path':p,'git_blob_content_sha256':hashlib.sha256(blob).hexdigest(),'bytes':len(blob)})
frozen=[('manifest-1115b19-tabular-input-limit-DEV.json','94054e97d5cf6c879e6c7b9fc302a43f88a623096450a72962a5468e59837662'),('manifest-1115b19-tabular-input-limit-DEV-final.json','9eb4279b2f015c82b8b6063d72d420ac1388e26571d5f0a0ad6d6a318498e85e'),('onebase-1115b19-tabular-input-limit-linux-amd64','3f55a5a0d7bfb83a1ce8d69609f1b27abd1ce1f77c100708650d1bd847490b20'),('onebase-1115b19-tabular-input-limit-windows-amd64.exe','da985170e49b24242cbb65cb6fb091a35e9c9992a8a13397d7437f02b763d60e'),('onebase-1115b18-kpi-size-scale-linux-amd64','24897fe0c06347704440982007e8ccac9a8550165fd2fb2b6b6cc2cd153e10d4')]
for p,h in frozen:assert sha(B/p)==h
manifest={'status':'B20_COMBINED_DEV_LOCAL_PG_GREEN_NATIVE_READER_PENDING','created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'runtime':HEAD,'source':SOURCE,'base_b19':BASE,'runtime_branch':'fix/xlsx-used-range-runtime-20261115','source_branch':'fix/xlsx-used-range-20261115','source_draft':'https://github.com/ivanarama/onebase/pull/2627','artifacts':artifacts,'combined_fix':{'result_proc_input_limit':'B19 retained: truncated positive-limit input HTTP422 before module/new snapshot; ordinary warning/zero unchanged','xlsx_dimension':'SetSheetDimension before WriteToBuffer: A1:last from 0-based inclusive ContentBounds + spans/style-only cells; empty/single A1; row-height/column-width maps alone excluded','outline':'same dimension preserved through existing ZIP outline pass','protected':'same typed values/styles/numeric formats/print settings/snapshot/rights and budgets','compatibility_scope':'ZIP regression PASS; default installed openpyxl3.1.5 read_only gate still pending on new SDK exports; other readers not claimed'},'tests':{'source':{'pass':17,'local_pg_skip':1,'log':str(B/'b20-source-dimension-v1.jsonl'),'sha256':sha(B/'b20-source-dimension-v1.jsonl')},'red_old_writer':{'old_source':red['old_writer_source'],'expected_failures':7,'receipt':str(B/'b20-red-old-xlsx/receipt.json'),'sha256':sha(B/'b20-red-old-xlsx/receipt.json'),'not_frozen_sdk_execution':True},'runtime_local':{'leaf_pass':counts['pass'],'fail':0,'local_pg_root_skip':10,'summary':str(B/'b20-local-targeted-summary.json'),'sha256':sha(B/'b20-local-targeted-summary.json')},'pg':{'sqlite':66,'postgres':66,'total':132,'fail':0,'skip':0,'expectednames_all_matched':True,'receipt':str(pgpath),'sha256':sha(pgpath),'cleanup_exact':True},'build_all':{'exit':0,'log':str(B/'b20-build-all.log'),'sha256':sha(B/'b20-build-all.log')},'i18n_en':{'exit':0,'log':str(B/'b20-i18n-en.log'),'sha256':sha(B/'b20-i18n-en.log')},'unrelated_api_repeated':False},'expected':{'path':str(B/'b20-ui-pg-expected.json'),'sha256':sha(B/'b20-ui-pg-expected.json'),'regex':expected['regex'],'names':expected['expected_subtests']},'changed_files':files,'frozen_verified':[{'path':str(B/p),'sha256':h}for p,h in frozen],'go_end':{'explicit':True,'processes':0,'receipt':str(B/'b20-go-end-processes.json'),'sha256':sha(B/'b20-go-end-processes.json')},'native_reader_gate':{'status':'PENDING_SEPARATE_BROWSER_SLOT','interpreter':'C:/Python314/python.exe','expected_openpyxl':'3.1.5','must_use_default_read_only_without_reset_or_forced_bounds':True,'old_B18_XLSX_rewritten':False},'scope':{'db_copies':0,'shared_changed':False,'config_changed':False,'release_execution':False,'live8444':'B18 remains; B19 intermediate NOT DEPLOYED; root controls next acceptance','P4_P7':'platform2 owns separate source; not included in B20','freeze':'by17:30, reboot checkpoint17:45; no execution without evening instruction'}}
out=B/'manifest-1115b20-xlsx-dimension-DEV.json';assert not out.exists()
out.write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
report=R/'20261115-platform-B20-XLSXdimension-combined-DEV.md'
report.write_text(f'''# B20 combined: result_proc input-limit + XLSX dimension

Runtime `{HEAD}`, source `{SOURCE}`, exactB19 parent `{BASE}`. B19 artifacts94054/9eb не менялись/не выкладывались. Source/runtime ветки pushed, Draft2627.

Writer явно задаёт A1:last из ContentBounds inclusive0-based с правильным col+1/row+1 до WriteToBuffer. Включены merge spans и пустые оформленные крайние ячейки, не только141 непустое значение. Пустой/одноклеточный A1; Width/Heightmaps сами по себе range не расширяют. API ошибки возвращаются. Outline/nonoutline проходят один и тот же dimension. B19 input-limit отказ доresult_proc/new snapshot сохранён; ordinary warning,0default, byte/cell/payload/shape/rights budgets не менялись.

Source17PASS/1localPGSKIP; exact oldwriter черезGooverlay7expectedRED. Build./.../i18nen и fullsheet/printform+relatedinterpreter+UI `{counts['pass']}` localleafPASS,10localPGrootsSKIP отдельно. Независимый prebuilt66SQLite+66PG=132exactleafPASS/FAIL0/SKIP0, cleanup exact, receipt `{pgpath}` SHA `{sha(pgpath)}`. API6 не повторялся. Новые публичныеZIP/unit10 и nativeHTTP7/dialect: empty/A1, far sparse/offset, merge span, style-only tail, collapsedgroups, AutoScale иKPI53893; числовые/print/source-snapshot controls сохранены.

| Artifact | SHA256 | Path |
|---|---|---|
| Linux UI prebuilt | `{artifacts['ui_pg']['sha256']}` | `{artifacts['ui_pg']['path']}` |
| Linux SDK | `{artifacts['linux']['sha256']}` | `{artifacts['linux']['path']}` |
| Windows SDK | `{artifacts['windows']['sha256']}` | `{artifacts['windows']['path']}` |

Manifest `{out}` SHA `{sha(out)}`. Полныеnames/regex `{B/'b20-ui-pg-expected.json'}` SHA `{sha(B/'b20-ui-pg-expected.json')}`. GO END процессов0/cleanWT; externalcommand/SHA binding, embeddedVCS не выдуман.

Native/default installed openpyxl3.1.5 read_only безreset/forcedbounds на НОВЫХ SDK exports ещё PENDING, не подменяется normal/pandas/старымиB18xlsx. Старыеfacts49066a8a и файлы не изменены; остальныеreaders UNKNOWN. Следующий короткий Chrome-слот — отдельно cc2. Shared/:8444/config/production/DBcopies0. К17:30 финальныйfreeze,17:45 ownprocess0/cleanpushed checkpoint; production execution только по вечернему слову.
''',encoding='utf-8')
print(json.dumps({'manifest':str(out),'sha256':sha(out),'local_leaf_pass':counts['pass'],'report':str(report)},ensure_ascii=False))
