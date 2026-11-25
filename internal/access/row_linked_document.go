package access

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

type linkedDocumentResolver interface {
	HasLinkedDocument(context.Context, *storage.LinkedDocument, uuid.UUID) (bool, error)
}

type compileLinkedDocument func(auth.RowPolicy, *metadata.Entity) (storage.Predicate, error)

type linkedDocumentScopeKey struct{}

func linkedDocumentCompiler(ctx context.Context, user *auth.User, dc DecisionContext, op string, lint bool) compileLinkedDocument {
	return func(p auth.RowPolicy, source *metadata.Entity) (storage.Predicate, error) {
		deny := storage.Predicate{Deny: true}
		src := p.LinkedDocument
		if op != "read" || dc.Lookup == nil || len(p.All) != 0 || len(p.Any) != 0 || p.Not != nil ||
			p.Op != "" || p.SameAs != "" || p.Value.User != "" || p.Value.UserAttr != "" ||
			p.Value.UserCard != nil || p.Value.LiteralSet || p.Value.Literal != nil || p.Value.List != nil {
			return deny, fmt.Errorf("linked_document is a read predicate on a reference field")
		}
		if ctx.Value(linkedDocumentScopeKey{}) != nil {
			return deny, fmt.Errorf("nested linked_document is not supported")
		}
		ref, ok := referenceField(source, strings.TrimSpace(p.Field))
		doc := dc.Lookup.GetEntity(strings.TrimSpace(src.Name))
		if !ok || doc == nil || doc.Kind != metadata.KindDocument || len(src.Fields) == 0 {
			return deny, fmt.Errorf("linked_document requires a reference, a document and its reference paths")
		}
		linked := &storage.LinkedDocument{Document: doc}
		for _, path := range src.Fields {
			var tp *metadata.TablePart
			var field *metadata.Field
			if table, column, isPart := strings.Cut(strings.TrimSpace(path), "."); isPart {
				tp, field = storage.ChoiceTablePartColumn(doc, table, column)
			} else if f, found := referenceField(doc, table); found {
				field = &f
			}
			if field == nil || !strings.EqualFold(field.RefEntity, ref.RefEntity) {
				return deny, fmt.Errorf("linked_document path %q must reference %q", path, ref.RefEntity)
			}
			linked.Links = append(linked.Links, storage.LinkedDocumentPath{TablePart: tp, Field: *field})
		}
		if !lint {
			if dc.LinkedDocuments == nil {
				return deny, fmt.Errorf("linked_document storage is unavailable")
			}
			decision, err := DecideWithContext(context.WithValue(ctx, linkedDocumentScopeKey{}, true), user, string(doc.Kind), doc.Name, "read", doc, dc)
			if err != nil || !decision.Allowed {
				return deny, err
			}
			linked.Scope = decision.Predicate
			linked.Match = func(id uuid.UUID) (bool, error) {
				return dc.LinkedDocuments.HasLinkedDocument(ctx, linked, id)
			}
		}
		return storage.Predicate{Field: ref.Name, LinkedDocument: linked}, nil
	}
}
