from pathlib import Path
import subprocess,json,datetime,hashlib
B=Path(r"C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111")
def git(w,*a):return subprocess.check_output(["git","-c","http.sslBackend=openssl",*a],cwd=w,creationflags=subprocess.CREATE_NO_WINDOW).decode("utf-8").rstrip("\r\n")
rows=[]
for repo in [Path(r"C:/Users/a.sevrjukova/onebase"),B/"tabular-link-color-runtime"]:
 remote={line.split()[1]:line.split()[0] for line in git(repo,"ls-remote","fork","refs/heads/*").splitlines()}
 for block in git(repo,"worktree","list","--porcelain").split("\n\n"):
  d=dict(line.split(" ",1) for line in block.splitlines() if " " in line)
  path=d.get("worktree","")
  if not path.startswith(B.as_posix()+"/") or not Path(path).exists():continue
  p=Path(path);branch=d.get("branch");st=git(p,"status","--porcelain")
  exact=[ref for ref,h in remote.items() if h==d["HEAD"]]
  rows.append({"worktree":path,"head":d["HEAD"],"branch":branch,"clean":not st,"current_branch_remote_exact":bool(branch and remote.get(branch)==d["HEAD"]),"exact_remote_refs":exact,"remote_backup_exact":bool(exact)})
assert all(x["clean"] and x["remote_backup_exact"] for x in rows)
expected={"tabular-link-color-runtime":"ab0511ff927b29475af07b80501a71a4519e1bc7","tabular-link-color-source":"a2ba78f8174c0bc26e5724b293408de25047a323","tabular-source-ci-close":"0594980b4de75fac38c44b7023624c79688d566d"}
for n,h in expected.items():
 r=next(x for x in rows if Path(x["worktree"]).name==n);assert r["head"]==h and r["current_branch_remote_exact"]
artifacts=[]
for n,h in [("onebase-1115b25-tabular-link-color-linux-amd64","992dab86d88332b0bb7f3f985071a84072a1e8f1efa50d14bec48f1964d3f8ec"),("onebase-1115b25-tabular-link-color-windows-amd64.exe","b4884534c4357d524acb706e64de8dd1553d140a2935dc0b0690a8d0020ec724"),("manifest-1115b25-tabular-link-color-DEV-final.json","fa4a4577d92f224f17093917e5ed771ea1b2ec48440e30c71951377b10c15c34")]:
 p=B/n;actual=hashlib.sha256(p.read_bytes()).hexdigest();assert actual==h;artifacts.append({"file":n,"bytes":p.stat().st_size,"sha256":actual})
r={"created_utc":datetime.datetime.now(datetime.timezone.utc).isoformat(),"owned_worktrees":rows,"count":len(rows),"all_clean":True,"all_heads_exact_on_fork":True,"active_three_current_branches_exact":True,"artifacts_unchanged":artifacts,"support_remote":git(B/"tabular-link-color-runtime","ls-remote","fork","refs/heads/wip/platform-b24-final-b25-handoff-20261115").split()[0]}
p=B/"reboot-fresh-worktrees.json";p.write_text(json.dumps(r,ensure_ascii=False,indent=2)+"\n",encoding="utf-8");print(json.dumps({"worktrees":len(rows),"all_clean":True,"all_remote_exact":True,"active_three_exact":True,"sha256":hashlib.sha256(p.read_bytes()).hexdigest()}))
