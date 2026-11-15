"""Back up code and metadata only; never upload DB, credentials, raw logs or customer fixtures."""
import pathlib,subprocess,json,os,hashlib,datetime
B=pathlib.Path(__file__).parent;S=B/'report-tabular-result-source'
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
PARENT='d232937812637564c730fbc17a9376d269dc5247';BRANCH='wip/platform-b20-handoff-20261115'
def git(*a,data=None,env=None):
 return subprocess.check_output(['git','-c','gc.auto=0',*a],cwd=S,input=data,env=env,creationflags=subprocess.CREATE_NO_WINDOW)
assert git('rev-parse','HEAD').decode().strip()==PARENT and not git('status','--porcelain').strip()
index=B/'b20-support-wip.index';assert not index.exists()
env=dict(os.environ,GIT_INDEX_FILE=str(index))
git('read-tree',PARENT,env=env)
files={}
for name in ['prepare-b19.py','build-kpi-b19.py','finalize-b19.py','b19-native-input-limit.py','prepare-b19-native-attempt2.py','b19-native-input-limit-attempt2.py','finalize-b19-native.py','clean-build-cache-b19.ps1','cleanup-old-b10-b17.ps1','recover-b10-b17-cleanup-receipt.py','write-cleanup-report-b20.py','prepare-b20-runtime.py','build-kpi-b20.py','finalize-b20.py','b20-installed-xlsx-reader.py','b20-native-dimension.py','finalize-b20-native.py','push-b20-support-wip.py']:
 files['scripts/'+name]=B/name
for name in ['manifest-1115b19-tabular-input-limit-DEV.json','manifest-1115b19-tabular-input-limit-DEV-final.json','manifest-1115b20-xlsx-dimension-DEV.json','manifest-1115b20-xlsx-dimension-DEV-final.json','b20-go-end-processes.json','b20-native-cleanup-end.json','b20-ui-pg-expected.json','b20-pg-handoff.json','b20-local-targeted-summary.json','20261115-platform-build-cache-cleanup-b19.json','20261115-platform-b10-b17-cleanup-recovered.json','b20-linux-artifact.json','b20-windows-artifact.json','b20-ui_pg-artifact.json','p1-b20-draft-status.json']:
 files['metadata/'+name]=B/name
for name in ['receipt.json','installed-default-reader.json','source-fixture-manifest.json']:
 files['metadata/native-'+name]=B/'b20-native-dimension-attempt1'/name
for name in ['20261115-platform-B20-XLSXdimension-combined-DEV.md','20261115-platform-B20-native-XLSXdimension-PASS.md','20261115-platform-очистка-диска-B10-B17.md','20261115-platform2-П4-П7-coordination.md']:
 files['reports/'+name]=R/name
manifest=[]
for target,p in sorted(files.items()):
 assert p.is_file() and p.suffix in ['.py','.ps1','.json','.md']
 data=p.read_bytes();oid=git('hash-object','-w','--stdin',data=data).decode().strip()
 path='docs/history/platform-b20-20261115/'+target
 git('update-index','--add','--cacheinfo','100644,'+oid+','+path,env=env)
 manifest.append({'target':path,'source':str(p),'bytes':len(data),'sha256':hashlib.sha256(data).hexdigest()})
readme=b'B20 support backup: code/metadata/reports only. No DB, cookies, credentials, binaries, raw server logs, customer layouts, screenshots or XLSX payloads. Product source is the parent commit. This WIP branch is not a release or merge request.\n'
oid=git('hash-object','-w','--stdin',data=readme).decode().strip()
git('update-index','--add','--cacheinfo','100644,'+oid+',docs/history/platform-b20-20261115/README.txt',env=env)
tree=git('write-tree',env=env).decode().strip()
message='chore(wip): сохранить B20 handoff и воспроизводимые проверки\n\nGenerated-with: Codex\n'.encode()
commit=git('commit-tree',tree,'-p',PARENT,data=message).decode().strip()
git('update-ref','refs/heads/'+BRANCH,commit,'0'*40)
git('-c','http.sslBackend=openssl','push','fork',BRANCH)
assert git('rev-parse','HEAD').decode().strip()==PARENT and not git('status','--porcelain').strip()
receipt={'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'branch':BRANCH,'commit':commit,'parent_product_source':PARENT,'file_count':len(files)+1,'product_worktree_untouched_clean':True,'fork_pushed':True,'data_excluded':['DB','credentials','cookies','SDK binaries','raw server logs','customer layouts','screenshots','XLSX payloads'],'files':manifest}
out=B/'b20-support-wip-push.json';assert not out.exists();out.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'branch':BRANCH,'commit':commit,'files':len(files)+1,'receipt':str(out)},ensure_ascii=False))
