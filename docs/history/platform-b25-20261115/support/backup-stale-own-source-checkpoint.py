from pathlib import Path
import subprocess,json,datetime
B=Path(r"C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111")
rows=json.loads((B/"reboot-precheck-worktrees-1730.json").read_text(encoding="utf-8"))
def git(w,*a):return subprocess.check_output(["git","-c","http.sslBackend=openssl",*a],cwd=w,creationflags=subprocess.CREATE_NO_WINDOW).decode("utf-8").rstrip("\r\n")
result=[]
for r in rows:
 if r["remote_exact"] and r["clean"]:continue
 w=Path(r["worktree"]);branch="wip/platform-reboot-20261115/"+w.name
 assert git(w,"rev-parse","HEAD")==r["head"]
 if not r["clean"]:
  git(w,"switch","-c",branch)
  paths=[x[3:] for x in git(w,"status","--porcelain").splitlines()]
  assert all(x.endswith(".go") and x.startswith("internal/") for x in paths)
  git(w,"add","--",*paths)
  git(w,"commit","-m","chore(wip): сохранить старый профилировочный исходник перед перезагрузкой","-m","Backup only; not shipping, not tested, no runtime execution.\n\nGenerated-with: Codex")
 else:
  git(w,"branch",branch,r["head"])
 head=git(w,"rev-parse",branch)
 git(w,"push","fork",branch)
 remote=git(w,"ls-remote","fork","refs/heads/"+branch).split()[0]
 assert remote==head and not git(w,"status","--porcelain")
 result.append({"worktree":str(w),"old_head":r["head"],"backup_head":head,"backup_branch":branch,"remote_exact":True,"clean":True,"scope":"archival source only; excluded B25"})
p=B/"reboot-stale-source-backup.json"
p.write_text(json.dumps({"created_utc":datetime.datetime.now(datetime.timezone.utc).isoformat(),"items":result},ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
print(json.dumps({"saved":len(result),"receipt":str(p)}))
