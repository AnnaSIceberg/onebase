import pathlib,subprocess,json
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');base=B/'reports-p4-p7-b21-runtime';W=B/'report-source-filters-b23-runtime';branch='fix/report-result-source-filters-b23-20261115';H='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
assert not W.exists();assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=base,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()==H
p=subprocess.run(['git','-c','gc.auto=0','worktree','add','-b',branch,str(W),H],cwd=base,capture_output=True,creationflags=subprocess.CREATE_NO_WINDOW);assert p.returncode==0,p.stderr.decode(errors='replace');print(json.dumps({'base':H,'worktree':str(W),'branch':branch,'B22_not_included':True}))
