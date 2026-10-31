"""UI4/launcher2/CLI4 acceptance; negative controls and real Go test discovery."""

from collections import Counter
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest


RUNNER = Path(__file__).with_name("test-coverage.py")
UI = "github.com/ivantit66/onebase/internal/ui"
LAUNCHER = "github.com/ivantit66/onebase/internal/launcher"
CLI = "github.com/ivantit66/onebase/internal/cli"

# This is a child process replacing only Go at the external command boundary.
# The real Go fixture below checks the actual -run/subtest/seed semantics.
FAKE_GO = r'''#!/usr/bin/env python3
import json, os, pathlib, re, sys
args = sys.argv[1:]
mode = os.environ.get('SCENARIO', '')
print('go: downloading fixture dependency', file=sys.stderr)
ui = 'github.com/ivantit66/onebase/internal/ui'
launcher = 'github.com/ivantit66/onebase/internal/launcher'
cli = 'github.com/ivantit66/onebase/internal/cli'
other = 'github.com/ivantit66/onebase/other'
with open('calls.jsonl', 'a') as f: f.write(json.dumps(args)+'\n')
if args == ['list', './...']:
    if mode == 'list-failure': sys.exit(2)
    print(other)
    print(ui if mode != 'missing-ui-package' else 'wrong/ui')
    print(launcher if mode != 'missing-launcher-package' else 'wrong/launcher')
    print(cli if mode != 'missing-cli-package' else 'wrong/cli')
    if mode == 'duplicate-cli-package': print(cli)
    if mode == 'duplicate-package': print(ui)
    sys.exit(0)
assert args[0] == 'test'
assert all(flag in args for flag in ['-race', '-count=1', '-timeout=10m', '-json'])
def emit(action, package, **kw):
    print(json.dumps(dict(Action=action, Package=package, **kw)))
names = ['TestAlpha', 'TestBravo', 'TestCharlie', 'TestDelta', 'ExampleValue', 'FuzzSeed']
if os.environ.get('NEW_TEST'): names += ['TestNewAutomaticallyDiscovered']
package = args[-1]
if mode.startswith('cli-'):
    mode = mode[4:] if package == cli else ''
if package == cli: names += ['TestCLIOnly']
if '-list' in args:
    assert package in [ui, launcher, cli] and args[args.index('-list')+1] == '^(Test|Example|Fuzz)'
    if mode == 'empty-launcher-inventory' and package == launcher: names=[]
    if mode == 'compile-failure': sys.exit(1)
    if mode == 'empty-inventory': names=[]
    if mode == 'duplicate-inventory': names += [names[0]]
    if mode == 'invalid-inventory': names += ['not a test']
    for n in names: emit('output', package, Output=n+'\n')
    emit('output', package, Output='ok\t'+package+'\t0.01s\n')
    emit('pass', package, Elapsed=0.01)
    sys.exit(0)
assert '-covermode=atomic' in args
coverage = pathlib.Path(next(a.split('=',1)[1] for a in args if a.startswith('-coverprofile=')))
package = args[-1]
assert ("-p=2" in args) == (package == other)
if package in [ui, launcher, cli]:
    pattern=args[args.index('-run')+1]
    assert pattern.startswith('^(?:') and pattern.endswith(')$') and '/' not in pattern
    selected = [n for n in names if re.fullmatch(pattern, n)]
    if mode == 'missing-execution': selected = selected[1:]
    if mode == 'duplicate-execution': selected += [selected[0]]
    for n in selected:
        emit('run', package, Test=n)
        for sub in ['one', 'two']:
            emit('run', package, Test=n+'/'+sub)
            if mode != 'missing-subtest-result': emit('pass', package, Test=n+'/'+sub, Elapsed=0.001)
        if mode != 'missing-test-result': emit('pass', package, Test=n, Elapsed=0.002)
if mode in ['failed-test', 'race', 'timeout']:
    emit('output', package, Output={'failed-test':'FAIL test', 'race':'WARNING: DATA RACE',
                                  'timeout':'panic: test timed out after 10m0s'}[mode])
    emit('fail', package)
    sys.exit(1)
if mode != 'missing-package-result': emit('pass', package, Elapsed=0.01)
if mode == 'event-failure': emit('fail', package)
if mode == 'missing-profile': sys.exit(0)
if mode == 'corrupt-profile': coverage.write_text('garbage\n'); sys.exit(0)
count = int(coverage.stem[-1]) if package != other else 7
statements = 2
file = package+'/sample.go'
if package == ui and coverage.stem == 'ui-4':
    if mode == 'incompatible-profile': statements=3
    if mode == 'missing-block': file=ui+'/different.go'
if package == launcher and coverage.stem == 'launcher-2':
    if mode == 'incompatible-launcher-profile': statements=3
    if mode == 'missing-launcher-block': file=launcher+'/different.go'
if package == cli and coverage.stem == 'cli-4':
    if mode == 'incompatible-profile': statements=3
    if mode == 'missing-block': file=cli+'/different.go'
record = file+':1.1,2.2 '+str(statements)+' '+str(count)+'\n'
content = 'mode: atomic\n'+record
if package == cli: content += cli+'/sample.go:3.1,4.2 1 0\n'
if mode == 'wrong-mode': content = content.replace('atomic', 'set')
if mode == 'duplicate-block': content += record
if mode == 'negative-counter': content = content.replace(' 7\n',' -1\n')
if mode == 'empty-profile': content='mode: atomic\n'
coverage.write_text(content)
'''


class CoverageCLI(unittest.TestCase):
    runner = RUNNER

    def cli_command(self, runner):
        return [sys.executable, str(runner)]

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        go = self.bin / "go"
        go.write_text(FAKE_GO, encoding="utf-8")
        go.chmod(0o755)
        self.attempts = 0
        self.saved = None

    def tearDown(self):
        try:
            result = self._outcome.result
            failures = result.failures + result.errors
            if any(test.id().startswith(self.id()) for test, _ in failures):
                self.preserve_failure()
        finally:
            self.tmp.cleanup()

    def preserve_failure(self):
        """Copy only this generated fixture, never the inherited environment."""
        if self.saved is None:
            parent = os.environ.get("ONEBASE_ACCEPTANCE_ARTIFACTS")
            if parent:
                parent = Path(parent)
                if not parent.is_absolute():
                    raise ValueError("ONEBASE_ACCEPTANCE_ARTIFACTS must be absolute")
                parent.mkdir(parents=True, exist_ok=True)
            self.saved = Path(tempfile.mkdtemp(prefix="coverage-failure-", dir=parent))
            self.saved.chmod(0o700)
            versions = {"python": sys.version}
            for executable, argument in [("go", "version"), ("node", "--version")]:
                try:
                    version = subprocess.run([executable, argument], capture_output=True,
                                             timeout=10, check=False)
                    versions[executable] = {"exit": version.returncode,
                                           "stdout": version.stdout.decode("utf-8", errors="replace"),
                                           "stderr": version.stderr.decode("utf-8", errors="replace")}
                except (OSError, subprocess.TimeoutExpired) as error:
                    versions[executable] = {"error": str(error)}
            (self.saved / "versions.json").write_text(json.dumps(versions, indent=2), encoding="utf-8")
        shutil.copytree(self.root, self.saved / "fixture", dirs_exist_ok=True)
        print(f"Acceptance failure evidence: {self.saved}", file=sys.stderr, flush=True)

    def command(self, args, timeout, env=None):
        self.attempts += 1
        prefix = self.root / f"acceptance-{self.attempts}"
        started = time.monotonic()
        record = {"argv": args, "cwd": str(self.root), "timeout": timeout,
                  "started_at": time.time(), "python": sys.version,
                  "state": "running"}
        metadata = prefix.with_suffix(".json")
        metadata.write_text(json.dumps(record, indent=2), encoding="utf-8")
        options = ({"start_new_session": True} if os.name != "nt" else
                   {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP})
        proc = subprocess.Popen(args, cwd=self.root, env=env, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, **options)
        record["pid"] = proc.pid
        metadata.write_text(json.dumps(record, indent=2), encoding="utf-8")
        try:
            stdout, stderr = proc.communicate(timeout=timeout)
        except subprocess.TimeoutExpired as error:
            # communicate's partial output is bytes even for text-mode callers.
            prefix.with_suffix(".stdout").write_bytes(error.stdout or b"")
            prefix.with_suffix(".stderr").write_bytes(error.stderr or b"")
            record.update(state="timeout", elapsed=time.monotonic() - started)
            metadata.write_text(json.dumps(record, indent=2), encoding="utf-8")
            try:
                self.preserve_failure()
            finally:
                self.stop_fixture(proc)
                record.update(exit=proc.returncode, reaped=True)
                metadata.write_text(json.dumps(record, indent=2), encoding="utf-8")
                self.preserve_failure()
            raise
        else:
            prefix.with_suffix(".stdout").write_bytes(stdout)
            prefix.with_suffix(".stderr").write_bytes(stderr)
            record.update(state="finished", exit=proc.returncode,
                          elapsed=time.monotonic() - started)
            metadata.write_text(json.dumps(record, indent=2), encoding="utf-8")
            if proc.returncode:
                try:
                    self.preserve_failure()
                finally:
                    self.stop_fixture(proc)
            return subprocess.CompletedProcess(args, proc.returncode,
                                               stdout.decode("utf-8"), stderr.decode("utf-8"))

    @staticmethod
    def stop_fixture(proc):
        if os.name == "nt":
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"],
                           capture_output=True, check=False)
        else:
            # Only the new session created for this fixture is addressed.
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        proc.communicate()

    def run_cli(self, scenario="", runner=None, new_test=False):
        env = dict(os.environ, SCENARIO=scenario, NEW_TEST="1" if new_test else "")
        env["PATH"] = str(self.bin) + os.pathsep + env["PATH"]
        result = self.command(self.cli_command(runner or self.runner), timeout=60, env=env)
        return result

    def test_full_inventory_and_exact_atomic_sum(self):
        result = self.run_cli(new_test=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        inventory = json.loads((self.root / "test-results/inventory.json").read_text())
        for label, count in [("ui", 4), ("launcher", 2), ("cli", 4)]:
            item = inventory["series"][label]
            names = item["names"]
            self.assertIn("ExampleValue", names)
            self.assertIn("FuzzSeed", names)
            self.assertIn("TestNewAutomaticallyDiscovered", names)
            self.assertEqual(item["groups"], [sorted(names)[i::count] for i in range(count)])
            self.assertEqual(Counter(n for g in item["groups"] for n in g), Counter(names))
            self.assertEqual(len(list((self.root / "test-results").glob(label+"-?.jsonl"))), count)
        lines = (self.root / "coverage.out").read_text().splitlines()
        self.assertEqual(lines, ["mode: atomic",
                                CLI+"/sample.go:1.1,2.2 2 10",
                                CLI+"/sample.go:3.1,4.2 1 0",
                                LAUNCHER+"/sample.go:1.1,2.2 2 3",
                                UI+"/sample.go:1.1,2.2 2 10",
                                "github.com/ivantit66/onebase/other/sample.go:1.1,2.2 2 7"])
        calls = [json.loads(s) for s in (self.root / "calls.jsonl").read_text().splitlines()]
        self.assertEqual(len(calls), 15)  # list, three inventories, other, UI4, launcher2, CLI4
        self.assertNotIn(UI, calls[4])
        self.assertNotIn(LAUNCHER, calls[4])
        self.assertNotIn(CLI, calls[4])
        self.assertIn("-p=2", calls[4])
        self.assertNotIn("-run", calls[4])
        self.assertEqual(calls[4][-1:], ["github.com/ivantit66/onebase/other"])
        self.assertTrue(all("-p=2" not in call for call in calls[5:]))
        self.assertEqual([c[-1] for c in calls[5:]], [UI]*4+[LAUNCHER]*2+[CLI]*4)

    def test_failures_never_publish_partial_or_stale_coverage(self):
        for mode in ["list-failure", "missing-ui-package", "missing-launcher-package",
                     "missing-cli-package", "duplicate-cli-package",
                     "duplicate-package", "compile-failure", "empty-launcher-inventory",
                     "empty-inventory", "duplicate-inventory", "invalid-inventory",
                     "missing-execution", "duplicate-execution", "missing-test-result", "missing-subtest-result", "failed-test", "race", "timeout",
                     "missing-package-result", "event-failure", "missing-profile", "corrupt-profile",
                     "incompatible-profile", "missing-block", "incompatible-launcher-profile",
                     "missing-launcher-block", "wrong-mode", "duplicate-block",
                     "negative-counter", "empty-profile",
                     *["cli-"+case for case in [
                         "empty-inventory", "duplicate-inventory", "invalid-inventory",
                         "compile-failure", "missing-execution", "duplicate-execution",
                         "missing-test-result", "missing-subtest-result", "missing-package-result",
                         "failed-test", "race", "timeout", "event-failure", "missing-profile",
                         "corrupt-profile", "wrong-mode", "empty-profile", "duplicate-block",
                         "incompatible-profile", "missing-block"]]]:
            with self.subTest(mode=mode):
                shutil.rmtree(self.root / "test-results", ignore_errors=True)
                (self.root / "coverage.out").write_text("stale success\n")
                result = self.run_cli(mode)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertFalse((self.root / "coverage.out").exists())

    def test_partition_omission_and_duplicate_fail_before_tests(self):
        for label, count in [("ui", 4), ("launcher", 2), ("cli", 4)]:
            for expression in ["[names[:-1][i::count] for i in range(count)]",
                               "[names for i in range(count)]"]:
                with self.subTest(label=label, count=count, expression=expression):
                    shutil.rmtree(self.root / "test-results", ignore_errors=True)
                    source = RUNNER.read_text(encoding="utf-8")
                    source = source.replace("groups = [names[i::count] for i in range(count)]",
                                            f"groups = ({expression} if "
                                            + ('"TestCLIOnly" in names' if label == "cli" else f"count == {count}")
                                            + " else "
                                            "[names[i::count] for i in range(count)])")
                    modified = self.root / "damaged-runner.py"
                    modified.write_text(source, encoding="utf-8")
                    result = self.run_cli(runner=modified)
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                    self.assertIn("partition loses or repeats", result.stderr)
                    self.assertFalse((self.root / "coverage.out").exists())

    def test_real_go_examples_fuzz_subtests_and_dynamic_discovery(self):
        (self.root / "go.mod").write_text("module github.com/ivantit66/onebase\n\ngo 1.26.0\n")
        ui = self.root / "internal/ui"
        ui.mkdir(parents=True)
        (ui / "value.go").write_text("package ui\nfunc Value() int { return 42 }\n")
        (ui / "value_test.go").write_text('''package ui
import ("fmt"; "testing")
func TestAlpha(t *testing.T) { for _, n := range []string{"a", "b"} {
 t.Run(n, func(t *testing.T) { if Value()!=42 { t.Fatal("value") } }) } }
func TestBravo(t *testing.T) { if Value()!=42 { t.Fatal("value") } }
func TestCharlie(t *testing.T) { if Value()!=42 { t.Fatal("value") } }
func ExampleValue() { fmt.Println(Value()); // Output: 42
}
func FuzzValue(f *testing.F) { f.Add(1); f.Add(2); f.Fuzz(func(t *testing.T, x int) {
 if Value()!=42 { t.Fatal("value") } }) }
''')
        # Added after the original fixture: no runner/allowlist update.
        (ui / "new_test.go").write_text('''package ui
import "testing"
func TestNew(t *testing.T) { if Value()!=42 { t.Fatal("new") } }
''')
        other = self.root / "other"
        other.mkdir()
        (other / "value.go").write_text("package other\nfunc Value() int { return 7 }\n")
        (other / "value_test.go").write_text('''package other
import "testing"
func TestOther(t *testing.T) { if Value()!=7 { t.Fatal("other") } }
''')
        launcher = self.root / "internal/launcher"
        launcher.mkdir(parents=True)
        (launcher / "launch.go").write_text("package launcher\nfunc Value() int { return 42 }\n")
        for name in ["value_test.go", "new_test.go"]:
            text = (ui / name).read_text().replace("package ui", "package launcher", 1)
            (launcher / name).write_text(text)
        cli = self.root / "internal/cli"
        cli.mkdir(parents=True)
        (cli / "value.go").write_text("package cli\nfunc Value() int { return 42 }\n")
        for name in ["value_test.go", "new_test.go"]:
            text = (ui / name).read_text().replace("package ui", "package cli", 1)
            (cli / name).write_text(text)
        result = self.command(self.cli_command(self.runner), timeout=180)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        expected = Counter(["TestAlpha", "TestAlpha/a", "TestAlpha/b", "TestBravo",
                            "TestCharlie", "ExampleValue", "FuzzValue",
                            "FuzzValue/seed#0", "FuzzValue/seed#1", "TestNew"])
        for label in ["ui", "launcher", "cli"]:
            journals = [json.loads(line)
                        for p in (self.root / "test-results").glob(label+"-?.jsonl")
                        for line in p.read_text().splitlines()]
            ran = Counter(e["Test"] for e in journals if e["Action"] == "run")
            self.assertEqual(ran, expected)
        lines = (self.root / "coverage.out").read_text().splitlines()
        self.assertTrue(any("/launcher/launch.go:" in line for line in lines))
        self.assertTrue(any("/ui/value.go:" in line for line in lines))
        self.assertTrue(any("/cli/value.go:" in line for line in lines))
        self.assertTrue(any("/other/value.go:" in line for line in lines))
        cover = self.command(["go", "tool", "cover", "-func=coverage.out"], timeout=60)
        self.assertEqual(cover.returncode, 0, cover.stderr)
        self.assertIn("100.0%", cover.stdout)


class FailureEvidenceCLI(unittest.TestCase):
    def test_failure_evidence_survives_cleanup(self):
        # Exercise the acceptance subprocess boundary and unittest's real
        # timeout/error/assertion reporting, including its tearDown cleanup.
        driver = r"""
import os, pathlib, sys, unittest
import test_coverage_test as acceptance
class Probe(acceptance.CoverageCLI):
    def runTest(self):
        (self.root / "coverage.out").write_text("stale success")
        child = self.root / "controlled.py"
        child.write_text('''import json, pathlib, sys, time
root = pathlib.Path.cwd()
(root / "coverage.out").unlink()
(root / "test-results").mkdir()
(root / "test-results/partial.jsonl").write_text('{"Action":"run"}\\n')
print("partial stdout", flush=True)
print("partial stderr", file=sys.stderr, flush=True)
if sys.argv[1] == "timeout": time.sleep(30)
sys.exit(7 if sys.argv[1] == "nonzero" else 0)
''')
        result = self.command([sys.executable, str(child), os.environ["PROBE_MODE"]],
                              timeout=0.5 if os.environ["PROBE_MODE"] == "timeout" else 10)
        self.assertEqual(result.returncode, 0)
        self.fail("controlled assertion")
unittest.main(argv=[sys.argv[0]], defaultTest="Probe.runTest")
"""
        for mode in ["timeout", "nonzero", "assertion"]:
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                artifacts = Path(directory) / "durable"
                env = dict(os.environ, ONEBASE_ACCEPTANCE_ARTIFACTS=str(artifacts), PROBE_MODE=mode)
                result = subprocess.run([sys.executable, "-c", driver],
                                        cwd=Path(__file__).parent, env=env,
                                        capture_output=True, text=True, timeout=60)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                saved, = artifacts.glob("coverage-failure-*")
                fixture = saved / "fixture"
                record = json.loads((fixture / "acceptance-1.json").read_text())
                self.assertFalse(Path(record["cwd"]).exists(), "temporary fixture was not cleaned")
                self.assertEqual((fixture / "acceptance-1.stdout").read_bytes(), b"partial stdout\n")
                self.assertEqual((fixture / "acceptance-1.stderr").read_bytes(), b"partial stderr\n")
                self.assertTrue((fixture / "test-results/partial.jsonl").exists())
                self.assertFalse((fixture / "coverage.out").exists())
                self.assertIn("python", json.loads((saved / "versions.json").read_text()))
                if mode == "timeout":
                    self.assertEqual(record["state"], "timeout")
                    self.assertTrue(record["reaped"])
                    self.assertNotEqual(record["exit"], 0)
                    self.assertIn("TimeoutExpired", result.stderr)
                elif mode == "nonzero":
                    self.assertEqual(record["exit"], 7)
                else:
                    self.assertEqual(record["exit"], 0)
                    self.assertIn("controlled assertion", result.stderr)


if __name__ == "__main__":
    unittest.main()
