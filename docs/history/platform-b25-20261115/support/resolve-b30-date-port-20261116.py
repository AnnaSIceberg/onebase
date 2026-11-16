import pathlib,re,subprocess
W=pathlib.Path(__file__).parent/'report-b30-date-mask-runtime'
for name,old,new in [('docs/features.md','## Компактные отчёты и сокращённый ввод дат','## Компактные отчёты и маска ввода даты'),('docs/forms.md','### Сокращённый клавиатурный ввод даты','### Маска ввода даты и времени'),('internal/ui/templates.go',None,None)]:
 p=W/name;s=p.read_text('utf-8')
 matches=list(re.finditer(r'^<<<<<<< HEAD\n(.*?)^=======\n(.*?)^>>>>>>> [^\n]*\n',s,re.M|re.S));assert len(matches)==1
 m=matches[0];ours,theirs=m.group(1,2)
 if old:
  assert ours.endswith(old+'\n') and theirs==new+'\n';value=ours[:-len(old+'\n')]+new+'\n'
 else:
  a='(dict "invalid" (t $.Lang "Введите корректную дату"))'
  v='(dict "invalid" (t $.Lang "Введите корректную дату") "incompleteDate" (t $.Lang "Введите дату полностью: ДД.ММ.ГГГГ") "incompleteDatetime" (t $.Lang "Введите дату и время полностью: ДД.ММ.ГГГГ ЧЧ:ММ:СС") "calendar" (t $.Lang "Открыть календарь"))'
  assert ours.count(a)==1 and v in theirs;value=ours.replace(a,v)
  assert '{{statusBarMeta .}}' in value and '{{if .Cfg.UIEnhancements}}' in value
 s=s[:m.start()]+value+s[m.end():];assert not re.search(r'^(<<<<<<<|=======|>>>>>>>)',s,re.M)
 p.write_text(s,encoding='utf-8')
 print(name)
