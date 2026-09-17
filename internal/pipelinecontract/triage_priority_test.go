package pipelinecontract

import (
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type trustedContributorWorkflowTrigger struct {
	Types []string `yaml:"types"`
}

type trustedContributorWorkflowStep struct {
	Uses  string            `yaml:"uses"`
	Env   map[string]string `yaml:"env"`
	Shell string            `yaml:"shell"`
	Run   string            `yaml:"run"`
}

type trustedContributorWorkflowJob struct {
	If             string                           `yaml:"if"`
	RunsOn         string                           `yaml:"runs-on"`
	TimeoutMinutes int                              `yaml:"timeout-minutes"`
	Permissions    map[string]string                `yaml:"permissions"`
	Steps          []trustedContributorWorkflowStep `yaml:"steps"`
}

type trustedContributorWorkflow struct {
	On          map[string]trustedContributorWorkflowTrigger `yaml:"on"`
	Permissions map[string]string                            `yaml:"permissions"`
	Jobs        map[string]trustedContributorWorkflowJob     `yaml:"jobs"`
}

func TestTrustedContributorPriorityWorkflowIsNarrowAndTextIndependent(t *testing.T) {
	raw := repositoryFile(t, ".github", "workflows", "trusted-contributor-priority.yml")
	var workflow trustedContributorWorkflow
	if err := yaml.Unmarshal([]byte(raw), &workflow); err != nil {
		t.Fatal(err)
	}

	if len(workflow.On) != 1 {
		t.Fatalf("workflow triggers = %#v, want only issues", workflow.On)
	}
	issues, ok := workflow.On["issues"]
	if !ok || len(issues.Types) != 1 || issues.Types[0] != "opened" {
		t.Fatalf("issues trigger = %#v, want only issues.opened", issues)
	}
	if len(workflow.Permissions) != 1 || workflow.Permissions["issues"] != "write" {
		t.Fatalf("workflow permissions = %#v, want only issues: write", workflow.Permissions)
	}
	if len(workflow.Jobs) != 1 {
		t.Fatalf("workflow jobs = %#v, want one narrow mutation job", workflow.Jobs)
	}
	job, ok := workflow.Jobs["prioritize"]
	if !ok {
		t.Fatal("workflow must define the prioritize job")
	}
	if strings.TrimSpace(job.If) != "github.repository == 'ivanarama/onebase' && github.event.issue.user.id == 330018641" {
		t.Fatalf("workflow identity gate = %q, want the upstream repository and immutable numeric GitHub user id", job.If)
	}
	if job.RunsOn != "ubuntu-latest" || job.TimeoutMinutes != 2 {
		t.Fatalf("workflow runner budget = %q/%d, want ubuntu-latest/2 minutes", job.RunsOn, job.TimeoutMinutes)
	}
	if len(job.Permissions) != 0 {
		t.Fatalf("job must not widen top-level permissions: %#v", job.Permissions)
	}
	if len(job.Steps) != 1 {
		t.Fatalf("workflow steps = %d, want one REST mutation", len(job.Steps))
	}
	step := job.Steps[0]
	if step.Uses != "" {
		t.Fatalf("workflow must not execute a third-party action: %q", step.Uses)
	}
	if len(step.Env) != 2 || step.Env["GH_TOKEN"] != "${{ github.token }}" ||
		step.Env["ISSUE_NUMBER"] != "${{ github.event.issue.number }}" {
		t.Fatalf("workflow environment = %#v, want only token and numeric issue id", step.Env)
	}
	if step.Shell != "bash" {
		t.Fatalf("workflow shell = %q, want bash", step.Shell)
	}
	const expectedScript = `set -euo pipefail
if [[ ! "$ISSUE_NUMBER" =~ ^[0-9]+$ ]]; then
  exit 1
fi
gh api --method POST \
  "repos/ivanarama/onebase/issues/${ISSUE_NUMBER}/labels" \
  --input - <<'JSON'
{"labels":["queue:p0"]}
JSON
`
	if step.Run != expectedScript {
		t.Fatalf("workflow script must remain the single exact label mutation\ngot:\n%s\nwant:\n%s", step.Run, expectedScript)
	}
	if strings.Count(step.Run, "queue:p0") != 1 {
		t.Fatalf("queue:p0 must be the one literal label payload, got %d occurrences", strings.Count(step.Run, "queue:p0"))
	}
	remainingEventContext := strings.Replace(raw, "github.event.issue.user.id", "", 1)
	remainingEventContext = strings.Replace(remainingEventContext, "github.event.issue.number", "", 1)
	if strings.Contains(remainingEventContext, "github.event") {
		t.Fatal("workflow must not consume any event data beyond immutable author id and numeric issue number")
	}
	rejectAll(t, raw,
		"pull_request_target",
		"workflow_dispatch",
		"github.event.issue.title",
		"github.event.issue.body",
		"github.event.comment",
		"actions/checkout",
		"approved",
		"ready-fix",
		"reviewed",
		"ship",
		"changes-requested",
		"needs-decision",
		"hold",
		"manual",
		"in-work",
		"plan-needed",
		"plan-in-review",
	)
}

type modeledTriagePriorityItem struct {
	number    int
	createdAt time.Time
	recovery  bool
	labels    map[string]bool
}

func modeledTriageEffectivePriority(item modeledTriagePriorityItem, now time.Time) int {
	priority := 2
	found := false
	for value := 0; value <= 3; value++ {
		if item.labels["queue:p"+strconv.Itoa(value)] {
			priority = value
			found = true
			break
		}
	}
	if !found {
		for value := 0; value <= 3; value++ {
			if item.labels["queue:auto:p"+strconv.Itoa(value)] {
				priority = value
				found = true
				break
			}
		}
	}
	if !found {
		switch {
		case item.labels["security"], item.labels["severity:critical"], item.labels["blocker"], item.labels["data-loss"]:
			priority = 0
		case item.labels["bug"]:
			priority = 1
		case item.labels["enhancement"], item.labels["documentation"]:
			priority = 2
		case item.labels["question"]:
			priority = 3
		}
	}

	if priority > 1 && now.After(item.createdAt) {
		boost := int(now.Sub(item.createdAt) / (7 * 24 * time.Hour))
		if boost > priority-1 {
			boost = priority - 1
		}
		priority -= boost
	}
	return priority
}

func modeledTriagePriorityQueue(items []modeledTriagePriorityItem, now time.Time) []int {
	recovery := make([]modeledTriagePriorityItem, 0, len(items))
	ordinary := make([]modeledTriagePriorityItem, 0, len(items))
	for _, item := range items {
		if item.recovery {
			recovery = append(recovery, item)
		} else {
			ordinary = append(ordinary, item)
		}
	}
	oldestFirst := func(left, right modeledTriagePriorityItem) bool {
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.Before(right.createdAt)
		}
		return left.number < right.number
	}
	sort.Slice(recovery, func(i, j int) bool { return oldestFirst(recovery[i], recovery[j]) })
	sort.Slice(ordinary, func(i, j int) bool {
		leftPriority := modeledTriageEffectivePriority(ordinary[i], now)
		rightPriority := modeledTriageEffectivePriority(ordinary[j], now)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return oldestFirst(ordinary[i], ordinary[j])
	})

	ordered := append(recovery, ordinary...)
	if len(ordered) > 5 {
		ordered = ordered[:5]
	}
	numbers := make([]int, 0, len(ordered))
	for _, item := range ordered {
		numbers = append(numbers, item.number)
	}
	return numbers
}

func TestTriageRecoveryPrecedesPriorityAndNewIssuesUseStablePriorityOrder(t *testing.T) {
	triage := skill(t, "triage-issues")
	docs := repositoryFile(t, "docs", "maintenance-pipeline.md")
	requireAllCompact(t, triage,
		"Recovery — абсолютная первая очередь независимо от любых priority labels",
		"Ни одна новая issue, включая P0, не обходит исполнимую recovery-транзакцию",
		"(effective priority ASC, created_at ASC, number ASC)",
		"ручная `queue:p0`…`queue:p3` имеет приоритет над `queue:auto:p0`…`queue:auto:p3`",
		"применяй class labels в строгом порядке",
		"иначе `enhancement`/`documentation` → P2, иначе `question` → P3, иначе P2",
		"За каждые полные 168 часов с `created_at` уменьши числовой уровень на один, но не ниже P1",
		"P0 остаётся отдельной полосой срочной работы",
	)
	requireAllCompact(t, docs,
		"незавершённое recovery всегда идёт первым",
		"новые заявки сортируются по effective priority, времени создания и номеру",
		"она обгоняет старые новые заявки, но не незавершённую recovery-транзакцию",
	)
	adapter := repositoryFile(t, ".agents", "skills", "triage-issues", "SKILL.md")
	requireAllCompact(t, adapter, "Логика конвейера живёт только в канонической процедуре")
	rejectAll(t, adapter, "effective priority ASC", "queue:p0")

	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	at := func(daysAgo int) time.Time { return now.Add(-time.Duration(daysAgo) * 24 * time.Hour) }
	items := []modeledTriagePriorityItem{
		{number: 90, createdAt: at(1), recovery: true, labels: map[string]bool{"question": true}},
		{number: 61, createdAt: at(0), labels: map[string]bool{"queue:p0": true}},
		{number: 45, createdAt: at(20), labels: map[string]bool{}},
		{number: 80, createdAt: at(2), recovery: true, labels: map[string]bool{"queue:p3": true}},
		{number: 51, createdAt: at(1), labels: map[string]bool{"bug": true}},
		{number: 42, createdAt: at(0), labels: map[string]bool{"enhancement": true}},
		{number: 41, createdAt: at(0), labels: map[string]bool{"documentation": true}},
	}
	want := []int{80, 90, 61, 45, 51}
	got := modeledTriagePriorityQueue(items, now)
	if len(got) != len(want) {
		t.Fatalf("ordered queue length = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ordered queue = %#v, want %#v", got, want)
		}
	}
	if tieOrder := modeledTriagePriorityQueue(items[5:], now); len(tieOrder) != 2 || tieOrder[0] != 41 || tieOrder[1] != 42 {
		t.Fatalf("same-priority same-time issues must use number as the final tie-breaker, got %#v", tieOrder)
	}

	manualWins := modeledTriagePriorityItem{
		createdAt: now,
		labels:    map[string]bool{"queue:p3": true, "queue:auto:p0": true},
	}
	if got := modeledTriageEffectivePriority(manualWins, now); got != 3 {
		t.Fatalf("manual priority must win over automatic priority, got P%d", got)
	}
	mixedClasses := modeledTriagePriorityItem{
		createdAt: now,
		labels:    map[string]bool{"enhancement": true, "question": true},
	}
	if got := modeledTriageEffectivePriority(mixedClasses, now); got != 2 {
		t.Fatalf("class priority must follow the shared precedence, got P%d for enhancement + question", got)
	}
	urgentOld := modeledTriagePriorityItem{createdAt: at(100), labels: map[string]bool{"queue:p0": true}}
	if got := modeledTriageEffectivePriority(urgentOld, now); got != 0 {
		t.Fatalf("P0 must remain outside aging, got P%d", got)
	}
}
