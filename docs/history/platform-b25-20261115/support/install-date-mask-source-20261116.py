import pathlib,subprocess
B=pathlib.Path(__file__).parent;W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116')
head=subprocess.check_output(['git','rev-parse','HEAD'],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
assert head=='0f61dc5fcfa7f1f4d716496a4517ac8cd08a7582'
target=W/'internal/ui/static/date-input.js';assert target.read_text('utf-8').startswith('/* Optional short calendar-date entry;')
target.write_bytes((B/'date-mask-candidate-20261116.js').read_bytes())
print('date mask installed in existing Draft #2611 source worktree; not yet tested')
