package access

import (
	"testing"

	"github.com/ivantit66/onebase/internal/storage"
)

// Всё, что маска подставляет вместо значения, страховка записи обязана
// узнать: иначе заглушку можно было бы записать в другой объект.
func TestMaskValueOutputIsRecognizedAsPlaceholder(t *testing.T) {
	for _, tc := range []struct {
		strategy string
		keep     int
		value    string
	}{
		{FieldMaskAll, 0, "+79161234455"},
		{FieldMaskTail, 4, "+79161234455"},
		{FieldMaskTail, 0, "+79161234455"},
		{FieldMaskTail, 20, "1234"},
		{FieldMaskTail, 4, "51234"},
		{FieldMaskTail, 4, "1 234"},
		{FieldMaskTail, 4, "1\t234"},
		{FieldMaskTail, 4, "1\n234"},
		{FieldMaskTail, 4, "1\u00a0234"},
		{FieldMaskTail, 4, "1\u2003234"},
		{FieldMaskTail, 4, "12 234"},
		{FieldMaskTail, 0, "1"},
		{FieldMaskCity, 0, "Москва, ул. Ленина, 1"},
		{"неизвестная", 0, "секрет"},
	} {
		masked := MaskValue(tc.strategy, tc.keep, tc.value)
		if !storage.IsMaskPlaceholder(masked) {
			t.Errorf("%s/%d: %q не распознана как заглушка", tc.strategy, tc.keep, masked)
		}
	}
	for _, plain := range []string{"+79161234455", "Москва", "текст • с точкой", "• Позвонить клиенту", "•\tОтправить письмо", ""} {
		if storage.IsMaskPlaceholder(plain) {
			t.Errorf("%q ошибочно принята за заглушку", plain)
		}
	}
}
