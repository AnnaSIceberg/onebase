package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestReportOutputRecordLabelKeepsCardPolicy(t *testing.T) {
	entity := &metadata.Entity{Name: "Protected", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{
		{Name: "Label", Type: metadata.FieldTypeString}, {Name: "House", Type: metadata.FieldTypeString, ReportPII: true},
	}}
	id := uuid.NewString()
	row := map[string]any{"id": id, "Label": "SYNTHETIC_HOUSE_FLAT", "House": "SYNTHETIC_HOUSE"}
	s := &Server{}
	ctx := auth.ContextWithUser(context.Background(), &auth.User{IsAdmin: true})
	// Представление из незащищённого поля в отчёте читается как обычно.
	if got := s.reportOutputRecordLabel(access.WithReportOutput(ctx), entity, row); got != "SYNTHETIC_HOUSE_FLAT" {
		t.Fatal("ordinary report label hidden", got)
	}
	// Представление из поля под report_pii сохраняет только UUID.
	protected := &metadata.Entity{Name: "ProtectedLabel", Kind: metadata.KindCatalog, Presentation: []string{"House"}, Fields: []metadata.Field{
		{Name: "House", Type: metadata.FieldTypeString, ReportPII: true}, {Name: "Label", Type: metadata.FieldTypeString},
	}}
	if got := s.reportOutputRecordLabel(access.WithReportOutput(ctx), protected, row); got != id {
		t.Fatal("protected report label must retain only its UUID", got)
	}
	if got := s.maskedRecordLabel(ctx, entity, row); got != "SYNTHETIC_HOUSE_FLAT" {
		t.Fatal("ordinary card label changed", got)
	}
}

// Пустое Наименование не открывает следующий строковый кандидат-ПДн: у
// админа карточная политика pii не маскирует, обязательная политика отчёта —
// маскирует всегда.
func TestReportOutputRecordLabelEmptyNameSkipsPII(t *testing.T) {
	entity := &metadata.Entity{Name: "BranchLabel", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "ПарольПочты", Type: metadata.FieldTypeString, PII: true},
	}}
	id := uuid.NewString()
	row := map[string]any{"id": id, "ПарольПочты": "SYNTHETIC_MAIL_SECRET"}
	s := &Server{}
	ctx := access.WithReportOutput(auth.ContextWithUser(context.Background(), &auth.User{IsAdmin: true}))
	if got := s.reportOutputRecordLabel(ctx, entity, row); strings.Contains(got, "SYNTHETIC_MAIL_SECRET") {
		t.Fatal("PII field became the report label", got)
	}
	if row["ПарольПочты"] != "SYNTHETIC_MAIL_SECRET" {
		t.Fatal("caller row modified")
	}
	row["Наименование"] = "Север"
	if got := s.reportOutputRecordLabel(ctx, entity, row); got != "Север" {
		t.Fatal("ordinary branch name hidden", got)
	}
}
