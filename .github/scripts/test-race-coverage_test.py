import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("race_coverage", Path(__file__).with_name("test-race-coverage.py"))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class RaceCoverageTest(unittest.TestCase):
    def test_command_batches_and_validates_actual_execution_before_publishing_coverage(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fake = root / "go"
            fake.write_text("#!" + sys.executable + "\n" + '''
import json, os, re, sys
from pathlib import Path
a = sys.argv[1:]
root = Path(os.environ['ONEBASE_RUNNER_TEST_ROOT'])
with (root / 'calls.jsonl').open('a') as stream:
    stream.write(json.dumps(a) + '\\n')
names = ['TestCase' + str(i) for i in range(130)] + ['Example', 'FuzzSeed']
if a == ['list', './...']:
    print('example/other\\nexample/internal/ui')
elif a == ['test', '-list', '.', 'example/internal/ui']:
    print('\\n'.join(names))
else:
    assert all(flag in a for flag in ['-race', '-count=1', '-timeout=10m', '-covermode=atomic'])
    selected = []
    if a[-1] == 'example/internal/ui':
        pattern = a[a.index('-run') + 1]
        selected = [name for name in names if re.fullmatch(pattern, name)]
    if os.environ.get('ONEBASE_RUNNER_TEST_DROP') and selected:
        selected = selected[:-1]
    for name in selected:
        print(json.dumps({'Action': 'run', 'Test': name}))
    profile = Path(next(arg.split('=', 1)[1] for arg in a if arg.startswith('-coverprofile=')))
    file = 'ui.go' if a[-1] == 'example/internal/ui' else 'other.go'
    profile.write_text('mode: atomic\\n' + file + ':1.1,2.2 1 ' + str(len(selected)) + '\\n')
''', encoding="utf-8")
            fake.chmod(0o755)
            env = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"], ONEBASE_RUNNER_TEST_ROOT=str(root))
            out = root / "coverage.out"
            script = Path(__file__).with_name("test-race-coverage.py")
            result = subprocess.run([sys.executable, str(script), str(out)], env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(out.read_text(encoding="utf-8"), "mode: atomic\nother.go:1.1,2.2 1 0\nui.go:1.1,2.2 1 132\n")
            calls = [json.loads(line) for line in (root / "calls.jsonl").read_text().splitlines()]
            self.assertEqual(len([call for call in calls if "-run" in call]), 2)
            # Missing execution fails even if go test itself returned success;
            # the previous successful profile must also have been removed.
            env["ONEBASE_RUNNER_TEST_DROP"] = "1"
            result = subprocess.run([sys.executable, str(script), str(out)], env=env, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(out.exists())

    def test_inventory_includes_examples_and_fuzz_and_rejects_wrong_execution(self):
        names = runner.discover_names("TestA\nExample\nFuzzSeed\nok module 0.1s\n")
        runner.verify_inventory(names, list(reversed(names)))
        for actual in [names[:-1], names + [names[0]], names + ["TestUnexpected"]]:
            with self.assertRaises(ValueError):
                runner.verify_inventory(names, actual)
        for output in ["ok module 0.1s\n", "TestA\nTestA\n", "TestInvalid[\n"]:
            with self.assertRaises(ValueError):
                runner.discover_names(output)

    def test_merge_sums_atomic_counters_and_preserves_uncovered_blocks(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            profiles = [root / "other.out", root / "ui1.out", root / "ui2.out"]
            profiles[0].write_text("mode: atomic\nmodule/other.go:1.1,2.2 2 0\n", encoding="utf-8")
            profiles[1].write_text("mode: atomic\nmodule/ui.go:1.1,2.2 1 3\nmodule/ui.go:3.1,4.2 2 0\n", encoding="utf-8")
            profiles[2].write_text("mode: atomic\nmodule/ui.go:1.1,2.2 1 4\nmodule/ui.go:3.1,4.2 2 5\n", encoding="utf-8")
            out = root / "coverage.out"
            runner.merge_profiles(profiles, out)
            self.assertEqual(out.read_text(encoding="utf-8"), "mode: atomic\nmodule/other.go:1.1,2.2 2 0\nmodule/ui.go:1.1,2.2 1 7\nmodule/ui.go:3.1,4.2 2 5\n")
            for invalid in ["mode: set\n", "mode: atomic\n", "mode: atomic\nmodule/other.go:1.1,2.2 3 1\n", "mode: atomic\nmodule/other.go:1.1,2.2 2 -1\n"]:
                profiles[1].write_text(invalid, encoding="utf-8")
                selected = profiles[:2] if "other.go" in invalid else [profiles[1]]
                before = out.read_bytes()
                with self.assertRaises(ValueError):
                    runner.merge_profiles(selected, out)
                self.assertEqual(out.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
