package query

import (
	"fmt"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
)

type reportLineageState struct {
	nextSource int
	scopes     []*reportLineageParser
}

func (p *reportLineageParser) bindOutputs() ([]ReportLineageOutput, error) {
	seen := map[string]bool{}
	out := make([]ReportLineageOutput, 0, len(p.outputs))
	for _, column := range p.outputs {
		key := metadata.SQLIdent(column.name)
		if seen[key] {
			return nil, fmt.Errorf("report lineage: duplicate output")
		}
		seen[key] = true
		view, err := p.bind(column.expr, "", 0)
		if err != nil {
			return nil, err
		}
		out = append(out, ReportLineageOutput{Name: column.name, Expression: view})
	}
	if p.selectDistinct {
		// SELECT DISTINCT changes row membership using every output, even when
		// a later renderer masks it. It is an explicit control dependency.
		exists := false
		for _, use := range p.uses {
			if use.clause == "select-distinct" {
				exists = true
			}
		}
		if !exists {
			for _, column := range p.outputs {
				p.uses = append(p.uses, reportUse{clause: "select-distinct", expr: column.expr})
			}
		}
	}
	return out, nil
}
func (p *reportLineageParser) derivedSource() error {
	start := p.pos + 1
	end, depth := start, 1
	for end < len(p.tokens) && depth > 0 {
		switch p.tokens[end].kind {
		case tLParen:
			depth++
		case tRParen:
			depth--
		}
		if depth > 0 {
			end++
		}
	}
	if depth != 0 || end <= start {
		return fmt.Errorf("report lineage: derived boundary")
	}
	childTokens := append(append([]tok(nil), p.tokens[start:end]...), tok{kind: tEOF})
	child := &reportLineageParser{tokens: childTokens, opts: p.opts, state: p.state, scopeID: len(p.state.scopes)}
	p.state.scopes = append(p.state.scopes, child)
	if len(p.state.scopes) > 32 {
		return fmt.Errorf("report lineage: scope budget")
	}
	if err := child.parse(); err != nil {
		return err
	}
	p.pos = end + 1
	if err := p.want("AS"); err != nil {
		return err
	}
	if p.peek().kind != tIdent || p.kw() != "" {
		return fmt.Errorf("report lineage: derived alias")
	}
	qualifier := strings.ToLower(p.peek().val)
	p.pos++
	for _, old := range p.sources {
		if strings.EqualFold(old.qualifier, qualifier) {
			return fmt.Errorf("report lineage: duplicate source qualifier")
		}
	}
	p.sources = append(p.sources, reportSource{qualifier: qualifier, derived: child, occurrence: -1})
	return nil
}
func (p *reportLineageParser) resolveView(path []string, depth int) (ReportExpression, error) {
	var candidates []int
	fieldPath := path
	if len(path) > 1 {
		for i, source := range p.sources {
			if strings.EqualFold(source.qualifier, path[0]) {
				candidates = append(candidates, i)
			}
		}
		if len(candidates) != 1 {
			return ReportExpression{}, fmt.Errorf("report lineage: field source unknown or ambiguous")
		}
		fieldPath = path[1:]
	} else {
		for i, source := range p.sources {
			if source.derived == nil {
				if _, ok := lineageOwnField(source, path[0]); ok {
					candidates = append(candidates, i)
				}
				continue
			}
			for _, column := range source.derived.outputs {
				if strings.EqualFold(column.name, path[0]) || strings.EqualFold(metadata.SQLIdent(column.name), path[0]) {
					candidates = append(candidates, i)
					break
				}
			}
		}
	}
	if len(candidates) != 1 || len(fieldPath) == 0 {
		return ReportExpression{}, fmt.Errorf("report lineage: field source unknown or ambiguous")
	}
	source := p.sources[candidates[0]]
	if source.derived == nil {
		origins, typ, err := p.resolve(path)
		return ReportExpression{Op: "field", Type: typ, Origins: origins}, err
	}
	if len(fieldPath) != 1 {
		return ReportExpression{}, fmt.Errorf("report lineage: derived navigation not proven")
	}
	outputs, err := source.derived.bindOutputs()
	if err != nil {
		return ReportExpression{}, err
	}
	var found *ReportExpression
	for _, column := range outputs {
		if strings.EqualFold(column.Name, fieldPath[0]) || strings.EqualFold(metadata.SQLIdent(column.Name), fieldPath[0]) {
			if found != nil {
				return ReportExpression{}, fmt.Errorf("report lineage: ambiguous derived output")
			}
			copy := column.Expression
			found = &copy
		}
	}
	if found == nil || found.Type == "" || metadata.IsReference(found.Type) {
		return ReportExpression{}, fmt.Errorf("report lineage: derived type not proven")
	}
	return ReportExpression{Op: "derived-field", Type: found.Type, Args: []ReportExpression{*found}}, nil
}

// A compact SQL DAG may expand exponentially when inspection is detached as a
// tree. Bound both expanded edges and metadata bytes before returning a receipt.
func lineageInspectionBounded(p *ReportLineage) bool {
	nodes, bytes := 0, 0
	var visit func(ReportExpression, int) bool
	visit = func(e ReportExpression, depth int) bool {
		nodes++
		bytes += len(e.Op) + len(e.Type) + len(e.Parameter) + 32
		for _, origin := range e.Origins {
			bytes += len(origin.Source.Kind) + len(origin.Source.Name) + len(origin.Source.TablePart) + len(origin.Field) + len(origin.Type)
			for _, part := range origin.Path {
				bytes += len(part)
			}
		}
		if nodes > 65536 || bytes > 4<<20 || depth > 64 {
			return false
		}
		for _, arg := range e.Args {
			if !visit(arg, depth+1) {
				return false
			}
		}
		return true
	}
	for _, column := range p.outputs {
		if !visit(column.Expression, 0) {
			return false
		}
	}
	for _, use := range p.uses {
		if !visit(use.Expression, 0) {
			return false
		}
	}
	return true
}
