import pathlib,json,hashlib
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
def load(name):return json.loads((B/name).read_text('utf-8-sig'))
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
old=load('20261115-platform-b10-b17-cleanup-recovered.json');cache=load('20261115-platform-build-cache-cleanup-b19.json')
out=R/'20261115-platform-очистка-диска-B10-B17.md'
out.write_text(f'''# Очистка собственного Temp: выполнено

Старые B10–B17 и промежуточные binaries: удалено **{old['total_removed_bytes_from_inventory_and_tree_deltas']}B (~{old['total_removed_GiB']:.2f}GiB)**. Это26 артефактов из platform-help-admin + untracked MZ/ELF outputs старых onebase-2558-codex и platform-result-form. Чистились только файлы после literal/root/reparse/active-process/Git-tracked проверок. Исходники, Git worktrees и manifest/квитанции сохранены; остальные три перечисленных старых root-пути не найдены.

Само удаление завершилось, но итоговый Measure-Object не прочитал bytes у OrderedDictionary. Повторного удаления не было. Квитанция восстановлена по исходной tool-инвентаризации имён/размеров, архивным manifest/SHA и before/after размерам двух собственных старых каталогов; все26 архивных SHA найдены. Это честно отмечено в recovered receipt, не выдаётся за сохранённый до удаления plan.

Сохранность B18/B19 SDK/prebuilt/manifest и B7/B8 повторно SHA-проверена. Рабочие bb0269/6efa не входили в выборку. Сами source worktrees старых задач не удалялись: их код/квитанции не считались мусором автоматически.

Receipt `{B/'20261115-platform-b10-b17-cleanup-recovered.json'}` SHA `{sha(B/'20261115-platform-b10-b17-cleanup-recovered.json')}`. После измерено Cfree{old['C_free_observed_after']}B (~{old['C_free_observed_after']/(1<<30):.2f}GiB); весь прирост свободного места этой очистке не приписывается.

Ранее в том же окне при Go0 до/после pinned `go clean -cache` освободил{cache['removed_file_bytes']}B (~{cache['removed_file_bytes']/(1<<30):.2f}GiB). Modcache, SDK, source, DB и квитанции не тронуты. Receipt `{B/'20261115-platform-build-cache-cleanup-b19.json'}` SHA `{sha(B/'20261115-platform-build-cache-cleanup-b19.json')}`.
''',encoding='utf-8')
print(out)
