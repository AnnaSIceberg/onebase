import pathlib
import re
import subprocess

W = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/workbook-b27-runtime')

def git(*args):
    return subprocess.check_output(['git', '-c', 'gc.auto=0', *args], cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=90).decode('utf-8')

assert git('rev-parse', 'HEAD').strip() == '7b89c38bfab4d2aa2f2c8d1843fb66f83032cfd3'
source = git('show', '9b8db2dc37a4d799bfaaeef0cda1d0512168194b:internal/dsl/langref/methods.go')
begin = source.index('\t// ─── КнигаТабличныхДокументов')
end = source.index('\t// ─── ТабличныйДокумент', begin)
block = source[begin:end]
assert block.count('Kind: KindMethod, Object: "КнигаТабличныхДокументов"') == 3
file = 'internal/dsl/langref/methods.go'
current = git('show', ':2:' + file)
assert 'Object: "КнигаТабличныхДокументов"' not in current
anchor = current.index('\t// ─── ТабличныйДокумент')
(W / file).write_text(current[:anchor] + block + current[anchor:], encoding='utf-8')
for file, count in [('README.md', 1), ('QUICKSTART.md', 2)]:
    current = git('show', ':2:' + file)
    updated, replaced = re.subn(r'\d+ функций, \d+ метод(?:ов|а)', '187 функций, 172 метода', current)
    assert replaced == count
    (W / file).write_text(updated, encoding='utf-8')
# The current runtime has 11 additional functions and 3 additional methods;
# retain its complete reference as the generator's input/output baseline.
file = 'docs/dsl-reference.md'
(W / file).write_text(git('show', ':2:' + file), encoding='utf-8')
print('runtime descriptors preserved; only three workbook methods added; docs await native generator')
