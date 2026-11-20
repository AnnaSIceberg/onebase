package query

import (
	"strings"

	"github.com/ivantit66/onebase/internal/i18n/i18nerr"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// SavedPhoneMatch is a server input, separate from the query parameters
// supplied by a form handler.
type SavedPhoneMatch struct {
	Target string
	Fields []string
	Keys   []string
}

func IsSavedPhoneMatchName(lower string) bool {
	return lower == "сохраненныйтелефонсовпадает" || lower == "saved_phone_match"
}

func UsesSavedPhoneMatch(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "сохраненныйтелефонсовпадает") || strings.Contains(lower, "saved_phone_match")
}

func (tr *translator) emitSavedPhoneMatch() error {
	match := tr.opts.SavedPhoneMatch
	if match == nil {
		return i18nerr.New("Подбор по сохранённому телефону доступен при открытии настроенной формы")
	}
	scope, ok := tr.sourceCtx.scopeIDAt(tr.pos - 1)
	alias := tr.peek(1)
	if !ok || tr.sourceCtx.scopes[scope].parent >= 0 ||
		tr.sourceCtx.sectionAt(tr.pos-1) != sectionWhere || alias.kind != tIdent || tr.peek(2).kind != tRParen {
		return i18nerr.New("СОХРАНЕННЫЙТЕЛЕФОНСОВПАДАЕТ ожидает псевдоним документа в условии ГДЕ")
	}
	if len(match.Fields) == 0 {
		return i18nerr.New("Подбор по сохранённому телефону: отсутствуют поля заявки")
	}
	qualifier := lowerFast(alias.val)
	typ, source, known := tr.columnSourceAt(tr.pos+1, lowerFast(match.Fields[0]), qualifier)
	if !known || typ != metadata.FieldTypeString || source.tokenPos < 0 {
		return i18nerr.New("СОХРАНЕННЫЙТЕЛЕФОНСОВПАДАЕТ ожидает псевдоним документа в условии ГДЕ")
	}
	sourceScope, local := tr.sourceCtx.scopeIDAt(source.tokenPos)
	kind, entity := upperFast(tr.tokens[source.tokenPos].val), tr.tokens[source.tokenPos+2].val
	if !local || sourceScope != scope || (kind != "ДОКУМЕНТ" && kind != "DOCUMENT") || !strings.EqualFold(entity, match.Target) {
		return i18nerr.New("Подбор по сохранённому телефону: документ не совпадает с настройкой формы")
	}
	tr.pos += 3
	if len(match.Keys) == 0 {
		tr.emit("(1=0)")
		return nil
	}

	physical := sourcePhysicalColumns(kind, entity, tr.opts)
	predicates := make([]string, 0, len(match.Fields))
	for _, field := range match.Fields {
		column := physical[lowerFast(field)]
		if column == "" {
			return i18nerr.New("Подбор по сохранённому телефону: отсутствуют поля заявки")
		}
		placeholders := make([]string, 0, len(match.Keys))
		for _, key := range match.Keys {
			tr.args = append(tr.args, key)
			placeholders = append(placeholders, dialectOrDefault(tr.opts.Dialect).Placeholder(len(tr.args)))
		}
		predicates = append(predicates, "("+storage.PhoneLast10SQL(source.qualifier+"."+column)+") IN ("+
			strings.Join(placeholders, ", ")+")")
	}
	tr.emit("(" + strings.Join(predicates, " OR ") + ")")
	return nil
}
