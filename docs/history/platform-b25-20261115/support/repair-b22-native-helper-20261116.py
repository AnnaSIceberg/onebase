import hashlib
import pathlib

B = pathlib.Path(__file__).parent
original = B / 'b22-native-afterrelease.py'
assert hashlib.sha256(original.read_bytes()).hexdigest() == '9c115527a17efee15f9f30121a0f4453b8eb0df915a521cc3e5271c6469b01fc'
text = original.read_text('utf-8')
replacements = [
    ("W=B/'b22-native-acl-xlsx-attempt1'", "W=B/'b22-native-acl-xlsx-attempt2-20261116'"),
    (''' with page.expect_download(timeout=30000)as pending:page.locator('a[href^="/ui/report/B22Ordinary/excel"]').first.click()
 direct=W/'ordinary-direct-b22.xlsx';pending.value.save_as(direct);check_xlsx(direct,'direct')''',
     ''' direct_response=context.request.get(ORIGIN+'/ui/report/B22Ordinary/excel?Label=all')
 require('direct_public_route200',direct_response.status==200)
 direct=W/'ordinary-direct-b22.xlsx';direct.write_bytes(direct_response.body());check_xlsx(direct,'direct')'''),
    (" page.goto(ORIGIN+'/ui/report/B22Ordinary/export/excel?Label=all',wait_until='domcontentloaded');require('background_native_job_page','/ui/export-jobs/'in page.url);page.locator('a[href$=\"/download\"]').wait_for(state='visible',timeout=30000)",
     " page.get_by_role('link',name='Excel',exact=True).first.click();require('background_native_job_page','/ui/export-jobs/'in page.url);page.locator('a[href$=\"/download\"]').wait_for(state='visible',timeout=30000)"),
]
for old, new in replacements:
    assert text.count(old) == 1
    text = text.replace(old, new)
target = B / 'b22-native-afterrelease-attempt2-20261116.py'
assert not target.exists()
target.write_text(text, encoding='utf-8')
print(hashlib.sha256(target.read_bytes()).hexdigest())
