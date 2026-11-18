package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// Проверяем реальные писатели и повторно читаем БД: распознаватель сам по
// себе не доказывает, что проверялось именно значение, ушедшее в SQL.
func TestMaskBackstopCanonicalAndBulletTextMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		entity := &metadata.Entity{Name: "MaskNote", Kind: metadata.KindCatalog,
			Fields:     []metadata.Field{{Name: "Comment", Type: metadata.FieldTypeString}},
			TableParts: []metadata.TablePart{{Name: "Lines", Fields: []metadata.Field{{Name: "Comment", Type: metadata.FieldTypeString}}}},
		}
		if err := db.Migrate(ctx, []*metadata.Entity{entity}); err != nil {
			t.Fatal(err)
		}
		tp := entity.TableParts[0]
		for _, writer := range []string{"upsert", "versioned", "tablepart"} {
			t.Run(writer, func(t *testing.T) {
				for _, tc := range []struct {
					name   string
					fields map[string]any
					want   string
					reject bool
				}{
					{name: "bullet list", fields: map[string]any{"Comment": "• Позвонить клиенту"}, want: "• Позвонить клиенту"},
					{name: "tabbed list", fields: map[string]any{"Comment": "•\tПозвонить клиенту\n• Отправить письмо"}, want: "•\tПозвонить клиенту\n• Отправить письмо"},
					{name: "interior bullet", fields: map[string]any{"Comment": "текст • с точкой"}, want: "текст • с точкой"},
					{name: "nil canonical lower mask", fields: map[string]any{"Comment": nil, "comment": storage.MaskFixed}, reject: true},
					{name: "nil canonical folded mask", fields: map[string]any{"Comment": nil, "COMMENT": storage.MaskFixed}, reject: true},
					{name: "nil lower canonical mask", fields: map[string]any{"Comment": storage.MaskFixed, "comment": nil}, reject: true},
					{name: "canonical wins over lower", fields: map[string]any{"Comment": "настоящее", "comment": storage.MaskFixed}, want: "настоящее"},
					{name: "nil canonical lower text", fields: map[string]any{"Comment": nil, "comment": "настоящее"}, want: "настоящее"},
					{name: "folded aliases choose mask consistently", fields: map[string]any{"COMMENT": storage.MaskFixed, "CoMmEnT": "настоящее"}, reject: true},
					{name: "folded aliases choose text consistently", fields: map[string]any{"COMMENT": "настоящее", "CoMmEnT": storage.MaskFixed}, want: "настоящее"},
					{name: "single masked rune", fields: map[string]any{"Comment": "•"}, reject: true},
					{name: "single masked prefix", fields: map[string]any{"Comment": "•1234"}, reject: true},
					{name: "mask all", fields: map[string]any{"Comment": storage.MaskFixed}, reject: true},
					{name: "mask tail", fields: map[string]any{"Comment": "••••1122"}, reject: true},
					{name: "mask city", fields: map[string]any{"Comment": "Москва, " + storage.MaskFixed}, reject: true},
				} {
					t.Run(tc.name, func(t *testing.T) {
						id := uuid.New()
						if err := db.Upsert(ctx, entity.Name, id, map[string]any{"Comment": "исходное"}, entity); err != nil {
							t.Fatal(err)
						}
						if writer == "tablepart" {
							if err := db.UpsertTablePartRows(ctx, entity.Name, tp.Name, id, []map[string]any{{"Comment": "исходное"}}, tp); err != nil {
								t.Fatal(err)
							}
						}
						var err error
						switch writer {
						case "upsert":
							err = db.Upsert(ctx, entity.Name, id, tc.fields, entity)
						case "versioned":
							version := int64(1)
							err = db.UpsertVersioned(ctx, entity.Name, id, tc.fields, entity, &version)
						case "tablepart":
							err = db.UpsertTablePartRows(ctx, entity.Name, tp.Name, id, []map[string]any{tc.fields}, tp)
						}
						if tc.reject {
							if !errors.Is(err, storage.ErrMaskedValueWrite) {
								t.Errorf("want masked-value rejection, got %v", err)
							}
						} else if err != nil {
							t.Errorf("ordinary text rejected: %v", err)
						}
						want := tc.want
						if tc.reject {
							want = "исходное"
						}
						var row map[string]any
						if writer == "tablepart" {
							rows, readErr := db.GetTablePartRows(ctx, entity.Name, tp.Name, id, tp)
							if readErr != nil {
								t.Fatal(readErr)
							}
							if len(rows) != 1 {
								t.Fatalf("rows = %v", rows)
							}
							row = rows[0]
						} else {
							var readErr error
							row, readErr = db.GetByID(ctx, entity.Name, id, entity)
							if readErr != nil {
								t.Fatal(readErr)
							}
						}
						got, ok := row["Comment"]
						if !ok {
							got = row["comment"]
						}
						if got != want {
							t.Errorf("stored Comment = %v, want %q", got, want)
						}
					})
				}
			})
		}
	})
}
