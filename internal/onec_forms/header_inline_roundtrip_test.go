package onec_forms

import (
	"path/filepath"
	"testing"
)

// header_inline — настройка OneBase, в XML её нет; YAML формы обязан
// сохранить её при чтении и записи (обмен формами, конфигуратор).
func TestFormYAMLKeepsHeaderInline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "объекта.form.yaml")
	if err := WriteFormYAML(&IRForm{Name: "ФормаОбъекта", Kind: "object", Entity: "Заявка", HeaderInline: true}, path); err != nil {
		t.Fatal(err)
	}
	back, err := ReadFormYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if !back.HeaderInline || !ToFormModule(back).HeaderInline {
		t.Fatal("header_inline потерян при записи, чтении или переводе в модель формы")
	}
}
