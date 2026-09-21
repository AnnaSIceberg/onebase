package storage_test

// Регистронезависимый lookup предопределённых (#1622): DSL не различает
// регистр, поэтому имя приезжает в нижнем регистре и обязано находить запись.
// Матрица: бинарное сравнение SQLite не сворачивает кириллицу, PostgreSQL —
// имеет собственные правила; EqualFold в Go выравнивает оба диалекта.

import (
	"context"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestGetPredefinedIDCaseFoldMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		catalog := &metadata.Entity{
			Name:   "Валюты",
			Kind:   metadata.KindCatalog,
			Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
			Predefined: []*metadata.PredefinedItem{{
				Name: "Рубль", Fields: map[string]any{"Наименование": "Российский рубль"},
			}},
		}
		if err := db.Migrate(ctx, []*metadata.Entity{catalog}); err != nil {
			t.Fatal(err)
		}
		exact, err := db.GetPredefinedID(ctx, catalog.Name, "Рубль")
		if err != nil {
			t.Fatalf("точное имя не нашлось: %v", err)
		}
		for _, q := range []string{"рубль", "РУБЛЬ", "Рубль"} {
			id, err := db.GetPredefinedID(ctx, catalog.Name, q)
			if err != nil {
				t.Fatalf("%q не нашлось: %v", q, err)
			}
			if id != exact {
				t.Fatalf("%q дал другой id: %s != %s", q, id, exact)
			}
		}
	})
}

func TestGetPredefinedIDCaseFoldAmbiguousMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		catalog := &metadata.Entity{
			Name:   "Валюты",
			Kind:   metadata.KindCatalog,
			Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
			Predefined: []*metadata.PredefinedItem{
				{Name: "Рубль", Fields: map[string]any{"Наименование": "РФ"}},
				{Name: "рубль", Fields: map[string]any{"Наименование": "Беларусь"}},
			},
		}
		if err := db.Migrate(ctx, []*metadata.Entity{catalog}); err != nil {
			t.Fatal(err)
		}
		_, err := db.GetPredefinedID(ctx, catalog.Name, "РУБЛЬ")
		if err == nil {
			t.Fatal("неоднозначное имя должно давать ошибку")
		}
		if !strings.Contains(err.Error(), "неоднозначно") {
			t.Fatalf("ожидалась неоднозначность, получили: %v", err)
		}
	})
}
