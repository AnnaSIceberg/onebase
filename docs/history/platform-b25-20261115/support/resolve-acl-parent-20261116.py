import pathlib, re
p = pathlib.Path('C:/Users/a.sevrjukova/platform2-report-acl-source/CHANGELOG.md')
s = p.read_text(encoding='utf-8')
pattern = r'^<<<<<<< HEAD\n(.*?)^=======\n(.*?)^>>>>>>> bed39b9d6083c9b8d4bf9a9dce741dbd5fc05902\n'
s, n = re.subn(pattern, lambda m: m[1] + m[2], s, flags=re.M | re.S)
assert n == 1 and '<<<<<<<' not in s and '>>>>>>>' not in s
p.write_text(s, encoding='utf-8', newline='\n')
