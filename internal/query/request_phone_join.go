package query

import (
	"strings"

	"github.com/ivantit66/onebase/internal/i18n/i18nerr"
	"github.com/ivantit66/onebase/internal/metadata"
)

const requestPhoneJoinName = "телефонызаявоксовпадают"

var requestPhoneFields = []string{"телефонмобильный", "телефондомашний", "телефонрабочий"}

func (tr *translator) emitRequestPhoneJoin() error {
	if !tr.opts.CompareRequestPhones {
		return i18nerr.New("Сопоставление телефонов заявок доступно только в настроенном встроенном отчёте")
	}

	call := tr.pos - 1
	scope, ok := tr.sourceCtx.scopeIDAt(call)
	left, right := tr.peek(1), tr.peek(3)
	if !ok || tr.sourceCtx.scopes[scope].parent >= 0 || tr.sourceCtx.sectionAt(call) != sectionFrom ||
		left.kind != tIdent || right.kind != tIdent || tr.peek(2).kind != tComma || tr.peek(4).kind != tRParen {
		return requestPhoneJoinArgsError()
	}
	leftAlias, rightAlias := lowerFast(left.val), lowerFast(right.val)
	joined := tr.onSourceAliases[scope]
	if leftAlias == rightAlias || joined == "" || joined != leftAlias && joined != rightAlias {
		return requestPhoneJoinArgsError()
	}
	leftColumns, leftOK := tr.requestPhoneJoinColumns(scope, call, leftAlias, leftAlias == joined)
	rightColumns, rightOK := tr.requestPhoneJoinColumns(scope, call, rightAlias, rightAlias == joined)
	if !leftOK || !rightOK {
		return requestPhoneJoinArgsError()
	}

	// Keep the report's exact string equality and its three ignored values.
	// These are server-only predicates, never expressions in the result.
	rightSet := strings.Join(rightColumns, ", ")
	comparisons := make([]string, 0, len(leftColumns))
	for _, col := range leftColumns {
		comparisons = append(comparisons, "("+col+" NOT IN ('', '(   )   -  -', '(495)   -  -') AND "+col+" IN ("+rightSet+"))")
	}
	tr.emit("(" + strings.Join(comparisons, " OR ") + ")")
	tr.pos += 5
	return nil
}

func requestPhoneJoinArgsError() error {
	return i18nerr.New("Сопоставление телефонов заявок ожидает два псевдонима А_Заявка в условии ПО")
}

func (tr *translator) requestPhoneJoinColumns(scope, call int, alias string, joined bool) ([]string, bool) {
	typ, source, known := tr.columnSourceAt(call, requestPhoneFields[0], alias)
	if !known || typ != metadata.FieldTypeString || source.tokenPos < 0 || source.tokenPos+4 >= call {
		return nil, false
	}
	sourceScope, sameScope := tr.sourceCtx.scopeIDAt(source.tokenPos)
	kind := upperFast(tr.tokens[source.tokenPos].val)
	entity := tr.tokens[source.tokenPos+2].val
	aliasKeyword, hasAlias := sqlKW(tr.tokens[source.tokenPos+3].val)
	if !sameScope || sourceScope != scope || !hasAlias || aliasKeyword != "AS" ||
		tr.tokens[source.tokenPos+4].kind != tIdent || lowerFast(tr.tokens[source.tokenPos+4].val) != alias ||
		(kind != "ДОКУМЕНТ" && kind != "DOCUMENT") || !strings.EqualFold(entity, "А_Заявка") {
		return nil, false
	}
	if joined {
		on := source.tokenPos + 5
		keyword, ok := sqlKW(tr.tokens[on].val)
		if on >= call || !ok || keyword != "ON" {
			return nil, false
		}
	}
	physical := sourcePhysicalColumns(kind, entity, tr.opts)
	columns := make([]string, 0, len(requestPhoneFields))
	for _, field := range requestPhoneFields {
		if source.fields[field] != metadata.FieldTypeString || physical[field] == "" {
			return nil, false
		}
		columns = append(columns, source.qualifier+"."+physical[field])
	}
	return columns, true
}
