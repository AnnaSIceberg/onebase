import importlib.util
import json
import os
import shutil
import contextlib
import io
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("race_coverage", Path(__file__).with_name("test-race-coverage.py"))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class RaceCoverageTest(unittest.TestCase):
    def test_command_covers_multiple_inventories_and_fails_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            fake = root / "go"
            fake.write_text("#!" + sys.executable + "\n" + r'''
import json, os, re, subprocess, sys
from pathlib import Path
a = sys.argv[1:]
root = Path(os.environ['ONEBASE_RUNNER_TEST_ROOT'])
with (root / 'calls.jsonl').open('a') as stream:
    stream.write(json.dumps(a) + '\n')
packages = ['example/' + p for p in ['other', 'internal/auth', 'internal/launcher', 'internal/ui', 'internal/dsl/interpreter']]
names = ['TestCase' + str(i) for i in range(257)] + ['Example', 'FuzzSeed']
mode = os.environ.get('ONEBASE_RUNNER_TEST_MODE', '')
if a == ['list', './...']:
    print('\n'.join(packages[:-1] if mode == 'missing-package' else packages))
elif a[:3] == ['list', '-f', '{{.Dir}}']:
    folder = root / a[-1]
    folder.mkdir(parents=True, exist_ok=True)
    print(folder)
elif '-c' in a:
    assert '-race' in a and '-covermode=atomic' in a
    if mode == 'discovery-error': sys.exit(1)
    binary = Path(a[a.index('-o') + 1])
    binary.write_text('#!' + sys.executable + '\n' + Path(__file__).read_text())
    binary.chmod(0o755)
elif a == ['-test.list=.']:
    inventory = [] if mode == 'empty' else names
    print('\n'.join(inventory))
elif a[:2] == ['tool', 'test2json']:
    assert '-p' in a and '-t' in a
    args = a[6:]
    env = dict(os.environ, FIXTURE_PACKAGE=a[3])
    sys.exit(subprocess.call([a[5], *args], env=env))
else:
    direct = a[0].startswith('-test.')
    if direct:
        assert all(flag in a for flag in ['-test.v=test2json', '-test.paniconexit0', '-test.count=1', '-test.timeout=10m'])
        assert os.environ['PWD'] == os.getcwd()
        assert Path(os.environ['GOCOVERDIR']).is_dir()
        package = os.environ['FIXTURE_PACKAGE']
        pattern = next(v.split('=', 1)[1] for v in a if v.startswith('-test.run='))
        selected = [name for name in names if re.fullmatch(pattern, name)]
        if mode == 'drop': selected = selected[:-1]
        if mode == 'duplicate': selected += selected[:1]
    else:
        assert all(flag in a for flag in ['-race', '-count=1', '-timeout=10m', '-covermode=atomic'])
        package, selected = a[-1], []
    if mode in ['error', 'timeout']:
        print(json.dumps({'Action':'output', 'Output':'panic: test timed out after 10m0s' if mode == 'timeout' else 'failure'}))
        sys.exit(1)
    for name in selected:
        for action, test in [('run', name), ('run', name + '/nested'), ('pass', name + '/nested'), ('pass', name)]:
            if action == 'pass' and test == name and mode == 'incomplete': continue
            print(json.dumps({'Action':action, 'Test':test, 'Package': 'wrong' if mode == 'wrong-package' else package, 'Elapsed': .01}))
    profile = Path(next(arg.split('=', 1)[1] for arg in a if arg.startswith(('-coverprofile=', '-test.coverprofile='))))
    profile.write_text('mode: atomic\n' + package + '.go:1.1,2.2 1 ' + str(len(selected)) + '\n')
    if mode == 'bad-coverage': profile.write_text('mode: set\n')
''', encoding="utf-8")
            fake.chmod(0o755)
            env = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"], ONEBASE_RUNNER_TEST_ROOT=str(root))
            out, report = root / "coverage.out", root / "report"
            script = Path(__file__).with_name("test-race-coverage.py")
            result = subprocess.run([sys.executable, str(script), str(out), str(report)], env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            evidence = json.loads((report / "report.json").read_text())
            self.assertTrue(evidence["success"])
            self.assertEqual(len(evidence["inventories"]), 3)
            calls = [json.loads(line) for line in (root / "calls.jsonl").read_text().splitlines()]
            executions = [call for call in calls if '-json' in call or call[:2] == ['tool', 'test2json']]
            self.assertEqual(len([call for call in calls if '-c' in call]), 3)
            self.assertEqual(len(executions), 2 + 3 * ((259 + runner.BATCH_SIZE - 1) // runner.BATCH_SIZE))
            self.assertEqual(executions[0][-1], 'example/internal/dsl/interpreter')
            self.assertEqual(executions[1][-1], 'example/other')
            self.assertEqual(executions[1][executions[1].index('-p') + 1], str(runner.OTHER_CONCURRENCY))
            for package in evidence['inventories']:
                selected = [p for p in evidence['processes'] if p['kind'] == 'tests' and p['packages'] == [package]]
                self.assertEqual(sum(len(p['names']) for p in selected), 259)
                self.assertTrue(all(len(p['names']) <= runner.BATCH_SIZE for p in selected))
                self.assertIn(package + '.go:1.1,2.2 1 259\n', out.read_text())
            self.assertEqual(out.read_text().count('mode: atomic'), 1)
            # A previous successful result must not survive any failed phase.
            for mode in ['missing-package', 'discovery-error', 'empty', 'drop', 'duplicate',
                         'incomplete', 'wrong-package', 'error', 'timeout', 'bad-coverage']:
                with self.subTest(mode=mode):
                    out.write_text('old success')
                    env['ONEBASE_RUNNER_TEST_MODE'] = mode
                    result = subprocess.run([sys.executable, str(script), str(out), str(report)], env=env, capture_output=True, text=True)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertFalse(out.exists())
                    self.assertFalse(json.loads((report / 'report.json').read_text())['success'])
                    if mode in {'error', 'timeout'}:
                        self.assertEqual(json.loads((report / 'report.json').read_text())['processes'][-1]['exit'], 1)

    def test_inventory_includes_examples_and_fuzz_and_rejects_wrong_execution(self):
        names = runner.discover_names("TestA\nExample\nFuzzSeed\nok module 0.1s\n")
        runner.verify_inventory(names, list(reversed(names)))
        for actual in [names[:-1], names + [names[0]], names + ["TestUnexpected"]]:
            with self.assertRaises(ValueError):
                runner.verify_inventory(names, actual)
        for output in ["ok module 0.1s\n", "TestA\nTestA\n", "TestInvalid[\n"]:
            with self.assertRaises(ValueError):
                runner.discover_names(output)

    def test_merge_sums_atomic_counters_and_rejects_corruption(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            profiles = [root / "other.out", root / "ui1.out", root / "ui2.out"]
            profiles[0].write_text("mode: atomic\nmodule/other.go:1.1,2.2 2 0\n", encoding="utf-8")
            profiles[1].write_text("mode: atomic\nmodule/ui.go:1.1,2.2 1 3\nmodule/ui.go:3.1,4.2 2 0\n", encoding="utf-8")
            profiles[2].write_text("mode: atomic\nmodule/ui.go:1.1,2.2 1 4\nmodule/ui.go:3.1,4.2 2 5\n", encoding="utf-8")
            out = root / "coverage.out"
            runner.merge_profiles(profiles, out)
            self.assertEqual(out.read_text(), "mode: atomic\nmodule/other.go:1.1,2.2 2 0\nmodule/ui.go:1.1,2.2 1 7\nmodule/ui.go:3.1,4.2 2 5\n")
            for invalid in ["mode: set\n", "mode: atomic\n", "mode: atomic\nmodule/other.go:1.1,2.2 3 1\n", "mode: atomic\nmodule/other.go:1.1,2.2 2 -1\n", "mode: atomic\nbad 1 1\n"]:
                profiles[1].write_text(invalid)
                selected = profiles[:2] if "other.go" in invalid else [profiles[1]]
                before = out.read_bytes()
                with self.assertRaises(ValueError):
                    runner.merge_profiles(selected, out)
                self.assertEqual(out.read_bytes(), before)

    def test_real_go_module_auto_includes_new_tests_examples_and_fuzz_seeds(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            (root / 'go.mod').write_text('module example\n\ngo 1.26.0\n')
            for package in ['other', *runner.BATCHED_PACKAGES, runner.ISOLATED_PACKAGE]:
                folder = root / package
                folder.mkdir(parents=True)
                (folder / 'value.go').write_text('package fixture\nfunc Value() int { return 1 }\n')
                (folder / 'race.go').write_text('//go:build race\npackage fixture\nconst raceEnabled = true\n')
                (folder / 'norace.go').write_text('//go:build !race\npackage fixture\nconst raceEnabled = false\n')
                source = r'''package fixture
import ("flag"; "fmt"; "os"; "testing"; "time")
func TestMain(m *testing.M) {
    flag.Parse()
    here, err := os.Stat("."); if err != nil {panic(err)}
    pwd, err := os.Stat(os.Getenv("PWD")); if err != nil || !os.SameFile(here, pwd) {panic("PWD")}
    if _, err := os.Stat("value.go"); err != nil {panic("cwd")}
    if !raceEnabled || testing.CoverMode() != "atomic" {panic("instrumentation")}
    fmt.Fprintln(os.Stderr, "fixture TestMain stderr")
    code := m.Run()
    f, err := os.OpenFile(os.Getenv("FIXTURE_RECEIPTS"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
    if err != nil {panic(err)}
    fmt.Fprintln(f, os.Getenv("PWD"), flag.Lookup("test.list").Value.String(), code)
    if err := f.Close(); err != nil {panic(err)}
    os.Exit(code)
}
func TestOriginal(t *testing.T) {
    switch os.Getenv("FIXTURE_MODE") {
    case "fail": t.Fatal("fixture failure")
    case "timeout": time.Sleep(time.Second)
    case "exit-zero": os.Exit(0)
    }
    t.Run("nested", func(t *testing.T) {if Value()!=1 {t.Fatal("value")}})
}
func ExampleValue() { fmt.Println(Value()); /* Output: 1 */ }
func FuzzValue(f *testing.F) { f.Add(1); f.Fuzz(func(t *testing.T, v int) { _ = Value() }) }
'''
                # Multiple real groups per package, discovered from the binary.
                source += "\n".join("func TestGenerated%d(t *testing.T) { _ = Value() }" % i for i in range(130))
                (folder / 'value_test.go').write_text(source)
            (root / 'internal/auth/new_test.go').write_text('package fixture\nimport "testing"\nfunc TestNewAutomaticallyIncluded(t *testing.T) { _ = Value() }\n')
            script = Path(__file__).with_name('test-race-coverage.py').resolve()
            real_go = shutil.which('go')
            wrapper = root / 'bin'
            wrapper.mkdir()
            calls = root / 'go-calls.jsonl'
            (wrapper / 'go').write_text('#!' + sys.executable + '\n' +
                'import json, os, sys\n' +
                'with open(' + repr(str(calls)) + ', "a") as f: f.write(json.dumps(sys.argv[1:]) + "\\n")\n' +
                'os.execv(' + repr(real_go) + ', [' + repr(real_go) + ', *sys.argv[1:]])\n')
            (wrapper / 'go').chmod(0o755)
            env = dict(os.environ, PATH=str(wrapper) + os.pathsep + os.environ['PATH'],
                       FIXTURE_RECEIPTS=str(root / 'receipts'))
            result = subprocess.run([sys.executable, str(script), str(root / 'coverage.out'), str(root / 'report')],
                                    cwd=root, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            evidence = json.loads((root / 'report/report.json').read_text())
            self.assertTrue(evidence['success'])
            auth = evidence['inventories']['example/internal/auth']
            self.assertTrue({'TestOriginal', 'TestNewAutomaticallyIncluded', 'ExampleValue', 'FuzzValue'} <= set(auth))
            for package, names in evidence['inventories'].items():
                executed = [test for process in evidence['processes'] for test in process.get('tests', []) if test['package'] == package]
                runner.verify_inventory(names, [test['name'] for test in executed])
                self.assertTrue(all(test['action'] == 'pass' for test in executed))
                groups = [p for p in evidence['processes'] if p['kind'] == 'tests' and p['packages'] == [package]]
                self.assertEqual(len(groups), 2)
                events = []
                for i in range(2):
                    log = (root / 'report' / (package.rsplit('/', 1)[-1] + f'-{i+1}.jsonl')).read_text()
                    self.assertIn('fixture TestMain stderr', log)
                    events.extend(json.loads(line) for line in log.splitlines())
                passed = {event.get('Test') for event in events if event.get('Action') == 'pass'}
                self.assertTrue({'TestOriginal/nested', 'ExampleValue', 'FuzzValue/seed#0'} <= passed)
            commands = [json.loads(line) for line in calls.read_text().splitlines()]
            self.assertEqual(len([a for a in commands if '-c' in a]), 3)
            self.assertEqual(len([a for a in commands if a[:2] == ['tool', 'test2json']]), 6)
            # TestMain runs for three discoveries and every execution; cleanup
            # receipts after m.Run distinguish discovery from real groups.
            receipts = (root / 'receipts').read_text().splitlines()
            self.assertEqual(len(receipts), 11)
            self.assertEqual(sum(' . ' in line for line in receipts), 3)
            cover = subprocess.run(['go', 'tool', 'cover', '-func=' + str(root / 'coverage.out')], cwd=root, capture_output=True, text=True)
            self.assertEqual(cover.returncode, 0, cover.stderr)
            self.assertIn('100.0%', cover.stdout)
            # Exercise failures/timeouts with the actual compiled binary and
            # test2json, including panic-on-exit-zero and stderr preservation.
            with patch.dict(os.environ, env), contextlib.chdir(root), contextlib.redirect_stdout(io.StringIO()):
                names, binary, cwd = runner.discover_package('example/internal/auth', root / 'auth.test', {'processes': []})
                for mode in ['fail', 'timeout', 'exit-zero']:
                    with self.subTest(mode=mode), patch.dict(os.environ, FIXTURE_MODE=mode):
                        report = {'processes': []}
                        log = root / (mode + '.jsonl')
                        with self.assertRaises(subprocess.CalledProcessError):
                            runner.run_tests(['example/internal/auth'], root / (mode + '.out'), ['TestOriginal'],
                                             binary=binary, cwd=cwd, report=report, log=log, timeout='50ms' if mode == 'timeout' else '10m')
                        self.assertNotEqual(report['processes'][0]['exit'], 0)
                        self.assertIn('fixture TestMain stderr', log.read_text())
                        self.assertIn('fail', [json.loads(line)['Action'] for line in log.read_text().splitlines()])


if __name__ == "__main__":
    unittest.main()
