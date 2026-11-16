import pathlib
B=pathlib.Path(__file__).parent;W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116')
p=W/'internal/ui/static/date_input_behavior_test.js';old=p.read_bytes();backup=B/'date-input-old-shortcuts-tests-20261116.js'
assert not backup.exists();backup.write_bytes(old)
p.write_bytes((B/'date-mask-node-tests-candidate-20261116.js').read_bytes())
print('updated DOM-event regressions for explicit full-mask contract; previous tests retained')
