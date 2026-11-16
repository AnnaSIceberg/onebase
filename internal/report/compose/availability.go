package compose

import (
	"sort"
	"strings"

	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/token"
	"github.com/ivantit66/onebase/internal/report"
)

// Availability belongs to resource slots, separately from scalar values. An
// available nil keeps ordinary DSL arithmetic; a scope/context dependency may
// instead prevent evaluation. No state is shared between aggregate invocations.
type measureAvailability struct {
	blocked []bool
	reverse [][]int
	groups  []availabilityDependency
	queue   []int
	next    int
	writers map[string]int
}

type availabilityDependency struct {
	target    int
	remaining int
}

func (a *measureAvailability) mark(slot int) {
	if !a.blocked[slot] {
		a.blocked[slot] = true
		a.queue = append(a.queue, slot)
	}
}

func (a *measureAvailability) propagate(out Row) {
	// Each slot/group edge is visited at most once, including active cycles.
	for a.next < len(a.queue) {
		slot := a.queue[a.next]
		a.next++
		for _, index := range a.reverse[slot] {
			group := &a.groups[index]
			group.remaining--
			if group.remaining == 0 {
				a.mark(group.target)
			}
		}
	}
	for field, slot := range a.writers {
		if a.blocked[slot] {
			out[field] = nil
		}
	}
}

func (a *measureAvailability) dependency(target int, candidates []int) {
	if len(candidates) == 0 {
		return
	}
	seen := map[int]bool{}
	for _, slot := range candidates {
		seen[slot] = true
	}
	ids := make([]int, 0, len(seen))
	for slot := range seen {
		ids = append(ids, slot)
	}
	sort.Ints(ids)
	index := len(a.groups)
	a.groups = append(a.groups, availabilityDependency{target: target, remaining: len(ids)})
	for _, slot := range ids {
		a.reverse[slot] = append(a.reverse[slot], index)
	}
}

// Do not choose a new winner for duplicate names. Exact output-key writes use
// the existing aggregate/Expr order. A case-fold ambiguity is unavailable only
// when every possible bound slot is unavailable. Pending declarations are
// considered only when no previous writer exists, never instead of a real
// excluded previous writer (including an Expr reading its own old field).
func availabilityBindings(name string, writers map[string]int, pending map[string][]int) []int {
	var candidates []int
	for field, slot := range writers {
		if strings.EqualFold(name, field) {
			candidates = append(candidates, slot)
		}
	}
	if len(candidates) == 0 {
		return pending[strings.ToLower(name)]
	}
	return candidates
}

// Identifiers are bounded lexer tokens, excluding call targets and member names.
// Strings/comments never introduce report dependencies.
func availabilityIdentifiers(expr string) []string {
	if len(expr) > maxDependencyExprBytes {
		return nil
	}
	lx := lexer.New(expr, "")
	var tokens []token.Token
	for n := 0; n < maxDependencyExprTokens; n++ {
		t := lx.NextToken()
		if t.Type == token.EOF {
			break
		}
		tokens = append(tokens, t)
	}
	var names []string
	for n, t := range tokens {
		if t.Type != token.IDENT || (n > 0 && tokens[n-1].Type == token.DOT) || (n+1 < len(tokens) && tokens[n+1].Type == token.LPAREN) {
			continue
		}
		names = append(names, t.Literal)
	}
	return names
}

func newMeasureAvailability(declared, active []report.Measure, slots, order []int, ctx measureContext, out Row) *measureAvailability {
	a := &measureAvailability{blocked: make([]bool, len(declared)), reverse: make([][]int, len(declared)), writers: map[string]int{}}
	excluded := make([]bool, len(declared))
	for n := range excluded {
		excluded[n] = true
	}
	declaredNames := map[string]bool{}
	for _, m := range declared {
		declaredNames[strings.ToLower(m.Field)] = true
	}
	for _, slot := range slots {
		excluded[slot] = false
	}
	// This is exactly the write order already used by aggregateContext.
	for slot, m := range declared {
		if excluded[slot] {
			a.mark(slot)
			a.writers[m.Field] = slot
		}
	}
	for n, m := range active {
		if m.Expr == "" {
			a.writers[m.Field] = slots[n]
		}
	}
	planned := make(map[string]int, len(a.writers))
	for field, slot := range a.writers {
		planned[field] = slot
	}
	pending := map[string][]int{}
	for _, n := range order {
		m := active[n]
		pending[strings.ToLower(m.Field)] = append(pending[strings.ToLower(m.Field)], slots[n])
	}
	for _, n := range order {
		m, slot := active[n], slots[n]
		for _, name := range availabilityIdentifiers(m.Expr) {
			if declaredNames[strings.ToLower(name)] {
				a.dependency(slot, availabilityBindings(name, planned, pending))
				continue
			}
			scope, field, ok := report.MeasureReference(name)
			if !ok {
				continue
			}
			var source Row
			self := false
			switch scope {
			case "grand":
				source, self = ctx.grand, ctx.grandSelf
			case "previous":
				source = ctx.previous
			case "first":
				source, self = ctx.first, ctx.firstSelf
			}
			if self {
				ids := availabilityBindings(field, planned, pending)
				if len(ids) == 0 {
					a.mark(slot)
				} else {
					a.dependency(slot, ids)
				}
				continue
			}
			var value any
			for key, v := range source {
				if strings.EqualFold(key, field) {
					value = v
					break
				}
			}
			if value == nil {
				a.mark(slot)
			}
		}
		planned[m.Field] = slot
		key := strings.ToLower(m.Field)
		remaining := make([]int, 0, len(pending[key]))
		for _, candidate := range pending[key] {
			if candidate != slot {
				remaining = append(remaining, candidate)
			}
		}
		pending[key] = remaining
	}
	a.propagate(out)
	return a
}
