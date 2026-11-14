# Stage2 private AudioPlay observation — CODE REVIEW ONLY

This is a test-only experiment on exact runtime
`f49a9e0bfd2eb5184ce9e13e583a5a163e967534`, based on Stage1
`6b34ca82935da6009296b5b09c4996faadef9396`. No product hook, SDK,
configuration edit, automatic correction, startup migration, seed or role sync.
**Implementation has not been compiled or executed.** The three-file diff is
submitted for review before a separate Go window and a later owner-runner/browser
window. A proof file saying `execution_approved` never grants that permission.

The actual B10 native AFTER is still RED. Stage1 SQLite12+PG12 PASS did not
reproduce it. Neither Stage1, source fixtures nor this proposed observer close
the actual failure. B10's live changed-values diff was not recorded; the earlier
"only source changed" fact belongs to B9, not B10.

## Files and entry points

- `../../audio_dirty_stage2_private_test.go`: Linux-only opt-in
  `^TestAudioDirtyStage2Private$`. No PG subtest or expected PASS count: this
  observes one private full-config SQLite header form per explicit mode.
- `capture.py`: import-only synchronous
  `capture_authenticated(*, page, context, origin, target_binding) -> None`.
  Main exits 2 without actions. No import-time I/O, coroutine, process/browser
  launch, login, file reads, credentials/cookies/storage_state, response body,
  post_data, screenshot/HAR/trace, raw exception output or auth injection.
- This README is the schema, allowlist, native mapping and STOP contract.

No ready-to-execute command, runner, input binding, database or credentials are
created by this diff. The default test path skips before opening any input.
Future opt-in names are `ONEBASE_AUDIO_STAGE2_PRIVATE=1` and
`ONEBASE_AUDIO_STAGE2_INPUT=<reviewed private JSON file>`; a supervisor must
add an exact test regex, timeout and private output limits after authorization.

## Owner login boundary

The reviewed coordinator is external to this branch:
`stage2_login_coordinator.py`, SHA
`e98b24eafb0d580252105cd9441b64c218e0c773859c0155de1cc56a48246e45`;
contract SHA
`9762798c02f92519ef9b5ac2d701e55be485910200875ce1e4eb0d8c9797c708`.
Its previous source `after_b10_browser.py` SHA
`e1b6dffeb3d601c022bc7b43551d9592859499775f6fe7ac23aa95c71835a8eb`
is provenance only: this experiment does not read, import or run that harness.

cc2b's future reviewed runner owns browser creation/finally. The coordinator
owns native DOM login and context.close finally. Only after login does it call
capture with SyncPlaywright objects, one page/context and an immutable
MappingProxyType containing exactly `entity` and `id`. These values remain
inside the owner process. Capture checks its exact allowed origin and native
`/ui` entry, then opens the own header. No direct event POST or FormData
fabrication: native GET/OnOpen/deferred requests and native Play make requests.
The backend uses the real session, auth, CSRF and public API/UI middleware.

The callback reads exactly the non-secret `X-OneBase-UI-Commands` negotiation
header and boolean dirty/trusted-input counters. It never reads all headers,
cookies, input values, credentials, storage state or raw response body. It
asserts clean-before-Play and no trusted input/change; either post-Play dirty
boolean is a fact, not a forced expectation. Client observations are assertions
only; authoritative server evidence is the flags collector. It removes its
own listeners/counter in finally, does not Close/Save/edit/retry Play, and does
not close resources it does not own. No callback result data are exported.

## Input schema v1 (private, strict)

All JSON proof/binding/input files are regular owner-UID files, mode0600,
nlink1, at most2MiB, no unknown keys/trailing JSON. Work root is owner0700,
no symlinks, `/home/onebase/platform-audio-stage2-20261114-<8hex>`.
`Pin` below means `{ "path": string, "sha256": lowercase64hex }`.
Pointers in Go deliberately distinguish missing booleans/integers from false/0.
Placeholders below are a schema description, **not filled input**.

```text
schema: "onebase.audio.stage2.private/v1"
base_runtime: "f49a9e0bfd2eb5184ce9e13e583a5a163e967534"
executable_sha256: Pin SHA of the future TEST prebuilt (not SDK199)
work_root: WORK
explicit_mode: "strict_false" | "strict_true"
original_strict_rls_parity: "UNKNOWN"
mode_provenance: Pin to WORK private proof (schema below)
ownership_preflight: Pin to WORK copy of reviewed d8c0c41b...82e1fe report
source_manifest: Pin to SOURCE/after-b10-manifest.json
copy:
  source_db: SOURCE/after-b10.sqlite
  source_db_sha256: private owner attestation (never emitted)
  database: WORK/after-b10.sqlite
  project: WORK/config-after-b10
  media: WORK/.onebase/files/after-b10
  fresh_copy: true
  source_wal_inactive: true
  source_immutable: true
  own_synthetic_verified: true
  media_files: { private relative filename: private SHA, ... }
private_bindings: Pin to WORK private binding (schema below)
login:
  existing_source: {path: owner Windows after_b10_browser.py, sha256: e1b6dffe...}
  reviewed_adapter: {path: owner Windows stage2_login_coordinator.py, sha256: e98b24ea...}
  capture_sha256: reviewed capture.py SHA
  reviewed_target_origin: "http://127.0.0.1:19548" | "http://127.0.0.1:18548"
  capture_import_only: true
native:
  options_known: true
  launch_mode: "run"
  config_source: "file"
  h2c: false
  strict_lexical_scope: explicit bool, compared with loaded app.yaml
  ui_scalar_mapping: EXACT complete Go Config scalar JSON after reviewed path rebasing
  api_scalar_mapping: EXACT complete api.Config scalar JSON
  queue_mapping: EXACT jobqueue.Config JSON; Workers must be 0
  actor_role_policies: EXACT stored role-name -> auth.Permission JSON
  platform_presentation: {version: ..., commit: ..., date: ..., author: ..., license: ...}
  components:
    scheduler: true
    scheduler_jobs_loaded_dormant: true
    queue: true
    bundle: true
    login_limit: true
    mailer: false
    webhooks: false
    debug_routes: false
    metrics: false
    workers_running: false
supervisor:
  proof: Pin to WORK supervisor proof (schema below)
  private_network: true
  net_deny: true
  jobs_stopped: true
  connections_zero: true
  owned_units: [4 distinct names with WORK basename prefix]
  owned_tunnel_count: 1
  backend_memory_max: explicit integer, 1..1073741824
  browser_memory_max: explicit integer, 1..1073741824
  runtime_max_seconds: explicit integer, 1..180
  browser_max_seconds: explicit integer, 1..60
  backend_port: 18548
  finally_cleanup_bound: true
```

SOURCE is exactly `/home/onebase/cc2b-audio-task-polish-20261114`.
Backend does not access owner Windows login/credential files; supervisor pins
cross-host login/capture provenance. No raw DSN, password, token or cookie in
this schema or any command line. Mode/supervisor/ownership proofs are below
WORK and independently reviewed; their booleans are attestations, not an OS
sandbox or historical witnesses. Unit memory/cgroup/firewall/timeout/browser
enforcement and finally are owner-supervisor responsibilities. Backend checks
its distinct network namespace, fixed loopback bind, file identities/caps,
and rejects inherited debug/control/master-key/DB/H2C/StrictRLS environments.

Private binding schema, no additional keys:
`schema="onebase.audio.stage2.bindings/v1"`, `own_synthetic=true`,
`actor_id=<own canonical nonzero UUID>`, `is_admin=false`,
`role_names=["ОТКСотрудник"]`, `entity="А_Звонок"`,
`record_id=<own canonical nonzero UUID>`, `form="Прослушивание"`,
`player="ПлеерЗаписи"`,
`form_os_relative="forms/а_звонок/прослушивание.form.os"`.
Backend validates actual middleware user identity/roles/admin, effective
permissions, selected form and exact named header_end player. UUIDs are only
private RAM comparisons; none are emitted in flags/errors/logs.

Mode proof schema (all fields required, no extras):
`schema="onebase.audio.stage2.mode/v1"`, `explicit_mode=<one of two>`,
`original_parity="UNKNOWN"`, `work_root=WORK`, `fresh_copy=true`,
`execution_approved=true`. Owner creates this only following a separate run
decision. It records a new controlled mode, not original parity.

Supervisor proof schema (all fields required, no extras):
`schema="onebase.audio.stage2.supervisor/v1"`, `work_root`, `explicit_mode`,
`ownership_sha256`, `login_source_sha256`, `coordinator_sha256`, `capture_sha256`,
`origin`, `native_mapping_reviewed=true`, `source_units_inactive=true`,
`private_network=true`, `net_deny=true`, `jobs_stopped=true`,
`finally_cleanup_bound=true`, `source_immutable=true`, `owned_units`,
`backend_memory_max`, `browser_memory_max`, `runtime_max_seconds`,
`browser_max_seconds`, `owned_tunnel_count`. Corresponding pins/scalars/lists
must exactly match the input. Missing facts STOP; no defaults or fabricated
witnesses. The future runner itself still needs a concrete diff review.

Frozen config manifest is read only for `files_sha256` (exact2131),
`config_archive_sha256=2a964f3f03ad36125f41a65e3ac2e02f0da8985591d48293196a1369722c4dee`
and `overlay_commit=ed54e6717300e29a60b876eabcb64867861c1710`.
Its other private members never leave RAM. Source and copy trees/hashes are
verified; source WAL/SHM must be absent. DB<=512MiB, config<=256MiB,
media<=64MiB, total pre-created WORK<=1GiB/2400 files; symlink/hardlink/device
inputs reject. Read-only checks do not open source SQLite. Copy/preparation
are outside this implementation and require separate ownership/window.

## Native mapping review

| Native location | Test composition / explicit limit |
| --- | --- |
| `cli/project_runtime.go:registryFromProject` | Full project.Load and every registry Load*: entity/program/manager/service/page/register/info/enum/constants/report/print/DSLprint/layout/module/processor/HTTP/page/exchange/intake/subsystem/journal/account/chart/widget/home/help. No minimized replacement form/module. |
| `cli/run_runtime.go:prepareServerRuntime` | Interpreter resolvers and lexical mode, UI app/platform/attachment/limit/feature/13presentation keys, locales bundle. Entire scalar JSON compared against reviewed witness including explicit zero/nil values. |
| Existing external metadata | Native extform LoadEnabled print/layout/report/processors from own copy, no EnsureSchema. Missing schema STOP. |
| Native role preparation | Load roles YAML and compare actual DB effective permissions, no SyncRoles/CreateUsers/ContextWithUser. Native login/session/RBAC/RLS/masking remain active. |
| `app/app.go:Build` | Same ui.New + api.New, shared repo/interpreter/store/scheduler, limiter5/min, BuildJobDSLVars, API config. Same public middleware/auth/CSRF/security/recovery; logging discarded before composition. No debugger route/debug token/metrics. |
| Scheduler / queue | Non-nil native constructors and exact queue mapping, Workers=0. **Review delta**: LoadJobs on NEW non-running scheduler only builds in-memory job/cron registry; source replaceJobs starts cron only if already running=false. No Start, RunNow, queue worker, reconciliation or scheduler tick. Omitting LoadJobs would silently change native DSL job lookup; reviewer must explicitly accept this dormant registration before Go/run. |
| Launch mode | Native run/file, Dev=false, H1 loopback18548. No nil scheduler substitute; unknown run/dev/options STOP. Controlled StrictRLS changes only guard initialization. |
| Native startup effects | No schemas/settings/AI/LLM apply, backup/demo/mail/webhook/file-storage setup, migrations, seeds, rolesync, key, egress or connection enable. Config requiring any such dependency rejects `unsupported_native_dependency`; it needs a separate concrete proposal, not silent nil/default emulation. |

No original-option witness is present in this code handoff. A future owner
must review and supply exact mappings for this frozen own shape. Rebased WORK
paths are explicit review inputs. Historical inherited StrictRLS remains
UNKNOWN even if other options can be proved. If current native shape needs a
disabled startup dependency or secret resolver/key, STOP before execution.

## Observation protocol

1. Wrap only H1 POST `/ui/document/А_Звонок/form-event`. Verify native H1
   Flusher/Hijacker/CloseNotifier/ReaderFrom interface set, no H2/Pusher. Wrapper
   preserves these, exposes standard Unwrap and forwards identical bytes,
   headers, status and errors. SSE/HTML/static/login/services are untouched.
2. Transparent bounded tee observes bytes **as native reads them**, does not
   pre-parse, replace/limit the native body, hide Read errors or construct a
   request. Only event/element metadata enter flags; native values stay in
   capped RAM. Native service metadata `_event/_element` is source-confirmed;
   no form-selected transport is fabricated.
3. Existing per-exec DebugSource only observes the FIRST statement of the
   exact loaded Cyrillic form module. No eval/breakpoint wait/DSL setter. It
   retains live formObjectThis and snapshots own header/TP on a read-only
   context derived from its live policy context, not a substitute context.
   Before-DSL snapshot is a proxy; absence or wrong actor/form is a gap.
4. At the FIRST native `application/json` body Write, before forwarding the
   same bytes, inspect the live policy/transaction context. Native guard and
   serialization have already decided. Read-only own-copy snapshot and
   baseline/original/projected reasons run there, not after cancellation.
   Record exact named header_end player, metadata eligibility/collisions,
   projection exclusion metadata names and classified reasons only.
5. Context/transaction closed/open, actor mismatch, DB snapshot changed,
   detected clone race, missing debugger entry, unknown reason or
   projection-vs-native-response mismatch => `OBSERVATION_GAP`, STOP. No
   WithoutCancel/Background policy fallback. Read-only snapshot contexts retain
   real user/policy; cleanup contexts are used only for HTTP/server shutdown.
   Copies are same-request scalar/TP clones; a repeated clone detects a
   changed view, not proof against arbitrary concurrent ABA writes.
6. After ServeHTTP: ONLY parse capped JSON reply booleans, flush sanitized
   flags, erase raw wire/form buffers/references. No live-object/reason/query.
   Record per-request deferred flag; if a Play cannot be observed in its actual
   response path, it is a gap, not an inferred native decision.
7. First decisive Play/gap seals only new observation. Backend continues
   serving native media/finally until the owner writes private `browser.done`
   after closing context **and** browser. No early shutdown manufacturing a
   capture failure and no new debug/status HTTP endpoint.

`browser.done` is owner0600 strict JSON: schema
`onebase.audio.stage2.browser-done/v1`, `explicit_mode`, `capture_sha256`,
`stage="CAPTURE_COMPLETED"`, `context_closed=true`, `browser_closed=true`.
Other stage/missing/late receipt -> gap. Future owner runner must write it
atomically in its own finally and still clean its4units+1tunnel after backend
`stop.request`. Those resources are not falsely declared closed by backend.

Output private files: `ready.json` (ready/mode/UNKNOWN/target-bound bool),
`flags.json` (schema below), `stop.request` (owned-supervisor-stop bool).
No other output path. At most16 records; body/reply RAM<=8MiB each per record;
flags file<=64KiB. Memory/runtime limits also require owner enforcement.

## Output allowlist v1

Top-level flags schema `onebase.audio.stage2.flags/v1`:
`status` (ACTUAL_RED_OBSERVED / NOT_REPRODUCED / OBSERVATION_GAP),
`explicit_mode`, `original_strict_rls_parity=UNKNOWN`, `records`,
`global_observation_gap`, `browser_cleanup_confirmed`, `pending_requests`,
`source_unchanged`, `sdk_changed=false`, `cause_claimed=false`,
`next=STOP_REVIEW_NO_AUTOFIX`. Status is not a release/readiness/PASS claim.

Record keys: sequence, metadata event/element, point
`first_native_json_write_pre_forward_proxy`, mode/UNKNOWN, selected metadata
form, player_count, named_header_player, excluded_metadata_names,
carrier_eligibility, baseline_proxy/original_proxy/projected_proxy,
observed_reply_dirty, deferred, ok, policy_context_live, transaction_settled,
own_db_snapshot_stable, native_writer_interfaces, native_ui_commands_header,
first_json_write_observed, observation_gap, reason_requested.
`own_db_snapshot_stable` compares only the owned header and its table parts
between entry and Write; it does not claim a whole-database snapshot stayed
unchanged across a legitimate audit save of a different object.

Carrier keys: metadata name/declared_type, scalar/save/main/entity/service
booleans, non_audio_collision_names. Reason keys: class, optional canonical
metadata field/type. Classes: clean, entity_field, table_part, form_attr,
form_table, no_record, record_read_error, record_missing, service_field,
unknown. Unknown raw reasons are discarded, never emitted. No values, UUID,
value digest/length, record number, caption/source, login, URL/query, SQL,
cookie, bearer, raw DB error or stack trace.

Observation gap codes are fixed source literals: request_observation_limit,
request_metadata_gap, target_metadata_gap, clone_race, actor_policy_parity,
selected_form_parity, policy_context_closed, transaction_context_closed,
transaction_not_settled, entry_snapshot_unavailable, entry_observation_missing,
live_scope_changed, named_header_player_parity, db_snapshot_changed,
reason_snapshot_unavailable, reason_class_unknown, observer_panic,
not_native_json, native_writer_interface_gap, event_limit, reply_capture_gap,
reply_semantic_gap, projection_response_gap, ui_header_parity,
browser_receipt_unavailable, browser_cleanup_or_capture_gap, runtime_bound,
backend_ended, http_cleanup_gap. Test setup emits only closed
`stage2_stop:<source literal>` codes, never native error text. All native slog
and stdlib log output is discarded before project/API composition.

## Controlled StrictRLS delta-plan (not permission)

Historical launcher inherited environment is UNKNOWN. Never inspect current
manager/raw environment or reconstruct false from a missing saved witness.
Use two **fresh independent** owner-copies of the same attested source DB,
2131-file config and cached own media. New WORK/proofs/bindings/units/cleanup
per mode; false copy is not reused as true copy after audit writes.

- `strict_false`: explicitly absent inherited ONEBASE_STRICT_RLS, do not
  initialize storage strict guard. Label diagnostic controlled false.
- `strict_true`: likewise reject inherited setting; explicitly initialize
  guard using native `access.GuardedEntitiesFromRoles(loadedRoles)` and
  `db.SetStrictRLSGuard`. Label diagnostic controlled true.

Both retain native auth/session/policies/roles and no ACL expansion. The env
string `0` is **not** false in native; this experiment does not use it. Each
mode requires reviewed provenance and exact options, owner copy approval,
then separate Go END and browser/runner window. A differing result narrows a
controlled factor; it does not establish historical mode or cause. Gap or
nonreproduction => facts and STOP for review, no synthetic retry without a new
actual difference, no SDK/autofix/actual AFTER/shared8444/prod change.

Generated-with: Codex
