import pathlib,subprocess,os,json
B=pathlib.Path(__file__).parent;S=B/'report-tabular-result-source'
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
branch='wip/platform-b20-handoff-20261115'
def git(*a,data=None,env=None):return subprocess.check_output(['git','-c','gc.auto=0',*a],cwd=S,input=data,env=env,creationflags=subprocess.CREATE_NO_WINDOW)
parent=git('rev-parse','refs/heads/'+branch).decode().strip();assert parent=='a7960bb06a48d732a9d4584f5a19bbe48fab76df'
index=B/'acl-queue-wip.index';assert not index.exists();env=dict(os.environ,GIT_INDEX_FILE=str(index))
git('read-tree',parent,env=env)
report=R/'20261115-platform-ACL-result-proc-403-500-после-выпуска.md'
for p,target in [(report,'reports/'+report.name),(pathlib.Path(__file__),'scripts/'+pathlib.Path(__file__).name)]:
 oid=git('hash-object','-w','--stdin',data=p.read_bytes()).decode().strip()
 git('update-index','--add','--cacheinfo','100644,'+oid+',docs/history/platform-b20-20261115/'+target,env=env)
tree=git('write-tree',env=env).decode().strip()
commit=git('commit-tree',tree,'-p',parent,data='docs(wip): сохранить послерелизный дефект ACL отчёта\n\nGenerated-with: Codex\n'.encode()).decode().strip()
git('update-ref','refs/heads/'+branch,commit,parent);git('-c','http.sslBackend=openssl','push','fork',branch)
receipt={'branch':branch,'commit':commit,'parent':parent,'scope':'post-release queue report only; no product code/runtime/SDK change','product_worktree_clean':not git('status','--porcelain').strip()}
out=B/'acl-report-queue-wip-push.json';assert not out.exists();out.write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8');print(json.dumps(receipt))
