#!/usr/bin/env python3
"""Full race/atomic suite with the existing 128-name UI batching boundary."""

from collections import Counter
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile


def discover_names(output):
    names = []
    for line in output.splitlines():
        if line.startswith(("Test", "Example", "Fuzz")):
            if not re.fullmatch(r"(?:Test|Example|Fuzz)\w*", line):
                raise ValueError(f"invalid discovered name: {line!r}")
            names.append(line)
    if not names or len(set(names)) != len(names):
        raise ValueError("empty or duplicate UI test inventory")
    return names


def verify_inventory(expected, actual):
    if Counter(expected) != Counter(actual):
        raise ValueError(f"execution differs from discovery: expected={Counter(expected)}, actual={Counter(actual)}")


def merge_profiles(profiles, destination):
    counts, statements = {}, {}
    for profile in profiles:
        with profile.open(encoding="utf-8") as stream:
            if stream.readline() != "mode: atomic\n":
                raise ValueError(f"not atomic coverage: {profile}")
            for line in stream:
                location, size, count = line.split()
                size, count = int(size), int(count)
                if size < 0 or count < 0 or (location in statements and statements[location] != size):
                    raise ValueError(f"invalid coverage block: {line!r}")
                statements[location] = size
                counts[location] = counts.get(location, 0) + count
    if not counts:
        raise ValueError("coverage contains no source blocks")
    temporary = destination.with_name(destination.name + ".tmp")
    with temporary.open("w", encoding="utf-8", newline="\n") as stream:
        stream.write("mode: atomic\n")
        for location in sorted(counts):
            stream.write(f"{location} {statements[location]} {counts[location]}\n")
    temporary.replace(destination)


def run_tests(packages, profile, names=None):
    args = ["go", "test", "-json", "-count=1", "-race", "-timeout=10m",
            "-covermode=atomic", f"-coverprofile={profile}"]
    if names is not None:
        args += ["-run", "^(" + "|".join(re.escape(name) for name in names) + ")$"]
    args += packages
    print("Running:", *args, flush=True)
    runs = []
    with subprocess.Popen(args, stdout=subprocess.PIPE, text=True, encoding="utf-8") as proc:
        for line in proc.stdout:
            print(line, end="", flush=True)
            if names is not None:
                event = json.loads(line)
                name = event.get("Test", "")
                if event.get("Action") == "run" and name and "/" not in name:
                    runs.append(name)
        if proc.wait():
            raise subprocess.CalledProcessError(proc.returncode, args)
    if names is not None:
        verify_inventory(names, runs)
    return runs


def main():
    destination = Path(sys.argv[1] if len(sys.argv) > 1 else "coverage.out")
    destination.unlink(missing_ok=True)  # Never retain an older successful run.
    packages = subprocess.check_output(["go", "list", "./..."], text=True, encoding="utf-8").splitlines()
    ui = [package for package in packages if package.endswith("/internal/ui")]
    if len(ui) != 1 or len(packages) != len(set(packages)):
        raise ValueError("package inventory must contain exactly one UI package")
    other = [package for package in packages if package != ui[0]]
    names = discover_names(subprocess.check_output(["go", "test", "-list", ".", *ui], text=True, encoding="utf-8"))
    with tempfile.TemporaryDirectory(prefix="onebase-race-") as directory:
        directory = Path(directory)
        profiles = [directory / "other.out"]
        run_tests(other, profiles[0])
        runs = []
        for start in range(0, len(names), 128):
            batch = names[start:start + 128]
            profile = directory / f"ui-{start}.out"
            print(f"UI batch {start // 128 + 1}: {len(batch)} of {len(names)} names", flush=True)
            runs.extend(run_tests(ui, profile, batch))
            profiles.append(profile)
        verify_inventory(names, runs)
        merge_profiles(profiles, destination)
    print(f"UI inventory verified: {len(names)} names executed exactly once; coverage: {destination}")


if __name__ == "__main__":
    main()
