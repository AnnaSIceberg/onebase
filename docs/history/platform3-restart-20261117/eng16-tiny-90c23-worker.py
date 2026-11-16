"""Prepared source only. Coordinator calls under its real exclusive held FD."""
from pathlib import Path
import datetime
import hashlib
import json
import os
import shutil
import subprocess
import time

BASE = Path(__file__).parent.resolve()
HEAD = "90c23e981b138c2c7c5d5848c3c3db398920a547"
CWD = Path("C:/Users/a.sevrjukova/AppData/Local/Temp/platform3-report-lineage-source-20261116")
GO = Path("C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe")
GO_SHA = "c0ea949d000a24883f37b1dd2db794a9fb2b824fbcff11facb469615a50ca6af"
PLAN = Path("C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis/20261117-platform3-ENG16-tiny-90c23-plan.json")
PLAN_SHA = "596f704844003132ade10f2ad968b941eb384295166a009bf3a1d9c6af798e20"
LOCK = Path("C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/perenos-codex/тяжёлый-прогон-2.lock")
COMMANDS = [
    ("query-info-slice", ["test", "-p=1", "./internal/query", "-run", "^TestReportLineageInfoSlice", "-count=1", "-timeout=35s", "-json"]),
    ("access-virtual", ["test", "-p=1", "./internal/access", "-run", "^TestReportOutputVirtual", "-count=1", "-timeout=35s", "-json"]),
]


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def summarize(raw, package):
    events = []
    for line in raw.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except (ValueError, TypeError):
            continue
        if event.get("Package") == package:
            events.append(event)
    runs = {event["Test"] for event in events if event.get("Action") == "run" and event.get("Test")}
    if not runs:
        raise RuntimeError("NO_SELECTED_TESTS")
    leaves = {name for name in runs if not any(other.startswith(name + "/") for other in runs)}
    outcomes = {}
    package_result = None
    for event in events:
        action = event.get("Action")
        test = event.get("Test")
        if action in ("pass", "fail", "skip"):
            if test in leaves:
                outcomes[test] = action
            elif not test:
                package_result = action
    if set(outcomes) != leaves:
        raise RuntimeError("INCOMPLETE_TEST_JSON")
    return {"leaf_pass": sum(value == "pass" for value in outcomes.values()),
            "leaf_fail": sum(value == "fail" for value in outcomes.values()),
            "leaf_skip": sum(value == "skip" for value in outcomes.values()),
            "package_result": package_result}


def run_in_slot2(window_id, assert_slot2_held, output_dir):
    if os.name != "nt" or not callable(assert_slot2_held):
        raise RuntimeError("WINDOWS_REAL_HELD_FD_CALLBACK_REQUIRED")
    if not isinstance(window_id, str) or not 0 < len(window_id) <= 128:
        raise RuntimeError("INVALID_WINDOW_ID")
    assert_slot2_held()
    # Compatible with the coordinator's selftested SlotLease closure. A boolean,
    # owner string, fake lock path or no-op lambda is not a held-FD callback.
    cells = getattr(assert_slot2_held, "__closure__", None) or ()
    leases = [cell.cell_contents for cell in cells
              if all(hasattr(cell.cell_contents, key) for key in ("stream", "identity", "claim", "assert_held"))]
    if len(leases) != 1:
        raise RuntimeError("REAL_SLOTLEASE_CLOSURE_REQUIRED")
    lease = leases[0]
    if os.path.normcase(os.path.abspath(lease.path)) != os.path.normcase(str(LOCK)):
        raise RuntimeError("NOT_THE_SHARED_SLOT2_LOCK")
    stream = lease.stream
    fd = stream.fileno()
    identity = os.fstat(fd).st_ino
    if lease.stream.closed or identity != lease.identity or os.get_inheritable(fd):
        raise RuntimeError("INVALID_HELD_FD")
    claim = lease.claim
    if claim.get("owner") != "platform3" or claim.get("head") != HEAD or claim.get("window_id") != window_id or claim.get("coordinator") != "perenos" or claim.get("pid") != os.getpid() or not claim.get("token"):
        raise RuntimeError("HELD_FD_CLAIM_MISMATCH")
    if sha(PLAN) != PLAN_SHA or sha(GO) != GO_SHA:
        raise RuntimeError("PLAN_OR_TOOLCHAIN_PIN_DRIFT")
    authority = json.loads(PLAN.read_text(encoding="utf-8"))
    if authority["candidate"] != HEAD or authority["owner"] != "platform3" or authority["source_root"] != CWD.as_posix():
        raise RuntimeError("AUTHORITY_IDENTITY_DRIFT")
    if [(item["name"], item["argv"]) for item in authority["commands"]] != COMMANDS:
        raise RuntimeError("COMMAND_PIN_DRIFT")
    dest = Path(output_dir).resolve()
    if dest.parent != BASE or not dest.name.startswith("eng16-tiny-90c23-") or dest.exists():
        raise RuntimeError("OUTPUT_ROOT_MUST_BE_NEW_OWN_LEAF")
    started = time.monotonic()
    deadline = started + 120
    checks = 0

    def held():
        nonlocal checks
        assert_slot2_held()
        lease.assert_held()
        if lease.stream is not stream or stream.closed or stream.fileno() != fd or os.fstat(fd).st_ino != identity:
            raise RuntimeError("HELD_FD_CHANGED")
        if time.monotonic() >= deadline:
            raise TimeoutError("WORKER_WALL_TIMEOUT_120s")
        if sha(PLAN) != PLAN_SHA:
            raise RuntimeError("PLAN_PIN_DRIFT")
        checks += 1

    def verify_source():
        held()
        for args, expected in ((["rev-parse", "HEAD"], HEAD), (["status", "--porcelain=v1", "--untracked-files=all"], "")):
            result = subprocess.run(["git", *args], cwd=CWD, capture_output=True, timeout=min(5, max(.1, deadline - time.monotonic())), creationflags=subprocess.CREATE_NO_WINDOW)
            if result.returncode != 0 or result.stdout.decode("utf-8", "replace").strip() != expected:
                raise RuntimeError("SOURCE_HEAD_OR_CLEAN_DRIFT")
        held()

    verify_source()
    if shutil.disk_usage("C:/").free < 2147483648:
        raise RuntimeError("C_DISK_BELOW_2_GiB_BEFORE_OUTPUT")
    dest.mkdir()
    (dest / "go-tmp").mkdir()
    (dest / "tmp").mkdir()
    env = os.environ.copy()
    env.update(authority["environment"])
    env.update(authority["no_pg_env"])
    env.update(GOTMPDIR=str(dest / "go-tmp"), TMP=str(dest / "tmp"), TEMP=str(dest / "tmp"))
    receipt_path = dest / "receipt.json"
    receipt = {"window_id": window_id, "owner": "platform3", "slot": 2, "head": HEAD,
               "plan_sha256": PLAN_SHA, "worker_sha256": sha(__file__), "go_sha256": GO_SHA,
               "started": now(), "status": "RUNNING", "phases": [],
               "synthetic_sqlite_only": True, "postgres": False, "actual_db": False,
               "sdk": False, "http": False, "browser": False, "new_wt": False,
               "archives": False, "whole_suites": False, "auto_rel38": False,
               "real_held_fd": {"identity": identity, "noninheritable": True},
               "guard_release_owner": "coordinator only; worker never acquires/releases shared lock"}

    def save():
        receipt["held_fd_checks"] = checks
        receipt_path.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    process = None
    save()
    try:
        for name, args in COMMANDS:
            held()
            verify_source()
            free = shutil.disk_usage("C:/").free
            if free < 2147483648:
                raise RuntimeError("C_DISK_BELOW_2_GiB_BEFORE_" + name)
            if deadline - time.monotonic() < 10:
                raise TimeoutError("NO_PHASE_BUDGET")
            if sha(GO) != GO_SHA:
                raise RuntimeError("GO_TOOLCHAIN_PIN_DRIFT")
            raw = dest / (name + ".jsonl")
            phase = {"name": name, "argv": [str(GO), *args], "started": now(),
                     "free_before": free, "status": "RUNNING", "raw": str(raw)}
            receipt["phases"].append(phase)
            save()
            try:
                with raw.open("xb") as log:
                    process = subprocess.Popen([str(GO), *args], cwd=CWD, env=env, stdout=log, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW)
                    phase["own_pid"] = process.pid
                    save()
                    phase_started = time.monotonic()
                    while process.poll() is None:
                        held()
                        if shutil.disk_usage("C:/").free < 1073741824:
                            raise RuntimeError("C_DISK_BELOW_1_GiB_DURING_" + name)
                        if time.monotonic() - phase_started > 50:
                            raise TimeoutError("OWN_PHASE_TIMEOUT_50s_" + name)
                        time.sleep(.25)
                phase["exit_code"] = process.returncode
                held()
                verify_source()
                if process.returncode:
                    phase["status"] = receipt["status"] = "FAIL"
                    break
                phase["tests"] = summarize(raw, "github.com/ivantit66/onebase/" + args[2].removeprefix("./"))
                if phase["tests"]["package_result"] != "pass" or phase["tests"]["leaf_fail"]:
                    raise RuntimeError("TEST_JSON_NOT_PASS_" + name)
                phase["status"] = "PASS"
            finally:
                phase["ended"] = now()
                phase["free_after"] = shutil.disk_usage("C:/").free
                if raw.exists():
                    phase["raw_sha256"] = sha(raw)
                save()
        else:
            receipt["status"] = "PASS"
    except BaseException as exc:
        if process is not None and process.poll() is None:
            subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"], capture_output=True, timeout=10, creationflags=subprocess.CREATE_NO_WINDOW)
            process.wait(timeout=5)
        receipt["status"] = "STOP"
        receipt["reason"] = str(exc)
        if receipt["phases"] and receipt["phases"][-1]["status"] == "RUNNING":
            receipt["phases"][-1]["status"] = "STOP"
        raise
    finally:
        receipt["ended"] = now()
        receipt["elapsed_seconds"] = round(time.monotonic() - started, 3)
        receipt["free_after"] = shutil.disk_usage("C:/").free
        save()
    return receipt


if __name__ == "__main__":
    raise SystemExit("NO_SELF_START: coordinator imports run_in_slot2 only under NEW exact assignment and a real held FD")
