import pathlib
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
p=R/'20261116-platform-текущая-очередь-дата-уровни-отборы-меню.md'
s=p.read_text('utf-8').replace('Новая маска ещё не реализована и SDK не собирался.','Source маски реализован в Draft #2611: d570ea4a + исправление Ctrl/Meta+V после Escape 5fdabda1548dc5110aed662f69dc590614c66f41, оба pushed. Node29PASS. Первая браузерная попытка выявила ошибку вставки после Escape и неполную realtime-фикстуру; обе исправлены, повторная source проверка идёт отдельно. В runtime B29 маски пока нет, новый SDK не готов.')
s=s.replace('После маски/цифр, перед меню.','После маски/цифр. Ограничения ПДн применяются независимо от добавления новых отборов.')
a=s.index('4. **Дерево');z=s.index('5. П8',a)
s=s[:a]+'''4. **Связь параметров Филиал→Направление, сортировка/представление.** Заказ codex2 ожидается; после общих отборов.
5. **Движковая часть скрытия ПДн в отчётах.** Точный заказ прочитан: `codex2-20261116-заказ-platform-ПДн-представления-ссылок-B29.md`, repro codex2 f024f8ec / DSL afa3b77f. Её правило: клиентские телефоны и полные адреса скрыты для всех ролей, включая admin/disclose/read:full; безопасное представление ссылок сохраняет UUID/переход. Все каналы: HTML, расшифровки, tooltip, печать, XLSX/XML/relationships, PDF с учётом переносов, управляемые KPI. Не менять данные/карточки/обмен. B29 этого общего ограничения не имеет. Принят в очередь, реализация не начата.

**Дерево меню снято с очереди platform и передано platform2 (wV:p1).** Больше код/PR меню не меняю. Точные runtime B28/B29, SHA SDK, Draft #2636 и сохранённый WIP переданы файлом `20261116-platform-для-platform2-runtime-B28-B29-меню.md`.

6. П8'''+s[z+5:]
s=s.replace('6. Очередь владельца','7. Очередь владельца').replace('enum CI ещё отслеживается.','enum тест SQLite+PG PASS в CI, однако общий PG job завершён таймаутом UI-матрицы; полный GREEN не заявляется.')
s+='\nLedger PLAN v2: ответ `20261116-1504-platform-ledger-контракт-для-perenos-codex.md`; server ledger-gate в B26/B29 не реализован, существующие документы — SOURCE GAP PLAN. RO-сверка26 anchors и SHA приложены; WIP backup8bd65af7c pushed.\n'
p.write_text(s,encoding='utf-8')
print(str(p))
