#!/usr/bin/env python3
"""Run the full race/atomic suite, bounding large package process budgets."""

from collections import Counter
import json
import os
import signal
import threading
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time

# Each process keeps the original 10m timeout. The build job has a 40m budget.
BATCH_SIZE = 128
BATCHED_PACKAGES = ("internal/auth", "internal/launcher", "internal/ui")
ISOLATED_PACKAGE = "internal/dsl/interpreter"
OTHER_CONCURRENCY = 4


def discover_names(output):
    names = []
    for line in output.splitlines():
        if line.startswith(("Test", "Example", "Fuzz")):
            if not re.fullmatch(r"(?:Test|Example|Fuzz)\w*", line):
                raise ValueError(f"invalid discovered name: {line!r}")
            names.append(line)
    if not names or len(set(names)) != len(names):
        raise ValueError("empty or duplicate test inventory")
    return names


def verify_inventory(expected, actual):
    if not expected or len(set(expected)) != len(expected) or Counter(expected) != Counter(actual):
        raise ValueError(f"execution differs from discovery: expected={Counter(expected)}, actual={Counter(actual)}")


def merge_profiles(profiles, destination):
    counts, statements = {}, {}
    if not profiles:
        raise ValueError("no coverage profiles")
    for profile in profiles:
        blocks = set()
        with profile.open(encoding="utf-8") as stream:
            if stream.readline() != "mode: atomic\n":
                raise ValueError(f"not atomic coverage: {profile}")
            for line in stream:
                location, size, count = line.split()
                size, count = int(size), int(count)
                if (not re.fullmatch(r".+:\d+\.\d+,\d+\.\d+", location)
                        or location in blocks or size < 0 or count < 0
                        or (location in statements and statements[location] != size)):
                    raise ValueError(f"invalid coverage block: {line!r}")
                blocks.add(location)
                statements[location] = size
                counts[location] = counts.get(location, 0) + count
        if not blocks:
            raise ValueError(f"empty coverage: {profile}")
    temporary = destination.with_name(destination.name + ".tmp")
    with temporary.open("w", encoding="utf-8", newline="\n") as stream:
        stream.write("mode: atomic\n")
        for location in sorted(counts):
            stream.write(f"{location} {statements[location]} {counts[location]}\n")
    temporary.replace(destination)


def discover_package(package, binary, report):
    # Compile exactly once; discovery and every group execute this same binary.
    binary = binary.resolve()
    cwd = subprocess.check_output(["go", "list", "-f", "{{.Dir}}", package],
                                  text=True, encoding="utf-8").strip()
    if not Path(cwd).is_absolute() or not Path(cwd).is_dir():
        raise ValueError(f"invalid package directory: {cwd!r}")
    env = dict(os.environ, PWD=cwd)
    args = ["go", "test", "-c", "-race", "-covermode=atomic", "-o", str(binary), package]
    list_args = [str(binary), "-test.list=."]
    started = time.monotonic()
    entry = {"kind": "discovery", "args": args, "list_args": list_args,
             "package": package, "cwd": cwd}
    report["processes"].append(entry)
    try:
        subprocess.run(args, check=True)
        entry["compile_elapsed"] = time.monotonic() - started
        listing = time.monotonic()
        output = subprocess.check_output(list_args, cwd=cwd, env=env, text=True, encoding="utf-8")
        entry["list_elapsed"] = time.monotonic() - listing
        names = discover_names(output)
        entry["names"] = names
        entry["success"] = True
        return names, binary, cwd
    finally:
        entry["elapsed"] = time.monotonic() - started


def kill_process_tree(proc):
    if proc.poll() is None:
        try:
            if os.name == "posix":
                os.killpg(proc.pid, signal.SIGKILL)
            else:
                proc.kill()
        except ProcessLookupError:
            pass


def run_tests(packages, profile, names=None, *, report=None, log=None, concurrency=None,
              binary=None, cwd=None, timeout="10m"):
    profile = profile.resolve()
    args = ["go", "test", "-json", "-count=1", "-race", "-timeout=10m",
            "-covermode=atomic", f"-coverprofile={profile}"]
    if concurrency is not None:
        args += ["-p", str(concurrency)]
    if names is not None:
        args += ["-run", "^(" + "|".join(re.escape(name) for name in names) + ")$"]
    args += packages
    env = None
    if binary is not None:
        if len(packages) != 1 or names is None or cwd is None or concurrency is not None:
            raise ValueError("a compiled group requires one package, inventory and cwd")
        coverdir = profile.with_suffix(".gocoverdir")
        coverdir.mkdir()
        env = dict(os.environ, PWD=str(cwd), GOCOVERDIR=str(coverdir))
        args = ["go", "tool", "test2json", "-p", packages[0], "-t", str(binary),
                "-test.v=test2json", "-test.paniconexit0", "-test.count=1",
                "-test.timeout=" + timeout, "-test.coverprofile=" + str(profile),
                "-test.gocoverdir=" + str(coverdir),
                "-test.run=^(" + "|".join(re.escape(name) for name in names) + ")$"]
    print("Running:", *args, flush=True)
    runs, completed, outcomes = [], [], []
    entry = {"kind": "tests", "args": args, "packages": packages, "names": names, "tests": outcomes, "cwd": str(cwd) if cwd else None}
    if report is not None:
        report["processes"].append(entry)
    started = time.monotonic()
    watchdog = None
    timed_out = threading.Event()
    stream = log.open("w", encoding="utf-8") if log else None
    try:
        with subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                              cwd=cwd, env=env, start_new_session=(os.name == "posix"),
                              text=True, encoding="utf-8") as proc:
            if binary is not None:
                # Match cmd/go's last-ditch deadline for a wedged TestMain or
                # inherited output pipe. The test's own limit remains 10m.
                match = re.fullmatch(r"([0-9]+(?:\.[0-9]+)?)(ms|s|m)", timeout)
                if not match:
                    kill_process_tree(proc)
                    raise ValueError("unsupported test timeout")
                seconds = float(match[1]) * {"ms": .001, "s": 1, "m": 60}[match[2]]
                def expire():
                    timed_out.set()
                    kill_process_tree(proc)
                watchdog = threading.Timer(seconds + 60, expire)
                watchdog.daemon = True
                watchdog.start()
            try:
                for line in proc.stdout:
                    if stream:
                        stream.write(line)
                    print(line, end="", flush=True)
                    event = json.loads(line)
                    if binary is not None and event.get("Package") != packages[0]:
                        raise ValueError("wrong package in test event")
                    name = event.get("Test", "")
                    if name and "/" not in name:
                        if event.get("Action") == "run":
                            runs.append(name)
                        if event.get("Action") in {"pass", "skip", "fail"}:
                            completed.append(name)
                            outcomes.append({"package": event.get("Package"), "name": name,
                                             "action": event["Action"], "elapsed": event.get("Elapsed", 0)})
            except BaseException:
                kill_process_tree(proc)
                proc.wait()
                raise
            entry["exit"] = proc.wait()
            if timed_out.is_set():
                raise TimeoutError("compiled test process exceeded its hard deadline")
            if proc.returncode:
                raise subprocess.CalledProcessError(proc.returncode, args)
        if names is not None:
            verify_inventory(names, runs)
            verify_inventory(names, completed)
        entry["success"] = True
        return runs
    finally:
        if watchdog:
            watchdog.cancel()
        entry["elapsed"] = time.monotonic() - started
        if stream:
            stream.close()


def main(destination=None, report_directory=None):
    destination = Path(destination or "coverage.out")
    directory = Path(report_directory or str(destination) + ".report")
    directory.mkdir(parents=True, exist_ok=True)
    destination.unlink(missing_ok=True)  # Never retain an older successful run.
    report = {"success": False, "processes": [], "inventories": {}, "batch_size": BATCH_SIZE,
              "other_concurrency": OTHER_CONCURRENCY}
    started = time.monotonic()
    try:
        packages = subprocess.check_output(["go", "list", "./..."], text=True, encoding="utf-8").splitlines()
        if not packages or len(packages) != len(set(packages)):
            raise ValueError("empty or duplicate package inventory")
        report["packages"] = packages
        selected = {}
        for suffix in (*BATCHED_PACKAGES, ISOLATED_PACKAGE):
            matches = [package for package in packages if package.endswith("/" + suffix)]
            if len(matches) != 1:
                raise ValueError(f"package inventory must contain exactly one {suffix}")
            selected[suffix] = matches[0]
        other = [package for package in packages if package not in selected.values()]
        if not other:
            raise ValueError("empty remaining package inventory")
        with tempfile.TemporaryDirectory(prefix="onebase-race-") as temporary:
            temporary = Path(temporary)
            compiled = {}
            for index, suffix in enumerate(BATCHED_PACKAGES):
                package = selected[suffix]
                names, binary, cwd = discover_package(package, temporary / f"tests-{index}", report)
                report["inventories"][package] = names
                compiled[package] = (binary, cwd)
            profiles = []
            # Interpreter runs alone: the XML budget checks must not compete with
            # another test process launched by this runner. Do not relax guards.
            for label, group, concurrency in [("interpreter", [selected[ISOLATED_PACKAGE]], None),
                                               ("other", other, OTHER_CONCURRENCY)]:
                profile = temporary / f"{label}.out"
                run_tests(group, profile, report=report, log=directory / f"{label}.jsonl", concurrency=concurrency)
                profiles.append(profile)
            for suffix in BATCHED_PACKAGES:
                package = selected[suffix]
                names, runs = report["inventories"][package], []
                binary, cwd = compiled[package]
                for start in range(0, len(names), BATCH_SIZE):
                    label = suffix.rsplit("/", 1)[-1] + f"-{start // BATCH_SIZE + 1}"
                    profile = temporary / f"{label}.out"
                    runs.extend(run_tests([package], profile, names[start:start + BATCH_SIZE],
                                          report=report, log=directory / f"{label}.jsonl", binary=binary, cwd=cwd))
                    profiles.append(profile)
                verify_inventory(names, runs)
            merge_profiles(profiles, destination)
        report["success"] = True
        print(f"Full package/test inventories verified; coverage: {destination}", flush=True)
    except BaseException as error:
        report["error"] = str(error)
        raise
    finally:
        report["elapsed"] = time.monotonic() - started
        (directory / "report.json").write_text(json.dumps(report, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main(*sys.argv[1:])
