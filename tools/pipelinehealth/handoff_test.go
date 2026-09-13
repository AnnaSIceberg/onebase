package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// The identical fixture is consumed by PromptPilot's real fallback gate. This
// tests the public CLI including parsing, contract checks and finalization.
func TestFallbackHandoffConsumerParity(t *testing.T) {
	data, err := os.ReadFile("testdata/fallback-handoff-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Protocol string `json:"protocol"`
		Cases    []struct {
			Stage  string         `json:"stage"`
			Target map[string]any `json:"target"`
			Health map[string]any `json:"health"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Protocol != "promptpilot-fallback-target-v1" || len(corpus.Cases) != 7 {
		t.Fatal("unexpected handoff parity corpus")
	}
	for _, fixture := range corpus.Cases {
		stage := fixture.Target["stage"].(string)
		t.Run(stage, func(t *testing.T) {
			item := testPR(42, fixture.Target["head"].(string), "ship", "reviewed")
			switch stage {
			case "review":
				item = testPR(42, headA)
			case "merge":
				item = addComment(item, 40, completion(headA, 35, 36))
			case "integration-merge-recovery":
				item = addComment(item, 30, syncIntent(headA, 10, 20, 25))
			case "integration-review", "integration-merge-ready":
				item = addComment(item, 30, syncIntent(headA, 10, 20, 25))
				item = addComment(item, 31, syncDone(30, headA, headB))
				if stage == "integration-merge-ready" {
					item = addComment(item, 40, completion(headB, 35, 36))
				}
			case "legacy-integration-review", "legacy-integration-merge-ready":
				item = withMergeHead(addComment(item, 30, completion(headA, 20, 25)))
				if stage == "legacy-integration-merge-ready" {
					item = addComment(item, 31, completion(headB, 40, 45))
				}
			default:
				t.Fatal("unknown corpus stage")
			}
			input, err := json.Marshal([]apiPull{item})
			if err != nil {
				t.Fatal(err)
			}
			inputPath := filepath.Join(t.TempDir(), "prs.json")
			if err := os.WriteFile(inputPath, input, 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-prs", inputPath, "-json",
				"-contract", "../../.claude/skills/review-queue/SKILL.md")
			encoded, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("pipelinehealth: %v\n%s", err, encoded)
			}
			var actual report
			if err := json.Unmarshal(encoded, &actual); err != nil {
				t.Fatal(err)
			}
			var value map[string]any
			if err := json.Unmarshal(encoded, &value); err != nil {
				t.Fatal(err)
			}
			project := func(raw any) any {
				if raw == nil {
					return nil
				}
				candidate := raw.(map[string]any)
				return map[string]any{"number": candidate["number"], "head": candidate["head"], "stage": candidate["stage"]}
			}
			value["integration_owner"] = project(value["integration_owner"])
			for _, field := range []string{"review_candidates", "content_review_candidates", "merge_executable"} {
				items := []any{}
				for _, raw := range value[field].([]any) {
					items = append(items, project(raw))
				}
				value[field] = items
			}
			barriers := []any{}
			for _, found := range actual.Findings {
				if found.Code == "single_flight_barrier" {
					barriers = append(barriers, map[string]any{"code": found.Code, "severity": found.Severity, "pr": float64(found.PR)})
				}
			}
			value["findings"] = barriers
			for field, expected := range fixture.Health {
				if !reflect.DeepEqual(value[field], expected) {
					t.Errorf("producer field %s = %#v; consumer fixture expects %#v", field, value[field], expected)
				}
			}
		})
	}
}
