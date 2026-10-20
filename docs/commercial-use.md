# Коммерческое использование и изменённые сборки OneBase

OneBase распространяется по MIT; полный текст находится в файле `LICENSE`.
Эта лицензия разрешает
использование, изменение, распространение и продажу копий при соблюдении её
условий, включая сохранение уведомления об авторских правах и текста разрешения.
Сам текст MIT не требует публикации собственных доработок или бесплатного
распространения производного продукта.

Собственная конфигурация и собственные доработки могут иметь выбранные вами
условия распространения. Они не отменяют условия лицензий заимствованных
компонентов. Разрешение использовать OneBase не означает передачу исключительных
прав на исходную платформу, сторонние библиотеки или чужой вклад.

## Документы поставки

Для проверки конкретной поставки нужны версия и исходная ревизия, перечень
применяемых компонентов и их версии, полные тексты лицензий и необходимые
уведомления правообладателей. Состав серверной и GUI-сборки может различаться.
Зависимости Go, встроенные браузерные библиотеки, шрифты и нативные компоненты
проверяются отдельно. Перечень компонентов сам по себе не заменяет их лицензии.

## Название и ответственность за изменённую сборку

Изменённую сборку следует обозначать так, чтобы пользователь мог определить
её поставщика и отличить её версию от версии исходной платформы. У конфигурации
есть отдельные свойства `name`, `version`, `author`, `copyright`, `license` и
`support` в `config/app.yaml`; они не меняют авторство платформы.

Поставщик изменённой сборки определяет собственные условия поддержки и обновления.
Указание названия форка само по себе не настраивает канал обновлений и не означает
поддержку этой сборки сопровождающими исходного OneBase.

## Дополнительное подтверждение автора

Подписанное подтверждение может дополнительно идентифицировать конкретный
релиз, ревизию и текст разрешения. Наличие PDF или электронной подписи не
является условием самой MIT. Русский перевод условий служит пояснением;
канонический текст MIT сохраняется на английском языке.

Для отдельного подтверждения заранее сообщите, кто его запрашивает, какой объект
и какую версию вы оформляете, какие сведения и вид подписи требуются. Подпись
одного автора не заменяет разрешения остальных правообладателей. Достаточность
документов для конкретной регистрационной процедуры проверяется отдельно.

Источники: [MIT](https://opensource.org/license/mit),
[ГК РФ, часть IV, ст. 1286.1](https://rospatent.gov.ru/ru/documents/grazhdanskiy-kodeks-rossiyskoy-federacii-chast-chetvertaya),
[обсуждение #1890](https://github.com/ivanarama/onebase/discussions/1890).

---

# Commercial use and modified distributions

OneBase is distributed under the MIT license in `LICENSE`. Subject to its terms,
including retention of the copyright and permission notices, MIT permits use,
modification, redistribution, sublicensing and sale. MIT itself does not require
publication of your changes or free distribution of a derived product.

Your configuration and your own changes may have your chosen distribution terms.
These terms do not replace the applicable licenses for upstream code and
third-party components or transfer their exclusive rights to you.

Identify the exact version and source revision, retain the component inventory,
full license texts and required notices for that distribution. Check Go
dependencies, embedded browser assets, fonts and native GUI components; an
inventory alone does not replace their license terms.

Identify a modified distribution's name, version and supplier separately from
the upstream platform and the application configuration. Configuration metadata
lives in `config/app.yaml`. Distribution metadata does not configure an update
channel or imply support by the upstream maintainers.

A separately signed confirmation can identify a specific release, revision and
license text. Neither a PDF nor a signature is a requirement of MIT itself.
Specify the requesting party, registration procedure, exact work/version and
required signature format before requesting such a document. A signer's
confirmation does not replace third-party permissions or guarantee registration.
