import pathlib,re,subprocess
W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116')
def git(*a):return subprocess.check_output(['git',*a],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW)
p=W/'CHANGELOG.md';s=p.read_text('utf-8');m=re.search(r'^<<<<<<< HEAD\n(.*?)^=======\n(.*?)^>>>>>>> origin/main\n',s,re.M|re.S);assert m
ours=m.group(1)
up=git('show','origin/main:CHANGELOG.md').decode('utf-8')
placeholder='Изменения после v0.14.0 будут перечисляться здесь.'
assert up.count(placeholder)==1 and 'ui.short_date_input: true' in ours
p.write_text(up.replace(placeholder,ours.rstrip()),encoding='utf-8')
assert not re.search(r'^(<<<<<<<|=======|>>>>>>>)',p.read_text('utf-8'),re.M)
