import datetime,hashlib,json,pathlib,subprocess,sys
B=pathlib.Path(__file__).parent
tasks=json.loads((B/'b28-native-plan-20261116.json').read_text('utf-8'))
out=B/'b28-native-combined-20261116.json';assert not out.exists()
receipt={'started':datetime.datetime.now(datetime.timezone.utc).isoformat(),'runtime':'f045bdb11b5e563120ff0381476957e5df9e2549','serial_browsers':True,'go_started':False,'runs':[]}
for t in tasks:
 helper=pathlib.Path(t['helper']);assert hashlib.sha256(helper.read_bytes()).hexdigest()==t['helper_sha256']
 log=B/(helper.stem+'.log')
 with log.open('xb') as f:p=subprocess.run([sys.executable,str(helper)],stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=300)
 r=json.loads(pathlib.Path(t['receipt']).read_text('utf-8'))
 entry={'helper':str(helper),'exit':p.returncode,'receipt':t['receipt'],'receipt_sha256':hashlib.sha256(pathlib.Path(t['receipt']).read_bytes()).hexdigest(),'status':r['status'],'checks':len(r['checks']),'cleanup':r['cleanup'],'runtime':r['runtime']}
 receipt['runs'].append(entry);out.write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8');print(json.dumps(entry),flush=True)
 assert p.returncode==0 and r['status']=='PASS' and all(r['checks'].values()) and all(r['cleanup'].values()) and r['runtime']==receipt['runtime']
receipt.update(status='PASS',checks=sum(t['checks'] for t in receipt['runs']),ended=datetime.datetime.now(datetime.timezone.utc).isoformat())
out.write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8');print(json.dumps({'status':'PASS','checks':receipt['checks']}))
