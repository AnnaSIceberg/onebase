import hashlib, json, pathlib, subprocess, sys
B=pathlib.Path(__file__).parent
M=json.loads((B/'b28-reports-build-20261116/manifest.json').read_text('utf-8'))
assert M['status']=='BUILD_AND_TARGETED_TESTS_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
M.update(source=M['runtime'],source_t8='b7c57cb628abc493a17505172885c71f9854f0e6',base_b21='0d77e91557b12bada4668fd7bfa5f1b13a37de08',base_b26='513274b7e1bc230d8c0de5fcd22a34deb672345c',upstream_inputs={'p4':'892365a03c8c35e3c48dea25ed897a3c9a072d95','p7':'449bafdb8a8c86f177e8dbbe768a62af885b2ab0','acl':'6ac8da0fa9bab99c57078cfc04d6235070097cd7','ordinary_xlsx':'54a8303aa668dbe35c577ec0a372bf25267b347c'})
out=B/'manifest-b28-native-input-20261116.json'
assert not out.exists();out.write_text(json.dumps(M,indent=2)+'\n',encoding='utf-8')
tasks=[
 ('b22-native-afterrelease-attempt3-20261116.py','native-b28-acl-xlsx-20261116.py','b28-native-acl-xlsx-attempt1-20261116', [('manifest-1115b22-report-acl-list-xlsx-DEV.json',out.name),('b22-native-acl-xlsx-attempt3-20261116','b28-native-acl-xlsx-attempt1-20261116')]),
 ('b21-native-context-drilldown-attempt3.py','native-b28-context-drilldown-20261116.py','b28-native-context-drilldown-attempt1-20261116',[('manifest-1115b21-report-context-drilldown-DEV.json',out.name),('b21-native-context-drilldown-attempt3','b28-native-context-drilldown-attempt1-20261116')]),
 ('native-b27r2-workbook-20261116.py','native-b28-workbook-20261116.py','b28-native-workbook-attempt1-20261116',[('b27r2-workbook-build-20261116/manifest.json',out.name),('BUILD_AND_TARGETED_TESTS_PASS_SDK_READY_NATIVE_PENDING',M['status']),('b27r2-native-workbook-attempt1-20261116','b28-native-workbook-attempt1-20261116')])]
inventory=[]
for source,target,directory,replacements in tasks:
 raw=(B/source).read_text('utf-8')
 for old,new in replacements:
  assert raw.count(old)==1,(source,old,raw.count(old))
  raw=raw.replace(old,new)
 target=B/target;assert not target.exists();target.write_text(raw,encoding='utf-8')
 inventory.append({'helper':str(target),'source_helper':source,'source_sha256':hashlib.sha256((B/source).read_bytes()).hexdigest(),'helper_sha256':hashlib.sha256(target.read_bytes()).hexdigest(),'receipt':str(B/directory/'receipt.json')})
(B/'b28-native-plan-20261116.json').write_text(json.dumps(inventory,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'helpers':len(inventory),'runtime':M['runtime']}))
