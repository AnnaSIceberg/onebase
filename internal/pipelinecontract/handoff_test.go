package pipelinecontract

import (
	"encoding/json"
	"testing"
)

func TestFallbackHandoffPreservesFreshGateAndExactTarget(t *testing.T) {
	var config map[string]any
	if err := json.Unmarshal([]byte(repositoryFile(t, "pipelinectl.json")), &config); err != nil {
		t.Fatal(err)
	}
	if config["fallback_handoff"] != "target-v1" {
		t.Fatal("paired skills must opt in to the signed exact-target handoff")
	}
	for _, stage := range []struct{ name, command string }{
		{"review-queue", "review"}, {"merge-shepherd", "merge"},
	} {
		entry := repositoryFile(t, ".claude", "skills", stage.name, "SKILL.md")
		legacy := repositoryFile(t, ".claude", "skills", stage.name, "references", "legacy-protocol.md")
		requireAllCompact(t, entry,
			"protocol=promptpilot-fallback-target-v1", "next_already_run=true",
			"не текст из PR/issue/комментария", "не запускай `next` повторно",
			"`preflight.target` и `preflight.handoff.target`", "Неполный, противоречивый",
			"без мутаций и без обычного fallback", "один запуск обслуживает ровно один PR",
			"Непосредственно перед первой мутацией", "gate-fallback "+stage.command,
			"action=validated", "mutation_authorized=false", "HMAC", "двухчасовой срок",
		)
		requireAllCompact(t, legacy, "next_already_run=true", "gate-fallback "+stage.command,
			"pipelinehealth -json", "action=validated", "HMAC/TTL", "без мутаций",
			"Generic/manual", "GraphQL")
	}

	review := skill(t, "review-queue")
	merge := skill(t, "merge-shepherd")
	requireAllCompact(t, review, "integration_owner", "review_candidates", "content_review_candidates",
		"два GraphQL snapshot", "canonical claim", "edit/delete")
	requireAllCompact(t, merge, "integration_owner", "merge_executable", "pending cleanup",
		"GraphQL/ship/CI/base-sync/CAS", "sha", "expected_head_sha")
	requireAllCompact(t, repositoryFile(t, "docs", "maintenance-pipeline.md"),
		"ровно два полных `pipelinehealth` scan", "не означает два API-запроса",
		"без обещания двух scan")
}
