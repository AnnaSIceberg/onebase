import pathlib,json,hashlib,datetime
B=pathlib.Path(__file__).parent
# Exact names/sizes from the read-only inventory before the authorized deletion.
inventory={
'reports-kpi-p1-b17-ui-linux-amd64.test':84196181,
'reports-p1-b16-ui-portable-linux-amd64.test':84146453,
'reports-p1-b16-ui-linux-amd64.test':84133773,
'audio-dirty-stage2-ff0518fd-linux-amd64.test':83981027,
'current-reference-b15-linux-amd64.test':83903864,
'reference-dirty-b14-linux-amd64.test':83804900,
'audio-diag-f49-stage1-linux.test':83782465,
'form-dirty-defaults-b13-linux-amd64.test':83752582,
'audio-viewstate-1114b10-final-ui-linux.test':83727586,
'onebase-1115b17-kpi-tabular-windows-amd64.exe':79289344,
'onebase-1115b16-tabular-reports-windows-amd64.exe':79259136,
'onebase-1115b15-current-reference-windows-amd64.exe':79106048,
'onebase-1115b14-reference-dirty-windows-amd64.exe':79063040,
'onebase-1115b13-form-dirty-defaults-windows-amd64.exe':79060992,
'onebase-1115b17-kpi-tabular-linux-amd64':78312262,
'onebase-1115b16-tabular-reports-linux-amd64':78282155,
'onebase-1115b15-current-reference-linux-amd64':78128686,
'onebase-1115b14-alldiff-diagnostic-linux-amd64':78113859,
'onebase-1114b12-audio-dirty-diagnostic-linux-amd64':78090981,
'onebase-1115b13-bid-dirty-diagnostic-linux-amd64':78089251,
'onebase-1115b14-reference-dirty-linux-amd64':78088665,
'onebase-1114b11-audio-service-state-linux-amd64':78084928,
'onebase-1115b13-form-dirty-defaults-linux-amd64':78083865,
'reports-p2p3p5p6-b16-api-linux-amd64.test':63371342,
'onebase-1114b10-audio-viewstate-final-windows-amd64.exe':61160960,
'onebase-1114b10-audio-viewstate-final-linux-amd64':60141730,
}
assert all(not(B/name).exists()for name in inventory)
recorded={}
def walk(obj,origin):
 if isinstance(obj,dict):
  p=obj.get('path');h=obj.get('sha256')
  if isinstance(p,str)and isinstance(h,str):
   name=p.replace('\\','/').rsplit('/',1)[-1]
   if name in inventory:recorded[name]={'sha256':h,'source_manifest':str(origin),'not_recomputed_after_deletion':True}
  for value in obj.values():walk(value,origin)
 elif isinstance(obj,list):
  for value in obj:walk(value,origin)
for p in set(B.glob('*manifest*.json'))|set(B.glob('*artifact.json'))|set(B.glob('*handoff.json')):
 try:walk(json.loads(p.read_text('utf-8-sig')),p)
 except (ValueError,UnicodeError):pass
preserved=[]
for name in ['manifest-1115b18-kpi-size-scale-DEV-final.json','manifest-1115b19-tabular-input-limit-DEV.json','manifest-1115b19-tabular-input-limit-DEV-final.json','onebase-1115b18-kpi-size-scale-linux-amd64','onebase-1115b18-kpi-size-scale-windows-amd64.exe','onebase-1115b19-tabular-input-limit-linux-amd64','onebase-1115b19-tabular-input-limit-windows-amd64.exe','reports-kpi-p1-b18-ui-linux-amd64.test','reports-kpi-p1-b19-ui-linux-amd64.test','onebase-1113b7-rest-guard-linux-amd64','onebase-1113b7-rest-guard-windows-amd64.exe','onebase-1114b8-effective-rights-linux-amd64','onebase-1114b8-effective-rights-windows-amd64.exe']:
 p=B/name
 with p.open('rb')as f:h=hashlib.file_digest(f,'sha256').hexdigest()
 preserved.append({'path':str(p),'bytes':p.stat().st_size,'sha256':h})
extra=[{'path':'C:/Users/a.sevrjukova/AppData/Local/Temp/onebase-2558-codex','before_bytes':1124559140,'after_bytes':1067023652}, {'path':'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-result-form-20261111','before_bytes':416063074,'after_bytes':260925976}]
total=sum(inventory.values())+sum(e['before_bytes']-e['after_bytes']for e in extra)
receipt={'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'status':'CLEANUP_COMPLETED_RECEIPT_RECOVERED','reason_receipt_recovered':'Authorized removal completed, then PowerShell Measure-Object could not read bytes on OrderedDictionary. No deletion repeated. Names/sizes recovered from pre-delete tool inventory; original hashes linked from retained manifests where available.','primary_removed_count':len(inventory),'primary_removed_bytes':sum(inventory.values()),'primary_files':[{'path':str(B/n),'bytes':v,'missing_after':True,'archived_hash':recorded.get(n)}for n,v in inventory.items()],'extra_old_directories':extra,'extra_directories_scope':'Only untracked MZ/ELF outputs removed. Sources/receipts/directories retained.','total_removed_bytes_from_inventory_and_tree_deltas':total,'total_removed_GiB':total/(1<<30),'C_free_observed_after':16233959424,'free_space_change_not_attributed_wholly_to_this_agent':True,'protected_preserved':preserved,'B18_B19_source_runtime_unchanged':True,'production_bb0269_6efa_not_in_selection':True,'no_recursive_delete':True,'no_process_stopped':True}
out=B/'20261115-platform-b10-b17-cleanup-recovered.json';assert not out.exists()
out.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'receipt':str(out),'sha256':hashlib.sha256(out.read_bytes()).hexdigest(),'removed_bytes':total,'GiB':total/(1<<30),'primary_files':len(inventory),'archived_hashes_recovered':len(recorded),'free_C_observed':16233959424}))
