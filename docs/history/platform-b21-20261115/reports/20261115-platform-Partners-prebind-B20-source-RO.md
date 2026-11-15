# Partners: prebind ДО migrate/seed — exact B20 source

Source e6cb9a375b0bae15a430ae93b0913c02938bb45e. Это консультация по коду: не читал БД/credentials, не запускал Go/CLI/schema/DDL, не менял shared/config. Actual типы/колонки/indices/row identity каждого целевого DB обязан сверить cc2 свежим preflight. Новый engine feature не нужен для этого одноразового prepare.

## Штатные пути и порядок

internal/storage/predefined.go:19–49: exported EnsurePredefinedColumns(ctx,entities) добавляет ТОЛЬКО служебные колонки + partial unique index для catalog с len(Predefined)>0. Сам по себе этот метод НЕ seed и НЕ привязывает existing row. Это Go API, отдельной CLI команды prebind/schema-only/no-seed в B20 нет.

internal/storage/migrate.go:564,695,778: Migrate -> migrateEntities -> EnsurePredefinedColumns внутри entity DDL scope -> ПОСЛЕ entity schema/index reconciliation SyncAllPredefined. Разрыва CLI между DDL и seed нет. cli/migrate.go:18–38 flags не содержат no-seed; --dry-run только preview, новые колонки не применяет. cli/migrate.go:217–229 applyAllMigrations вызывает Migrate. cli/run.go:260 run тоже вызывает Migrate; deploy.go:124 и142 вызывает Migrate и ещё SyncAllPredefined. Поэтому обычный migrate/run/deploy НЕ годится для «сначала schema, потом вручную bind».

predefined.go:143–166: findPredefinedByName ищет среди _is_predefined=TRUE/1; только _predefined_name EqualFold. При notfound выделяется uuid.New(). Existing строка с правильными кодом/наименованием, но без binding, автоматически НЕ принимается. Преждевременный seed может создать новый UUID/дубликат. predefined.go:803–835 возвращает отказ при неоднозначных casefold именах; index сам по себе exact-case, не EqualFold.

## Точные system types

| Поле | PostgreSQL | SQLite |
|---|---|---|
| _predefined_name | TEXT, nullable, без DEFAULT | TEXT, nullable, без DEFAULT |
| _is_predefined | BOOLEAN NOT NULL DEFAULT FALSE | INTEGER NOT NULL DEFAULT 0 |
| id (обычная ссылка) | UUID | TEXT |

Anchors dialect.go:92–101,160–169; ddl.go:38–43. Native partial unique index: CREATE UNIQUE INDEX ... ON table (_predefined_name) WHERE _is_predefined = TRUE (PG) / =1 (SQLite), predefined.go:38–44.

Физические имена derive из metadata.TableName/SQLIdent, types.go:965–969/sqlident.go:29. Для логического А_РекламаКаналы результат а_рекламаканалы; индекс idx_а_рекламаканалы_predefined — короткие имена <63UTF8bytes, hash truncation не требуется. Schema qualification взять из verified target, не из произвольного search_path. Existing index с тем же именем обязательно сверить definition/owner: IF NOT EXISTS не доказывает правильность. EnsurePredefinedColumns также вызывает indexName (legacy_idents.go:491), который может DROP legacy index своей таблицы; низкоуровневый helper не следует объявлять вообще не меняющим другие DDL.

## Минимальный prepare contract после резервной копии — НЕ ИСПОЛНЕН

Одна TX с остановленными target writers/seed. PG: native-compatible migration advisory xact lock из migration_lock.go:25–34, затем ACCESS EXCLUSIVE TABLE lock. SQLite: отдельный controlled connection BEGIN IMMEDIATE + target offline; не приписывать ему native dblock lock без реального вызова. Lock/timeouts и schema/DB identity задаёт owner target.

1. Проверить существующую таблицу/PK id и типы имеющихся system columns; несовместимые типы/nullable/default => STOP, не cast/перестраивать молча.
2. Добавить ТОЛЬКО отсутствующие _predefined_name и _is_predefined с типами/default выше. PG ADD COLUMN IF NOT EXISTS; SQLite сначала PRAGMA table_info, затем ADD COLUMN в той же TX — SQLite не поддерживает ADD COLUMN IF NOT EXISTS. Не запускать общий Migrate для этого шага.
3. Под table lock/той же snapshot подтвердить ровно1 строку по заранее утверждённому GUID, ожидаемым code/name и остальным identity guards manifest. Не искать/UPDATE по имени вместо GUID. Снять before-image двух system полей; обе незаполнены/непривязаны ИЛИ уже та же корректная пара Partners+true. Чужая привязка => STOP.
4. Проверить casefold Partners по ВСЕМ строкам (включая ещё не flagged), exact/casefold дубли любых уже flagged names и конфликт index name. Иная Partners-строка => STOP, не удалить/слить/переназначить. Для ASCII Partners LOWER достаточно для этого имени; для общей проверки native EqualFold нельзя выдавать SQL locale LOWER за доказанно эквивалентный.
5. Создать/сверить native partial unique index из предыдущего раздела. Обновить ONLY target id с CAS old system values: SET _predefined_name=:approved_predefined_name, _is_predefined=TRUE/1 WHERE id=:approved_existing_uuid AND <точные before-image guards>. affected rows ровно1, либо подтверждённый idempotent same row. Имя binding брать из exact accepted YAML predefined item, Partners здесь ожидаемый контракт, не строка из UI.
6. До COMMIT проверить: binding count1, GUID тот же; все прочие столбцы target row и все иные строки/UUID/business counts неизменны. Ошибка/гонка/неточная identity => ROLLBACK. Receipt aggregate/schema/binding-only, без business payload.

DDL shapes (для review, placeholders требуют адаптера owner; не готовый автоисполняемый файл):

    PG: ALTER TABLE <verified_schema>."а_рекламаканалы" ADD COLUMN IF NOT EXISTS "_predefined_name" TEXT;
        ALTER TABLE <verified_schema>."а_рекламаканалы" ADD COLUMN IF NOT EXISTS "_is_predefined" BOOLEAN NOT NULL DEFAULT FALSE;
    SQLite (только если PRAGMA подтвердил отсутствие):
        ALTER TABLE "а_рекламаканалы" ADD COLUMN "_predefined_name" TEXT;
        ALTER TABLE "а_рекламаканалы" ADD COLUMN "_is_predefined" INTEGER NOT NULL DEFAULT 0;

Этот шаг привязывает существующий UUID; он не создаёт application objects, не прогоняет hooks/модули/seed/roles. После COMMIT и отдельного owner approval обычный migrate сможет reuse GUID. Но SyncPredefined всё равно обновляет item.Fields из YAML (predefined.go:238–299), поэтому binding НЕ гарантия неизменности всех business полей последующего seed: заранее сопоставить accepted YAML item.Fields с существующей строкой и отчётностью. Привязка сохраняет identity, не отменяет остальные migration effects.

## Что не доказано

Не выполнялись backup/actual prebind/CLI/lookup и не проверялась свежая live schema. Report фактов ЛПЦ подтверждает GUID/code/name источника/цели и отсутствие _predefined_name, но не заменяет свежие owner assertions. Никакой production/DEV DDL отсюда не применён. Это минимальный source-derived prepare proposal, root решает выполнение в своей согласованной очереди.

Generated-with: Codex
