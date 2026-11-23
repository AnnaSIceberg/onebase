package metadata

// ListFilterAccountPriority orders the choices of one reference filter in
// selected document lists. The target catalog links to a platform account.
type ListFilterAccountPriority struct {
	Documents    []string `yaml:"documents"`
	Field        string   `yaml:"field"`
	AccountField string   `yaml:"account_field"`
	NameField    string   `yaml:"name_field"`
	Roles        []string `yaml:"roles"`
}
