import json, pathlib, subprocess
B = pathlib.Path(__file__).parent
gh = 'C:/Users/a.sevrjukova/gh-cli/bin/gh.exe'
def call(args, data=None):
 p = subprocess.run([gh,*args],input=data,stdout=subprocess.PIPE,stderr=subprocess.PIPE,creationflags=subprocess.CREATE_NO_WINDOW)
 assert p.returncode == 0, p.stderr.decode('utf-8',errors='replace')
 return json.loads(p.stdout)
p = call(['api','repos/ivanarama/onebase/pulls/2635'])
assert p['draft'] and p['state']=='open' and p['head']['sha']=='b7c57cb628abc493a17505172885c71f9854f0e6'
body = (B/'t8-pr-body-verified-20261116.md').read_text(encoding='utf-8')
p = call(['api','--method','PATCH','repos/ivanarama/onebase/pulls/2635','--input','-'],json.dumps({'body':body}).encode())
assert p['body']==body and p['draft']
print(json.dumps({'pr':2635,'body_updated':True,'draft':True}))
