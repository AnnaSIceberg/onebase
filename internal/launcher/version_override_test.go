package launcher

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const versionOverrideChild = "ONEBASE_VERSION_OVERRIDE_CHILD"

// A previous HTTP test may leave an update watcher running. Keep version.Build
// writes (including their cleanup) in a process that never ran that watcher.
func versionOverrideSubprocess(t *testing.T) bool {
	t.Helper()
	if child := os.Getenv(versionOverrideChild); child != "" {
		if child != t.Name() {
			t.Fatalf("unexpected version override child sentinel: %q", child)
		}
		return false
	}
	ctx, cancel := versionOverrideContext(t, 30*time.Second)
	defer cancel()
	output, err := runVersionOverrideSubprocess(ctx, t.Name(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Keep go test from reporting the child percentage as the merged total.
	t.Logf("isolated version test:\n%s", strings.ReplaceAll(string(output), "coverage:", "child coverage="))
	return true
}

func versionOverrideContext(t *testing.T, budget time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	deadline := time.Now().Add(budget)
	if outer, ok := t.Deadline(); ok && outer.Add(-5*time.Second).Before(deadline) {
		deadline = outer.Add(-5 * time.Second)
	}
	return context.WithDeadline(t.Context(), deadline)
}

func runVersionOverrideSubprocess(ctx context.Context, name string, env []string) ([]byte, error) {
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 {
		return nil, fmt.Errorf("version subprocess requires a future deadline")
	}
	args := []string{"-test.run=^" + regexp.QuoteMeta(name) + "$", "-test.v", "-test.timeout=" + time.Until(deadline).String()}
	if testing.CoverMode() != "" {
		// go test supplies this directory for -coverprofile. Each process emits
		// its own counter file; the parent's coverage teardown merges them all
		// into the original profile, using the same instrumented executable.
		dir := flag.Lookup("test.gocoverdir")
		if dir == nil || dir.Value.String() == "" {
			return nil, fmt.Errorf("version subprocess: coverage enabled without test.gocoverdir")
		}
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}
	cmd := exec.CommandContext(ctx, os.Args[0], args...) //nolint:gosec // Current test executable, anchored exact test name, no shell.
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), versionOverrideChild+"="+name)
	cmd.Env = append(cmd.Env, env...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return output, fmt.Errorf("version subprocess %q: process error: %w; context error: %v; deadline exceeded: %t\n%s",
			name, err, ctx.Err(), errors.Is(ctx.Err(), context.DeadlineExceeded), output)
	}
	return output, nil
}

// Exercise the real runner so assertions, process failures, and hung children
// cannot become a passing wrapper test. No production updater is involved.
func TestVersionOverrideSubprocessFailures(t *testing.T) {
	const controlEnv = "ONEBASE_VERSION_OVERRIDE_FAILURE_CONTROL"
	if os.Getenv(versionOverrideChild) == t.Name() {
		switch os.Getenv(controlEnv) {
		case "assertion":
			t.Fatal("version child assertion reached parent")
		case "exit":
			fmt.Fprintln(os.Stderr, "version child nonzero exit")
			os.Exit(23)
		case "deadline":
			for {
				time.Sleep(time.Hour)
			}
		default:
			t.Fatal("missing subprocess failure control")
		}
	}
	for _, control := range []string{"assertion", "exit", "deadline"} {
		t.Run(control, func(t *testing.T) {
			ctx, cancel := versionOverrideContext(t, 5*time.Second)
			defer cancel()
			_, err := runVersionOverrideSubprocess(ctx, "TestVersionOverrideSubprocessFailures", []string{controlEnv + "=" + control})
			if err == nil || !strings.Contains(err.Error(), "process error:") {
				t.Fatalf("failed child did not reach parent: %v", err)
			}
			if control == "deadline" {
				if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
					t.Fatalf("hung child was not killed by parent deadline: %v", err)
				}
			} else {
				if ctx.Err() != nil || !strings.Contains(err.Error(), "version child ") {
					t.Fatalf("child failure lost or confused with timeout: %v", err)
				}
			}
		})
	}
}

// A foreign nonempty sentinel must fail before version mutation or a recursive
// subprocess launch. Exercise the guard through the same executable and runner.
func TestVersionOverrideSubprocessSentinel(t *testing.T) {
	const controlEnv = "ONEBASE_VERSION_OVERRIDE_SENTINEL_CONTROL"
	if os.Getenv(controlEnv) == "foreign" {
		versionOverrideSubprocess(t)
		t.Fatal("foreign sentinel passed the guard")
	}
	ctx, cancel := versionOverrideContext(t, 5*time.Second)
	defer cancel()
	_, err := runVersionOverrideSubprocess(ctx, t.Name(), []string{
		versionOverrideChild + "=foreign-test",
		controlEnv + "=foreign",
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected version override child sentinel") || ctx.Err() != nil {
		t.Fatalf("foreign sentinel was not rejected by the child: %v", err)
	}
}

func TestVersionOverrideSubprocessCancellation(t *testing.T) {
	const readyEnv = "ONEBASE_VERSION_OVERRIDE_CANCEL_READY"
	if os.Getenv(versionOverrideChild) == t.Name() {
		fmt.Fprintln(os.Stdout, "version child ready for cancellation")
		if err := os.WriteFile(os.Getenv(readyEnv), []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	ready := filepath.Join(t.TempDir(), "ready")
	ctx, cancel := versionOverrideContext(t, 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runVersionOverrideSubprocess(ctx, t.Name(), []string{readyEnv + "=" + ready})
		result <- err
	}()
	// Cancel only after the real child acknowledges startup. The deadline is
	// the failure bound; elapsed time alone never proves child readiness.
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		} else if !os.IsNotExist(err) {
			cancel()
			<-result
			t.Fatal(err)
		}
		select {
		case err := <-result:
			t.Fatalf("child exited before cancellation: %v", err)
		case <-ctx.Done():
			<-result
			t.Fatalf("child did not acknowledge startup: %v", ctx.Err())
		case <-tick.C:
		}
	}
	cancel()
	err := <-result
	if err == nil || !errors.Is(ctx.Err(), context.Canceled) ||
		!strings.Contains(err.Error(), "context error: context canceled") ||
		!strings.Contains(err.Error(), "version child ready for cancellation") {
		t.Fatalf("child cancellation was not propagated: %v", err)
	}
}
