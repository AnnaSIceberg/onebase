package query

import (
	"strings"

	"github.com/ivantit66/onebase/internal/i18n/i18nerr"
	"github.com/ivantit66/onebase/internal/metadata"
)

const optionalReferenceLabel = "представлениепридоступе"

// An optional label keeps the ordinary reference expansion for readers of the
// catalog. If the catalog is unavailable, lowering to NULL before projection
// analysis prevents an auto-JOIN and keeps the authorized source rows intact.
func rewriteOptionalReferenceLabels(tokens []tok, opts CompileOpts) ([]tok, error) {
	var out []tok
	var context sourceContext
	var sources map[int][]columnSource
	copied := 0
	for i, token := range tokens {
		if token.kind != tIdent || !strings.EqualFold(token.val, optionalReferenceLabel) || i+1 >= len(tokens) || tokens[i+1].kind != tLParen {
			continue
		}
		if sources == nil {
			context = preScanSourceContextWithOpts(tokens, opts)
			sources = indexColumnSources(tokens, opts, context)
		}
		scope, validScope := context.scopeIDAt(i)
		if !validScope || context.scopes[scope].parent >= 0 || !standaloneReferenceLabel(tokens, context, i) {
			return nil, optionalReferenceLabelError()
		}
		catalog, validCatalog := referenceLabelCatalog(tokens, opts, sources[scope], i)
		if !validCatalog {
			return nil, optionalReferenceLabelError()
		}
		out = append(out, tokens[copied:i]...)
		if opts.CanReadReferenceLabel != nil && !opts.CanReadReferenceLabel(catalog) {
			out = append(out, tok{kind: tIdent, val: "NULL"})
		} else {
			out = append(out, tokens[i+2:i+5]...)
			out = append(out, tok{kind: tDot, val: "."}, tok{kind: tIdent, val: "Наименование"})
		}
		copied = i + 6
	}
	if out == nil {
		return tokens, nil
	}
	return append(out, tokens[copied:]...), nil
}

func standaloneReferenceLabel(tokens []tok, context sourceContext, i int) bool {
	if i == 0 || i+8 >= len(tokens) || context.selectDepthAt(tokens, i) != context.tokenDepth[i] {
		return false
	}
	previous, _ := sqlKW(tokens[i-1].val)
	if tokens[i-1].kind != tComma && (tokens[i-1].kind != tIdent || previous != "SELECT" && previous != "DISTINCT") {
		return false
	}
	as, _ := sqlKW(tokens[i+6].val)
	next, _ := sqlKW(tokens[i+8].val)
	return tokens[i+2].kind == tIdent && tokens[i+3].kind == tDot &&
		tokens[i+4].kind == tIdent && tokens[i+5].kind == tRParen &&
		tokens[i+6].kind == tIdent && as == "AS" && tokens[i+7].kind == tIdent &&
		(tokens[i+8].kind == tComma || tokens[i+8].kind == tIdent && next == "FROM")
}

func referenceLabelCatalog(tokens []tok, opts CompileOpts, sources []columnSource, i int) (string, bool) {
	qualifier, field := lowerFast(tokens[i+2].val), lowerFast(tokens[i+4].val)
	var owner *columnSource
	for j := range sources {
		source := &sources[j]
		if source.qualifier == qualifier {
			if owner != nil {
				return "", false
			}
			owner = source
		}
	}
	if owner == nil || owner.tokenPos < 0 || owner.tokenPos+4 >= len(tokens) {
		return "", false
	}
	// Require an explicit alias on a physical source, not a derived table
	// or a virtual register table whose projection may alter the field type.
	pos := owner.tokenPos
	as, _ := sqlKW(tokens[pos+3].val)
	if tokens[pos+3].kind != tIdent || as != "AS" || tokens[pos+4].kind != tIdent || lowerFast(tokens[pos+4].val) != qualifier {
		return "", false
	}
	target, isReference := strings.CutPrefix(string(owner.fields[field]), "reference:")
	if !isReference {
		return "", false
	}
	for _, entity := range opts.Entities {
		if entity != nil && entity.Kind == metadata.KindCatalog && strings.EqualFold(entity.Name, target) {
			return entity.Name, true
		}
	}
	return "", false
}

func optionalReferenceLabelError() error {
	return i18nerr.New("ПредставлениеПриДоступе требует отдельную колонку выборки с алиасом и прямой ссылкой на справочник")
}
