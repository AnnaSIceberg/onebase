# Обычный YAML report → XLSX dimension: после выпуска

Принято отдельным послерелизным дефектом по поручению perenos. B20/current release не расширяется; Go/Chrome/HTTP/DB/SDK не запускались и не менялись.

Прочитан `perenos-codex/отчёты/20261115-1256-B20-обычный-отчёт-XLSX-dimension-A1.md`. Переданное доказательство на B20 Windows683984d7: обычный report export содержит28 XML-строк, но dimension=A1; default openpyxl read_only обрезает чтение, reset показывает строки/итоги. Файл chief-main.xlsx SHA `5d145ed83d8c163dd2d3b3d10f1f110edbd646d4bd34105fd9e2c1c0e3970cd7`. Сам файл/stand независимо не запускал и не менял.

B20 native34/defaultreader8 проверяли именно `sheet.Document.XLSX()` ТабличногоДокумента. Общий PASS обычного query/composition Excel writer не заявлялся и этим finding не подтверждается. Подтверждённую совместимость P1 нельзя переносить на другой writer.

После выпуска: обычный grouped/detail/totals report → штатный `/ui/report/<name>/excel`, фактический used range в ZIP/XML и default установленный reader без reset противnormal; проверить обычный writer отдельно, не менять исходные28строк/итоги ради oracle. До исправления perenos оставляет диагностический reset в helper с явной отметкой; пользовательский XLSX вручную не переписывается. Excel/LibreOffice UNKNOWN.

Generated-with: Codex
