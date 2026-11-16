package query

import (
	"fmt"

	"github.com/ivantit66/onebase/internal/metadata"
)

type reportInfoSlice struct {
	direction, boundary string
	dimensions          []metadata.Field
}

// Only the compiler's periodic information-register slices with an omitted
// filter and a single host parameter boundary are proven in this profile.
func (p *reportLineageParser) infoSlice() (*reportInfoSlice, error) {
	p.pos++ // dot
	if p.peek().kind != tIdent {
		return nil, fmt.Errorf("report lineage: slice method unknown")
	}
	direction := ""
	switch infoVTKinds[upperFast(p.peek().val)] {
	case "last_slice":
		direction = "desc"
	case "first_slice":
		direction = "asc"
	default:
		return nil, fmt.Errorf("report lineage: virtual source unsupported")
	}
	p.pos++
	if p.peek().kind != tLParen {
		return nil, fmt.Errorf("report lineage: slice arguments missing")
	}
	p.pos++
	slice := &reportInfoSlice{direction: direction}
	if p.peek().kind == tParam {
		if p.peek().val == "" {
			return nil, fmt.Errorf("report lineage: slice boundary name missing")
		}
		slice.boundary = p.peek().val
		p.pos++
	}
	if p.peek().kind == tComma {
		p.pos++ // an omitted filter, never an unanalysed expression
	}
	if p.peek().kind != tRParen {
		return nil, fmt.Errorf("report lineage: slice argument/filter not proven")
	}
	p.pos++
	return slice, nil
}

func (p *reportLineageParser) infoSliceUses() ([]ReportLineageUse, error) {
	var uses []ReportLineageUse
	for _, source := range p.sources {
		if source.slice == nil {
			continue
		}
		fieldView := func(name string) (ReportExpression, error) {
			field, ok := lineageOwnField(source, name)
			if name == "" || !ok || field.Type == "" {
				return ReportExpression{}, fmt.Errorf("report lineage: slice field ambiguous")
			}
			return ReportExpression{Op: "field", Type: field.Type, Origins: []ReportFieldOrigin{{Occurrence: source.occurrence, Source: source.ref, Field: field.Name, Type: field.Type, Path: []string{field.Name}}}}, nil
		}
		// The compiler's LatestPerKey partitions by EVERY declared dimension,
		// including protected dimensions not named anywhere in the source SQL.
		for _, dimension := range source.slice.dimensions {
			view, err := fieldView(dimension.Name)
			if err != nil {
				return nil, err
			}
			uses = append(uses, ReportLineageUse{Scope: p.scopeID, Clause: "slice-partition", Expression: view})
		}
		period, err := fieldView("period")
		if err != nil {
			return nil, err
		}
		uses = append(uses, ReportLineageUse{Scope: p.scopeID, Clause: "slice-order-" + source.slice.direction, Expression: period})
		if source.slice.boundary != "" {
			op := "<="
			if source.slice.direction == "asc" {
				op = ">="
			}
			bound := ReportExpression{Op: op, Type: metadata.FieldTypeBool, Args: []ReportExpression{period, {Op: "parameter", Parameter: source.slice.boundary}}}
			uses = append(uses, ReportLineageUse{Scope: p.scopeID, Clause: "slice-bound", Expression: bound})
		}
	}
	return uses, nil
}
