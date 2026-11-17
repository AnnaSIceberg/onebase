package ui

import (
	"context"
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
	if got := s.reportOutputRecordLabel(access.WithReportOutput(ctx), entity, row); got != id {
		t.Fatal("protected report label must retain only its UUID", got)
	}
	if got := s.maskedRecordLabel(ctx, entity, row); got != "SYNTHETIC_HOUSE_FLAT" {
		t.Fatal("ordinary card label changed", got)
	}
}
