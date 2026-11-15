from pathlib import Path
B=Path(__file__).parent
s=(B/'b19-native-input-limit.py').read_text('utf-8')
s=s.replace("W=B/'b19-native-input-limit-attempt1'","W=B/'b19-native-input-limit-attempt2'")
s=s.replace("def documents():","def native_wire_count(method,status):\n text=Path(serverlog.name).read_text('utf-8')\n return sum('method='+method+' ' in line and '/ui/report/NativeTabular' in line and 'route=/ui/report/{name} ' in line and 'status='+str(status)+' ' in line for line in text.splitlines())\ndef documents():")
s=s.replace("from playwright.sync_api import sync_playwright","from playwright.sync_api import sync_playwright\nfrom pathlib import Path")
s=s.replace("before=len(responses);r=report_request('NativeTabular',method)\n page.locator", "before=len(responses);wire_before=native_wire_count(method,303);r=report_request('NativeTabular',method)\n page.locator")
s=s.replace("require(case+'_post303'if method=='POST'else case+'_get303',any(x['method']==method and x['status']==303 and x['redirect_to_viewer']for x in responses[before:]))","require(case+'_native_wire303',native_wire_count(method,303)==wire_before+1)")
s=s.replace("'redirect':303,'viewer_http':200,'raw_input_rows':8", "'native_server_redirect':303,'browser_terminal_response':r.status,'viewer_documents_actual':len(docs),'raw_input_rows':8")
s=s.replace("before=len(responses);r=report_request('NativeTabular',method,query,guard)","before=len(responses);wire_before=native_wire_count(method,303);r=report_request('NativeTabular',method,query,guard)")
s=s.replace("require(case+'_no_viewer_published',not any(x['kind']=='viewer'or x['redirect_to_viewer']for x in responses[before:]))", "require(case+'_no_viewer_published',native_wire_count(method,303)==wire_before and not any(x['kind']=='viewer'or x['redirect_to_viewer']for x in responses[before:]))")
s=s.replace("receipt['browser_version']=browser.version", "receipt['browser_version']=browser.version\n receipt['observer_correction']='Attempt1 expected raw303 in Playwright response event; native server journal proves POST303 followed by viewerGET200 while old Chrome observer reports terminal200. Attempt2 checks native wire303 count and actual seven-document viewer; product/SDK/config fixture unchanged.'")
p=B/'b19-native-input-limit-attempt2.py';assert not p.exists();p.write_text(s,encoding='utf-8')
