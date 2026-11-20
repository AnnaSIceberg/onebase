package metadata

import "strings"

// SavedPhoneMatchSpec configures an indexed match between a saved document
// opened in a managed form and another document.
type SavedPhoneMatchSpec struct {
	LinkCommand    string   `yaml:"link_command,omitempty"`
	SourceFields   []string `yaml:"source_fields"`
	TargetDocument string   `yaml:"target_document"`
	TargetFields   []string `yaml:"target_fields"`
}

func (fm *FormModule) IsSavedPhoneLink(el *FormElement) bool {
	return fm != nil && fm.SavedPhoneMatch != nil && fm.SavedPhoneMatch.LinkCommand != "" &&
		el != nil && el.Kind == FormElementButton && el.Placement == "header_end" &&
		strings.EqualFold(el.Name, fm.SavedPhoneMatch.LinkCommand) &&
		strings.TrimSpace(el.Handlers[FormEventOnClick]) != ""
}
