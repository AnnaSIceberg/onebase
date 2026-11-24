package ui

import (
	"os/exec"
	"testing"
)

func TestManagedNestedTableFitBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is needed for the managed table layout test")
	}
	cmd := exec.CommandContext(t.Context(), node, "--test", "static/managed_nested_table_fit_behavior_test.js") //nolint:gosec // Test executable resolved from PATH.
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("managed nested table layout: %v\n%s", err, out)
	}
}
