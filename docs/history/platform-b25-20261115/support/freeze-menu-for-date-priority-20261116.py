import datetime,hashlib,json,pathlib,subprocess
B=pathlib.Path(__file__).parent;W=pathlib.Path('C:/Users/a.sevrjukova/platform-recursive-menu-20261116')
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
def git(*args):return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW)
assert git('rev-parse','HEAD').decode().strip()=='3c88d2368843da899665a94d8113ef29eee9d715'
files=['internal/metadata/menu.go','internal/navigation/tree.go']
assert set(git('diff','--name-only').decode().splitlines())==set(files)
raw=git('diff','--binary');assert raw
patch=B/'menu-recursive-first-model-WIP-20261116.patch';assert not patch.exists();patch.write_bytes(raw)
record={'head':git('rev-parse','HEAD').decode().strip(),'draft':2636,'worktree':str(W),'files':files,'patch':str(patch),'patch_sha256':hashlib.sha256(raw).hexdigest(),'status':'INCOMPLETE_MODEL_NO_SDK_NO_GO_CHECKS; retained while date mask takes priority'}
receipt=B/'menu-first-model-WIP-20261116.json';assert not receipt.exists();receipt.write_text(json.dumps(record,indent=2)+'\n',encoding='utf-8')
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M');report=R/(tag+'-platform-приоритет-маска-даты-меню-сохранено.md')
report.write_text(f'''# Приоритет: маска даты, затем дерево меню

Её заказ принят: ввод цифр в `__.__.____` / `__.__.____ __:__:__`, курсор, Backspace/Delete, вставка с разделителями и без, понятная ошибка неполного значения. Enter не формирует отчёт. Область — формы объектов, параметры отчётов, отборы списков, включая операторов КЦ только в части даты. Draft upstream-first, browser acceptance, накопление SDK в B26→B29+, пакетная выкладка в ЛПЦ :8448 только по её слову; :8444/:8443 без её слова не менять.

Enum-заказ завершён до этого переключения: B29 `902e4d97e`, source Draft #2637 `2e185f061`, native38PASS/JS0/cleanupalltrue, отчёт `20261116-1403-platform-для-cc2-enum-B29-native38-SDK.md`. SDK не установлен. Независимый PG312 и ещё идущий source CI остаются открытыми.

Меню Draft #2636 / branch `feature/recursive-command-menu-20261116` / HEAD `{record['head']}` содержит только проект контракта. Начатая модель из двух файлов ещё не завершена, не проверена и не входит SDK. Сохранена точным patch `{patch}`, SHA `{record['patch_sha256']}`; квитанция `{receipt}`. После remote backup рабочая копия будет очищена только от этих двух собственных правок. Задача меню не считается выполненной и следует после маски даты.

Место продолжения меню: сохранить старые Tree v1/JSON/hash/операторские настройки; новый рекурсивный формат `nodes` — Tree v2, единый смешанный порядок. В normalized Node хранить исходные наследуемые role_clauses (перестановка не снимает ограничение роли), typed target/Object только на листьях. Обобщить delta layout/parent-cycle-depth checks/Diff/round-trip и server role projection, settings editor/preview и рекурсивный renderer. Свёртывание — user+subsystem+stable folder ID, native details/keyboard/aria-expanded; автоматическое раскрытие активных предков не записывает предпочтения соседних ветвей. Не ограничиваться только добавлением поля Nodes: фиксированные обходы есть во всех этих слоях. Никаких новых агентов/общих выкладок/рекурсивного поиска C:.

Generated-with: Codex
''',encoding='utf-8')
paths=[report,patch,receipt,pathlib.Path(__file__)]
lst=B/('support-'+tag+'-files.json');assert not lst.exists()
lst.write_text(json.dumps([{'source':str(p),'target':'reports/'+p.name if p.parent==R else 'support/'+p.name}for p in paths],ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'tag':tag,'list':str(lst),'report':str(report),'patch_sha256':record['patch_sha256']}))
