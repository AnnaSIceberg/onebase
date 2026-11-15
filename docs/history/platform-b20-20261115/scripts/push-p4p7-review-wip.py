import pathlib, subprocess, os, json, hashlib
B=pathlib.Path(__file__).parent; S=B/'report-tabular-result-source'
R=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
branch='wip/platform-b20-handoff-20261115'
def git(*a,data=None,env=None):return subprocess.check_output(['git','-c','gc.auto=0',*a],cwd=S,input=data,env=env,creationflags=subprocess.CREATE_NO_WINDOW)
parent=git('rev-parse','refs/heads/'+branch).decode().strip();assert parent=='d7457450f16129a920cd02b4b9b7ded440877648'
files=[R/n for n in ['20261115-platform-П7-source-review-2766-REQUEST-CHANGES.md','20261115-platform-П7-source-review-2766-pins.json','20261115-platform-П7-delta-review-02c62ecd-CODE-OK.md','20261115-platform-П7-delta-review-02c62ecd-pins.json','20261115-platform-П7-02c-source-log-review.json','20261115-platform-П4-source-review-cf5-CODE-OK.md','20261115-platform-П4-source-review-cf5-pins.json','20261115-platform-обычный-report-XLSX-dimension-после-выпуска.md','20261115-platform-KPI-T3-переход-формы-RO.md']]
assert all(p.is_file() for p in files)
index=B/'p4p7-review-wip.index';assert not index.exists();env=dict(os.environ,GIT_INDEX_FILE=str(index))
git('read-tree',parent,env=env)
inventory=[]
for p in [*files,pathlib.Path(__file__)]:
 raw=p.read_bytes();oid=git('hash-object','-w','--stdin',data=raw).decode().strip();target='docs/history/platform-b20-20261115/'+('scripts/' if p.suffix=='.py' else 'reports/')+p.name
 git('update-index','--add','--cacheinfo','100644,'+oid+','+target,env=env);inventory.append({'path':p.name,'sha256':hashlib.sha256(raw).hexdigest()})
tree=git('write-tree',env=env).decode().strip();commit=git('commit-tree',tree,'-p',parent,data='docs(wip): сохранить ревью П4 и П7 и очередь после выпуска\n\nGenerated-with: Codex\n'.encode()).decode().strip()
git('update-ref','refs/heads/'+branch,commit,parent);git('-c','http.sslBackend=openssl','push','fork',branch)
out=B/'p4p7-review-wip-push.json';assert not out.exists();receipt={'branch':branch,'parent':parent,'commit':commit,'scope':'review reports and post-release queue only; no product edits, binaries, DB, credentials or private captures','files':inventory,'product_worktree_clean':not git('status','--porcelain').strip()};out.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');print(json.dumps({'commit':commit,'files':len(inventory),'receipt_sha256':hashlib.sha256(out.read_bytes()).hexdigest(),'product_worktree_clean':receipt['product_worktree_clean']}))
