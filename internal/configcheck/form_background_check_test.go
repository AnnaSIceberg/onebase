package configcheck

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

func TestCheckFormBackground_InvalidColorWarns(t *testing.T) {
	warns := CheckFormBackground(projWithElement(&metadata.FormElement{
		Kind:       metadata.FormElementGroupBox,
		Name:       "ГруппаДействия",
		Background: "зеленый-в-клеточку",
	}))
	if len(warns) != 1 {
		t.Fatalf("ожидалось 1 предупреждение, получили %d: %+v", len(warns), warns)
	}
	if warns[0].Code != "form.background-color" {
		t.Errorf("Code = %q, ожидался form.background-color", warns[0].Code)
	}
	if !strings.Contains(warns[0].SuggestedFix, "hex") {
		t.Errorf("подсказка не называет допустимые форматы: %q", warns[0].SuggestedFix)
	}
}

func TestCheckFormBackground_NonGroupKindWarns(t *testing.T) {
	warns := CheckFormBackground(projWithElement(&metadata.FormElement{
		Kind:       metadata.FormElementField,
		Name:       "ПолеНаименование",
		DataPath:   "Объект.Наименование",
		Background: "#e8f5e9",
	}))
	if len(warns) != 1 {
		t.Fatalf("ожидалось 1 предупреждение, получили %d: %+v", len(warns), warns)
	}
	if warns[0].Code != "form.background-kind" {
		t.Errorf("Code = %q, ожидался form.background-kind", warns[0].Code)
	}
}

func TestCheckFormBackground_ValidColorSilent(t *testing.T) {
	warns := CheckFormBackground(projWithElement(&metadata.FormElement{
		Kind:       metadata.FormElementGroupBox,
		Name:       "ГруппаДействия",
		Background: "#e8f5e9",
	}))
	if len(warns) != 0 {
		t.Fatalf("валидный фон не должен предупреждать: %+v", warns)
	}
}
