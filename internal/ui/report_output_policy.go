package ui

import (
	"context"
	"fmt"

	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
)

// reportOutputMaskPlan is the shared mandatory boundary for native report
// execution and filters. Ordinary card/widget/query callers keep queryMaskPlan.
func (s *Server) reportOutputMaskPlan(ctx context.Context, compiled query.Result) access.QueryMaskPlan {
	return access.ReportOutputQueryMaskPlan(auth.UserFromContext(ctx), compiled, s.sourceMeta)
}

// A presentation built from protected report fields keeps only its technical
// identity; an ordinary label (branch name, document number) stays readable.
func (s *Server) reportOutputRecordLabel(ctx context.Context, entity *metadata.Entity, row map[string]any) string {
	if access.ReportOutputLabelProtected(entity) {
		return fmt.Sprint(row["id"])
	}
	// Поля обязательной политики ПДн убираются из копии строки до подписи при
	// любой роли: иначе при пустом Наименовании RowLabel взял бы следующий
	// строковый кандидат — pii-поле, которое карточная политика админу
	// оставляет открытым.
	clean := make(map[string]any, len(row))
	for key, value := range row {
		clean[key] = value
	}
	s.maskRecordWithDecisions(access.ReportOutputDecisions(entity), clean)
	return s.maskedRecordLabel(ctx, entity, clean)
}
