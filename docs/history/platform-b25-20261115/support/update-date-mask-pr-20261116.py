import json,pathlib,subprocess
B=pathlib.Path(__file__).parent
body=(B/'date-mask-pr-body-20261116.md').read_text('utf-8')
payload=json.dumps({'title':'feat(ui): компактные отчёты и маска ввода даты без неявной отправки','body':body},ensure_ascii=False).encode('utf-8')
p=subprocess.run(['C:/Users/a.sevrjukova/gh-cli/bin/gh.exe','api','--method','PATCH','repos/ivanarama/onebase/pulls/2611','--input','-'],input=payload,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW)
assert p.returncode==0,p.stderr.decode('utf-8','replace')
out=json.loads(p.stdout);assert out['draft'] and out['body']==body
print(json.dumps({'url':out['html_url'],'head':out['head']['sha'],'draft':out['draft']}))
