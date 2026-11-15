# B20 combined: XLSX dimension + result_proc input — native/reader PASS

Runtime `e6cb9a375b0bae15a430ae93b0913c02938bb45e`, source `d232937812637564c730fbc17a9376d269dc5247`, exactB19 parent `cd9910c81fd7812874b9ca2cafa1870857abee5e`. Windows SDK `683984d7a6b7342d20124dd295cc94bc8be1f9ee9e38e50efd3404228ca70c07`, Chromium `109.0.5414.46`. Linux SDK `185925eff94b53424eb2a5e2c08d62bb98222b900d35d5761d313d0f914d559a` отдельно; данный native proof — Windows, не Linux.

**34 native checks PASS/JS0**, fresh ownSQLite/auth удалена,8НОВЫХexports. Actual XLSX ZIP dimension: largeA1:BO830/55610cells/1655merges;autoA1:BO51/3417cells/97merges;sparse/style-tailA1:BO830;offsetspanA1:H5;collapsedA1:D3;single/emptyA1. Model/style/merge хвосты не потеряны. Source layout512146a7 и исходный модуль такие же, как вB18; добавлены только собственные syntheticbuttons. Пустой/обычный/sparse/span/style-only/collapsed/auto иKPI проверены, не подменены одним dense тестом.

Installed Python3.14.3/openpyxl3.1.5: **default read_only=True, безreset/forcedbounds**, противnormal на этих новыхфайлах. ВоВСЕХ8 counts, заполненные координаты и типизированные значения равны; empty0rows/0positions/0values вОБОИХ, заранее разрешённое исключение не понадобилось.

| Новый файл | Строк | Колонок | Позиций | Непустых значений | Default vs normal |
|---|---:|---:|---:|---:|---|
| auto51-b20.xlsx | 51 | 67 | 3417 | 141 | PASS |
| collapsed-b20.xlsx | 3 | 4 | 12 | 12 | PASS |
| empty-b20.xlsx | 0 | 1 | 0 | 0 | PASS |
| large53893-b20.xlsx | 830 | 67 | 55610 | 141 | PASS |
| plain-b20.xlsx | 1 | 1 | 1 | 1 | PASS |
| span-b20.xlsx | 5 | 8 | 40 | 1 | PASS |
| sparse-b20.xlsx | 830 | 67 | 55610 | 1 | PASS |
| styled-b20.xlsx | 830 | 67 | 55610 | 1 | PASS |

У исходной KPI-геометрии141значение — строки; это не доказательство числовой бизнес-полноты KPI/1С. Отдельные plain/sparse/styled controls имеют настоящий int42 и сохраняют тип. Проверена конкретная openpyxl3.1.5 совместимость; Excel/LibreOffice/прочие readers UNKNOWN, pandas/reset не использовались как acceptance.

Go/source:17PASS/1localPGSKIP, oldwriter7expectedRED; runtime212localleafPASS/10localPGrootsSKIP; build./.../i18nen PASS. Independent prebuilt66SQLite+66PG=132exactleafPASS/FAIL0/SKIP0/cleanup exact, SHAreceipt `a7d1bf16b6ed62913e20b05bde838b3b85c680778fcd40a29cd65ae43a2f4812`. Наследуемый B19 positive-limit422/ordinary-warning контроль повторён вPG132; unrelatedB18native/API6 не повторялись.

Native receipt `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b20-native-dimension-attempt1\receipt.json` SHA `d2bb0d9f15cd9b863432ec338b4b0e318986827d983a2523e4fe40a918b111a4`. Installed reader `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b20-native-dimension-attempt1\installed-default-reader.json` SHA `6f5b1306f34e17387bef240b4179144a6f9966dcac9183bb342a4108f5247d36`. Fixture `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b20-native-dimension-attempt1\source-fixture-manifest.json` SHA `1e53f00dbd3fa18fee5616f192ff99198769a0262796939a090d821562e65b07`.4PNG `C:\Users\a.sevrjukova\Desktop\ТабДок-B20-XLSX-dimension-20261115`, fullSHAs вsupplement. OriginalB20 buildmanifestf98dd30e и B18/B19 artifacts неизменны, старыеxlsx42abc/4c8a повторноSHAverified.

**Final supplement** `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-1115b20-xlsx-dimension-DEV-final.json` SHA `f03a5d04a473f2978927bbc32485bc8b4ba422bb4d117be40b408d8fb6b8946b`. **EXPLICIT BROWSER/APP/PORT/TUNNEL END**: CIMGo/compiler/vet/Chrome/driver/nodeplaywright0,18570closed, ownAuthDBremoved. :8444/shared/config/production не менялись. Её новое слово — подготовитьобщийвыпуск3базк17:30, НЕИСПОЛНЯТЬ до вечернего слова; B20 — tested base дляпоследующих reviewedchanges.17:45freshreboot checkpoint/cleanpushedWIP остаётся обязательным.
