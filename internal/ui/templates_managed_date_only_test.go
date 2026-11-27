package ui

import (
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

func TestFormDateOnlyInLegacyAndEnhancedInterfaces(t *testing.T) {
	enhanced := &metadata.FormPresentation{DateOnly: true}
	tests := []struct {
		name    string
		element metadata.FormElement
		legacy  bool
		want    bool
	}{
		{name: "enhanced opt-in", element: metadata.FormElement{Enhanced: enhanced}, want: true},
		{name: "enhanced opt-in is ignored in legacy UI", element: metadata.FormElement{Enhanced: enhanced}, legacy: true},
		{name: "explicit opt-in in legacy UI", element: metadata.FormElement{DateOnly: true}, legacy: true, want: true},
		{name: "explicit opt-in in enhanced UI", element: metadata.FormElement{DateOnly: true}, want: true},
		{name: "date and time by default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formDateOnly(&tt.element, tt.legacy); got != tt.want {
				t.Fatalf("formDateOnly = %t, want %t", got, tt.want)
			}
		})
	}
}
