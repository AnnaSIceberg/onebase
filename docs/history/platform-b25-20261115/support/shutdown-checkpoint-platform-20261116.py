import collections,datetime,hashlib,json,pathlib,re,subprocess
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
GH='C:/Users/a.sevrjukova/gh-cli/bin/gh.exe'
def run(args,cwd=None):return subprocess.check_output(args,cwd=cwd,creationflags=subprocess.CREATE_NO_WINDOW)
def git(*a,cwd=None):return run(['git',*a],cwd or B/'report-b30-date-mask-runtime').decode('utf-8').strip()
paths=[B/'report-b30-date-mask-runtime',B/'report-b29-enum-runtime',B/'report-b28-runtime',B/'workbook-b27-runtime',B/'platform-b26-ci-20261116',pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116'),pathlib.Path('C:/Users/a.sevrjukova/platform-report-group-levels-20261116'),pathlib.Path('C:/Users/a.sevrjukova/platform-report-enum-20261116'),pathlib.Path('C:/Users/a.sevrjukova/platform-workbook-xlsx-20261116'),pathlib.Path('C:/Users/a.sevrjukova/platform-tabular-parent-ci-20261116'),pathlib.Path('C:/Users/a.sevrjukova/platform2-report-drilldown'),pathlib.Path('C:/Users/a.sevrjukova/platform2-report-context'),pathlib.Path('C:/Users/a.sevrjukova/platform2-list-xlsx-source'),pathlib.Path('C:/Users/a.sevrjukova/platform2-report-acl-source')]
rows=[]
for w in paths:
 if not w.exists():continue
 branch=git('branch','--show-current',cwd=w);head=git('rev-parse','HEAD',cwd=w);status=git('status','--porcelain',cwd=w)
 rows.append({'branch':branch,'head':head,'worktree':str(w),'status':status})
# Source Draft #2638 remains a contract-only branch. The unfinished code is WIP.
draft='feature/report-group-levels-20261116'
rows.append({'branch':draft,'head':git('rev-parse','refs/heads/'+draft),'worktree':'current level WT switched to separate WIP; no duplicate source implementation','status':''})
remote={line.split()[1].removeprefix('refs/heads/'):line.split()[0]for line in git('ls-remote','fork',*['refs/heads/'+row['branch']for row in rows]).splitlines()}
for row in rows:
 row['remote_head']=remote.get(row['branch']);row['remote_matches']=row['head']==row['remote_head']
 assert row['remote_matches'],row
 assert not row['status'],row
ci=json.loads(run([GH,'pr','view','2611','--repo','ivanarama/onebase','--json','headRefOid,statusCheckRollup,mergeStateStatus']))
ci_path=B/'shutdown-date-source-ci-20261116.json';assert not ci_path.exists();ci_path.write_text(json.dumps(ci,indent=2)+'\n',encoding='utf-8')
evidence=[]
for job in [133300453736,133300453880,133300453906,133301908915]:
 p=B/('shutdown-date-ci-job-'+str(job)+'-20261116.log');assert not p.exists()
 result=subprocess.run([GH,'api',f'repos/ivanarama/onebase/actions/jobs/{job}/logs'],stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW,timeout=60)
 if result.returncode==0:
  p.write_bytes(result.stdout)
  lines=result.stdout.decode('utf-8','replace').splitlines()
  hits=[line for line in lines if re.search(r'FAIL:|Error:|error:|##\[error\]|Timed out|AssertionError|Error Context|panic:',line)]
  excerpt=B/('shutdown-date-ci-job-'+str(job)+'-failures-20261116.txt');excerpt.write_text('\n'.join(hits[-30:])+'\n',encoding='utf-8')
  evidence.append({'job':job,'log':str(p),'sha256':hashlib.sha256(result.stdout).hexdigest(),'failure_excerpt':str(excerpt)})
 else:evidence.append({'job':job,'download_status':'UNAVAILABLE','error':result.stderr.decode('utf-8','replace')[:200]})
receipt={'created':datetime.datetime.now(datetime.timezone.utc).isoformat(),'branches':rows,'source_ci':str(ci_path),'ci_logs':evidence,'go_started':False,'browser_started':False,'shared_changes':False,'process_check':'targeted elevated CIM found no own matching processes; native45 cleanupalltrue; ports18586/18587 not listening','global_heavy_lock':'no lock held by this platform run; no heavy commands started after shutdown order'}
p=B/'shutdown-platform-refs-20261116.json';assert not p.exists();p.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'receipt':str(p),'branches_verified':len(rows),'all_clean':all(not x['status']for x in rows),'ci_log_status':[{k:v for k,v in x.items()if k in ['job','download_status','failure_excerpt']}for x in evidence]}),flush=True)
