package metadata

// SavedPhoneMatchSpec configures an indexed match between a saved document
// opened in a managed form and another document.
type SavedPhoneMatchSpec struct {
	SourceFields   []string `yaml:"source_fields"`
	TargetDocument string   `yaml:"target_document"`
	TargetFields   []string `yaml:"target_fields"`
}
