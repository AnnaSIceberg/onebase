import pathlib,json,hashlib,subprocess,os,datetime,sys
B=pathlib.Path(r"C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111")
W=B/"tabular-link-color-runtime"
branch="wip/platform-b24-final-b25-handoff-20261115"
tag=sys.argv[1]
files=json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8"))
def git(*args,data=None,env=None):
 return subprocess.check_output(["git","-c","gc.auto=0","-c","http.sslBackend=openssl",*args],cwd=W,input=data,env=env,creationflags=subprocess.CREATE_NO_WINDOW)
assert not git("status","--porcelain").strip()
parent=git("rev-parse","refs/heads/"+branch).decode().strip()
remote=git("ls-remote","fork","refs/heads/"+branch).decode().split()[0]
assert remote==parent
idx=B/("support-"+tag+".index")
assert not idx.exists()
env=dict(os.environ,GIT_INDEX_FILE=str(idx));git("read-tree",parent,env=env)
inv=[]
for item in files:
 p=pathlib.Path(item["source"]);target=item["target"];raw=p.read_bytes()
 assert len(raw)<2*1024*1024
 oid=git("hash-object","-w","--stdin",data=raw).decode().strip()
 git("update-index","--add","--cacheinfo","100644,"+oid+",docs/history/platform-b25-20261115/"+target,env=env)
 inv.append({"path":target,"bytes":len(raw),"sha256":hashlib.sha256(raw).hexdigest()})
tree=git("write-tree",env=env).decode().strip()
msg="docs(wip): сохранить checkpoint platform "+tag+"\n\nGenerated-with: Codex\n"
commit=git("commit-tree",tree,"-p",parent,data=msg.encode()).decode().strip()
git("update-ref","refs/heads/"+branch,commit,parent)
git("push","fork",branch)
assert git("ls-remote","fork","refs/heads/"+branch).decode().split()[0]==commit
out=B/("support-"+tag+"-push.json")
assert not out.exists()
r={"branch":branch,"commit":commit,"parent":parent,"files":inv,"created_utc":datetime.datetime.now(datetime.timezone.utc).isoformat(),"scope":"metadata/source reports only; no SDK/DB/secrets/payloads"}
out.write_text(json.dumps(r,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
print(json.dumps({"commit":commit,"file_count":len(inv),"receipt":str(out),"sha256":hashlib.sha256(out.read_bytes()).hexdigest()}))
