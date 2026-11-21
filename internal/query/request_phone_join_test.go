package query_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestRequestPhoneJoinRequiresTrustedReportAndPhysicalAliases(t *testing.T) {
	fields := []metadata.Field{
		{Name: "Label", Type: metadata.FieldTypeString},
		{Name: "ТелефонМобильный", Type: metadata.FieldTypeString, PII: true},
		{Name: "ТелефонДомашний", Type: metadata.FieldTypeString, PII: true},
		{Name: "ТелефонРабочий", Type: metadata.FieldTypeString, PII: true},
	}
	ent := &metadata.Entity{Name: "А_Заявка", Kind: metadata.KindDocument, Fields: fields}
	other := &metadata.Entity{Name: "ДругойДокумент", Kind: metadata.KindDocument, Fields: fields}
	const (
		from = " ИЗ Документ.А_Заявка КАК З1 ВНУТРЕННЕЕ СОЕДИНЕНИЕ Документ.А_Заявка КАК З2 ПО "
		call = "ТЕЛЕФОНЫЗАЯВОКСОВПАДАЮТ(З1, З2)"
	)
	for _, dialect := range []storage.Dialect{storage.SQLiteDialect{}, storage.PgDialect{}} {
		t.Run(dialect.Name(), func(t *testing.T) {
			opts := query.CompileOpts{Dialect: dialect, Entities: []*metadata.Entity{ent, other}}
			valid := "ВЫБРАТЬ З1.Label" + from + call
			_, err := query.Compile(valid, opts)
			require.ErrorContains(t, err, "настроенном встроенном отчёте")

			opts.CompareRequestPhones = true
			res, err := query.Compile(valid, opts)
			require.NoError(t, err)
			require.Contains(t, res.SQL, "NOT IN ('', '(   )   -  -', '(495)   -  -')")
			require.NotContains(t, strings.ToLower(res.SQL), "телефонызаявоксовпадают")
			require.NotContains(t, strings.ToLower(strings.Join(res.ProjectionFields, " ")), "телефон")
			require.Len(t, res.Args, 0)

			for _, text := range []string{
				"ВЫБРАТЬ " + call + " КАК Совпадение" + from + "1 = 1",
				"ВЫБРАТЬ З1.Label" + from + "1 = 1 ГДЕ " + call,
				"ВЫБРАТЬ З1.Label" + from + "ТЕЛЕФОНЫЗАЯВОКСОВПАДАЮТ(З1, З1)",
				strings.Replace(valid, "Документ.А_Заявка КАК З2", "Документ.ДругойДокумент КАК З2", 1),
				strings.Replace(valid, "Документ.А_Заявка КАК З2", "(ВЫБРАТЬ * ИЗ Документ.А_Заявка) КАК З2", 1),
				"ВЫБРАТЬ Т.Label ИЗ (" + valid + ") КАК Т",
			} {
				_, err := query.Compile(text, opts)
				require.ErrorContains(t, err, "два псевдонима А_Заявка", text)
			}
		})
	}
}
