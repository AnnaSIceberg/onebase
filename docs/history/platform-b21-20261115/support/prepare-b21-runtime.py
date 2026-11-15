import pathlib,subprocess,json
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');base=B/'report-tabular-result-runtime';new=B/'reports-p4-p7-b21-runtime';branch='feature/reports-p4-p7-b21-runtime-20261115'
def git(cwd,*args):return subprocess.run(['git','-c','gc.auto=0',*args],cwd=cwd,capture_output=True,creationflags=subprocess.CREATE_NO_WINDOW)
assert not git(base,'status','--porcelain').stdout.strip()
assert git(base,'rev-parse','HEAD').stdout.decode().strip()=='e6cb9a375b0bae15a430ae93b0913c02938bb45e'
assert not new.exists();p=git(base,'worktree','add','-b',branch,str(new),'e6cb9a375b0bae15a430ae93b0913c02938bb45e');assert p.returncode==0,p.stderr.decode(errors='replace')
for tag,head in [('p7','02c62ecd322eb423b78dd0ebe66d999e384db5dd'),('p4','cf5cf1005e75aedf3f766436b0412a053c7be19b')]:
 p=git(base,'diff','--binary','4379007a81d1bfc8751ebcf8dbb1143ba6460ed8',head);assert p.returncode==0;(B/('b21-'+tag+'-source.patch')).write_bytes(p.stdout)
p=git(new,'apply','--3way',str(B/'b21-p7-source.patch'));(B/'b21-p7-apply.log').write_bytes(p.stdout+p.stderr)
print(json.dumps({'worktree':str(new),'p7_apply_exit':p.returncode,'apply_output':(p.stdout+p.stderr).decode(errors='replace')}))
