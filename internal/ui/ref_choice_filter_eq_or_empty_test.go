package ui

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Пустой источник у eq закрывает подбор (fail closed), а у eq_or_empty —
// оставляет записи с пустым реквизитом: реклама без филиала доступна и в
// звонке, где филиал ещё не выбран (как список отбора 1С из пустой ссылки).
func TestChoicePredicatesEqualOrEmptyEmptySource(t *testing.T) {
	target := &metadata.Entity{
		Name: "Реклама", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Филиал", Type: "reference:Филиал", RefEntity: "Филиал"},
		},
	}
	element := func(op metadata.FormChoiceOperator) *metadata.FormElement {
		return &metadata.FormElement{
			ID: "ad", Name: "ПолеРеклама", Kind: metadata.FormElementField, DataPath: "Объект.Реклама",
			ChoiceFilter: []metadata.FormChoiceCondition{{Field: "Филиал", Op: op, From: "Объект.Филиал"}},
		}
	}
	empty := map[string]string{"Объект.Филиал": ""}
	// Метод сервера (после #1769): источник без разыменования, поэтому ни
	// реестр, ни хранилище не нужны — пустой сервер.
	s := &Server{}
	ctx := context.Background()

	predicates, closed, err := s.choicePredicates(ctx, nil, nil, target, element(metadata.FormChoiceOpEqualOrEmpty), empty)
	if err != nil {
		t.Fatalf("eq_or_empty: %v", err)
	}
	if closed || len(predicates) != 1 || predicates[0].Value != nil {
		t.Fatalf("eq_or_empty with empty source: closed=%v predicates=%+v, want one predicate with nil value", closed, predicates)
	}

	if _, closed, err := s.choicePredicates(ctx, nil, nil, target, element(metadata.FormChoiceOpEqual), empty); err != nil || !closed {
		t.Fatalf("eq with empty source: closed=%v err=%v, want fail-closed", closed, err)
	}

	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	predicates, closed, err = s.choicePredicates(ctx, nil, nil, target, element(metadata.FormChoiceOpEqualOrEmpty), map[string]string{"Объект.Филиал": id.String()})
	if err != nil || closed || len(predicates) != 1 || predicates[0].Value != id {
		t.Fatalf("eq_or_empty with source: closed=%v err=%v predicates=%+v", closed, err, predicates)
	}
}
