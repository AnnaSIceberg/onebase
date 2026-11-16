package query

import "fmt"

// A UNION is a scope with positional output, not a bag of field names. Its
// global ORDER may only bind the merged outputs of every participating branch.
func (p *reportLineageParser) parseUnion() (bool, error) {
	depth := 0
	var separators []int
	orderAt := -1
	for i, t := range p.tokens {
		switch t.kind {
		case tLParen:
			depth++
		case tRParen:
			depth--
		}
		if depth != 0 || t.kind != tIdent {
			continue
		}
		kw, _ := sqlKW(t.val)
		if kw == "UNION" {
			separators = append(separators, i)
		}
		if kw == "ORDER" {
			orderAt = i
		}
	}
	if len(separators) == 0 {
		return false, nil
	}
	if len(separators) > 16 {
		return true, fmt.Errorf("report lineage: union budget")
	}
	if orderAt >= 0 && orderAt < separators[len(separators)-1] {
		return true, fmt.Errorf("report lineage: branch ORDER unsupported")
	}
	last := len(p.tokens)
	if last > 0 && p.tokens[last-1].kind == tEOF {
		last--
	}
	if orderAt >= 0 {
		last = orderAt
	}
	starts := []int{0}
	ends := []int{}
	operations := []string{}
	for _, at := range separators {
		ends = append(ends, at)
		next := at + 1
		op := "union-distinct"
		if next < len(p.tokens) && p.tokens[next].kind == tIdent {
			if kw, _ := sqlKW(p.tokens[next].val); kw == "ALL" {
				next++
				op = "union-all"
			}
		}
		if next >= len(p.tokens) || p.tokens[next].kind != tIdent {
			return true, fmt.Errorf("report lineage: UNION branch")
		}
		if kw, _ := sqlKW(p.tokens[next].val); kw != "SELECT" {
			return true, fmt.Errorf("report lineage: UNION SELECT")
		}
		starts = append(starts, next)
		operations = append(operations, op)
	}
	ends = append(ends, last)
	branches := make([]*reportLineageParser, 0, len(starts))
	for i, start := range starts {
		if start >= ends[i] {
			return true, fmt.Errorf("report lineage: empty UNION branch")
		}
		tokens := append(append([]tok(nil), p.tokens[start:ends[i]]...), tok{kind: tEOF})
		branch := &reportLineageParser{tokens: tokens, opts: p.opts, state: p.state, scopeID: len(p.state.scopes)}
		p.state.scopes = append(p.state.scopes, branch)
		if len(p.state.scopes) > 32 {
			return true, fmt.Errorf("report lineage: scope budget")
		}
		if err := branch.parse(); err != nil {
			return true, err
		}
		if i > 0 && len(branch.outputs) != len(branches[0].outputs) {
			return true, fmt.Errorf("report lineage: UNION output arity")
		}
		branches = append(branches, branch)
	}
	for column, first := range branches[0].outputs {
		merged, err := p.newExpr("scope-output", "")
		if err != nil {
			return true, err
		}
		merged.owner, merged.outputIndex = branches[0], column
		for i := 1; i < len(branches); i++ {
			next, err := p.newExpr("scope-output", "")
			if err != nil {
				return true, err
			}
			next.owner, next.outputIndex = branches[i], column
			merged, err = p.newExpr(operations[i-1], "", merged, next)
			if err != nil {
				return true, err
			}
		}
		p.outputs = append(p.outputs, reportOutput{name: first.name, expr: merged})
		// Duplicate elimination depends on all selected keys, even when a parent
		// later selects only a count. Keep it in the control dependency channel.
		for _, op := range operations {
			if op == "union-distinct" {
				p.uses = append(p.uses, reportUse{clause: "distinct", expr: merged})
				break
			}
		}
	}
	if orderAt >= 0 {
		p.pos = orderAt
		if err := p.want("ORDER"); err != nil {
			return true, err
		}
		if err := p.wantBy(); err != nil {
			return true, err
		}
		if err := p.list("order"); err != nil {
			return true, err
		}
		if p.peek().kind != tEOF {
			return true, fmt.Errorf("report lineage: UNION order tail")
		}
	}
	return true, nil
}
